package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joaoopereira/simple-rtsp-recorder/internal/config"
)

const (
	sessionCookie   = "srr_admin"
	sessionDuration = 12 * time.Hour
	maxFailures     = 5
	lockoutDuration = 15 * time.Minute
)

// auth issues and checks admin sessions. A session is a cookie
// "<expiry>.<hmac>" signed with a per-process random key, so sessions end
// when the app restarts (e.g. after an update). The signature also covers
// the password hash: changing the password logs everyone out.
type auth struct {
	key []byte

	mu       sync.Mutex
	failures map[string]*failure
}

type failure struct {
	count int
	first time.Time
}

func newAuth() *auth {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return &auth{key: key, failures: map[string]*failure{}}
}

func (a *auth) sign(expiry int64, passwordHash string) string {
	m := hmac.New(sha256.New, a.key)
	m.Write([]byte(strconv.FormatInt(expiry, 10) + "|" + passwordHash))
	return hex.EncodeToString(m.Sum(nil))
}

func (a *auth) valid(r *http.Request, passwordHash string) bool {
	if passwordHash == "" {
		return false
	}
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	exp, sig, ok := strings.Cut(c.Value, ".")
	expiry, err := strconv.ParseInt(exp, 10, 64)
	if !ok || err != nil || time.Now().Unix() > expiry {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(a.sign(expiry, passwordHash)))
}

func (a *auth) setSession(w http.ResponseWriter, r *http.Request, passwordHash string) {
	expiry := time.Now().Add(sessionDuration).Unix()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    strconv.FormatInt(expiry, 10) + "." + a.sign(expiry, passwordHash),
		Path:     "/",
		MaxAge:   int(sessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteStrictMode,
	})
}

func clearSession(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", MaxAge: -1, HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteStrictMode})
}

// isHTTPS is true behind the Cloudflare Tunnel, which talks plain HTTP to us.
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// clientIP identifies the client for the login lockout. Behind the tunnel
// every request comes from localhost (cloudflared), so Cloudflare's header is
// used then — and only then, so LAN clients can't fake it.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if cf := r.Header.Get("Cf-Connecting-Ip"); cf != "" {
			return cf
		}
	}
	return host
}

// locked reports whether an IP has too many recent failed logins.
func (a *auth) locked(ip string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.failures[ip]
	if f == nil {
		return false
	}
	if time.Since(f.first) > lockoutDuration {
		delete(a.failures, ip)
		return false
	}
	return f.count >= maxFailures
}

func (a *auth) fail(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.failures[ip]
	if f == nil || time.Since(f.first) > lockoutDuration {
		f = &failure{first: time.Now()}
		a.failures[ip] = f
	}
	f.count++
}

func (a *auth) succeed(ip string) {
	a.mu.Lock()
	delete(a.failures, ip)
	a.mu.Unlock()
}

var errAdminRequired = errors.New("é preciso iniciar sessão como administrador")

// requireAdmin wraps handlers that only the admin may use.
func (s *Server) requireAdmin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.valid(r, s.Store.Get().AdminPasswordHash) {
			writeError(w, http.StatusUnauthorized, errAdminRequired)
			return
		}
		h(w, r)
	}
}

func (s *Server) adminStatus(w http.ResponseWriter, r *http.Request) {
	hash := s.Store.Get().AdminPasswordHash
	writeJSON(w, http.StatusOK, map[string]bool{
		"passwordSet": hash != "",
		"loggedIn":    s.auth.valid(r, hash),
	})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if s.auth.locked(ip) {
		writeError(w, http.StatusTooManyRequests, errors.New("demasiadas tentativas falhadas; tente daqui a 15 minutos"))
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("pedido inválido"))
		return
	}
	if err := s.Store.CheckAdminPassword(body.Password); err != nil {
		if errors.Is(err, config.ErrNoAdminPassword) {
			writeError(w, http.StatusForbidden, errors.New("ainda não foi definida a palavra-passe de administrador; defina-a no PC com: simple-rtsp-recorder.exe set-admin-password"))
			return
		}
		s.auth.fail(ip)
		s.Log.Warn("admin login failed", "ip", ip)
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	s.auth.succeed(ip)
	s.auth.setSession(w, r, s.Store.Get().AdminPasswordHash)
	s.Log.Info("admin logged in", "ip", ip)
	writeJSON(w, http.StatusOK, map[string]bool{"passwordSet": true, "loggedIn": true})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	clearSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("pedido inválido"))
		return
	}
	if err := s.Store.SetAdminPassword(body.Password); err != nil {
		code := http.StatusInternalServerError
		if errors.Is(err, config.ErrWeakPassword) {
			code = http.StatusBadRequest
		}
		writeError(w, code, err)
		return
	}
	// Other sessions are now invalid; keep this one logged in.
	s.auth.setSession(w, r, s.Store.Get().AdminPasswordHash)
	s.Log.Info("admin password changed", "ip", clientIP(r))
	w.WriteHeader(http.StatusNoContent)
}
