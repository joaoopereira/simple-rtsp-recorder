package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaoopereira/simple-rtsp-recorder/internal/config"
	"github.com/joaoopereira/simple-rtsp-recorder/internal/logging"
	"github.com/joaoopereira/simple-rtsp-recorder/internal/recorder"
	"github.com/joaoopereira/simple-rtsp-recorder/internal/updater"
	"github.com/joaoopereira/simple-rtsp-recorder/web"
)

func newTestServer(t *testing.T) (*httptest.Server, *config.Store) {
	t.Helper()
	dir := t.TempDir()
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := store.Get()
	cfg.Camera = config.Camera{Host: "cam", Port: 554, Path: "live", User: "admin", Password: "secret"}
	if err := store.Update(cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.SetAdminPassword(adminPassword); err != nil {
		t.Fatal(err)
	}
	log := logging.Discard()
	rec := recorder.New("ffmpeg", func() (string, string) {
		return store.Get().Camera.StreamURL(), store.RecordingsDir()
	}, filepath.Join(dir, "state"), log.Logger)
	upd := updater.New(updater.Options{Current: "dev", Log: log.Logger, Enabled: func() bool { return false }})
	s := &Server{Version: "test", Store: store, Recorder: rec, Updater: upd, Log: log, Static: web.FS}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return srv, store
}

const adminPassword = "correct horse"

// client keeps cookies, like a browser.
func client() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

// adminClient returns a client logged in as admin.
func adminClient(t *testing.T, srv *httptest.Server) *http.Client {
	t.Helper()
	c := client()
	if resp, body := doWith(t, c, "POST", srv.URL+"/api/admin/login", `{"password":"`+adminPassword+`"}`); resp.StatusCode != 200 {
		t.Fatalf("login = %d %s", resp.StatusCode, body)
	}
	return c
}

func do(t *testing.T, method, url, body string) (*http.Response, string) {
	t.Helper()
	return doWith(t, http.DefaultClient, method, url, body)
}

func doWith(t *testing.T, c *http.Client, method, url, body string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return resp, b.String()
}

func TestSettingsPasswordIsWriteOnly(t *testing.T) {
	srv, store := newTestServer(t)
	c := adminClient(t, srv)
	do := func(t *testing.T, method, url, body string) (*http.Response, string) {
		return doWith(t, c, method, url, body)
	}

	_, body := do(t, "GET", srv.URL+"/api/settings", "")
	if strings.Contains(body, "secret") || strings.Contains(body, "pbkdf2") || !strings.Contains(body, `"hasPassword":true`) {
		t.Fatalf("GET /api/settings leaked or hid password state: %s", body)
	}

	// Saving without a password field keeps the stored one.
	var dto map[string]any
	json.Unmarshal([]byte(body), &dto)
	dto["title"] = "Futsal"
	payload, _ := json.Marshal(dto)
	if resp, body := do(t, "PUT", srv.URL+"/api/settings", string(payload)); resp.StatusCode != 200 {
		t.Fatalf("PUT = %d %s", resp.StatusCode, body)
	}
	if got := store.Get(); got.Camera.Password != "secret" || got.Title != "Futsal" || got.AdminPasswordHash == "" {
		t.Fatalf("after PUT: %+v", got)
	}

	// An explicit password replaces it.
	dto["camera"].(map[string]any)["password"] = "new"
	payload, _ = json.Marshal(dto)
	do(t, "PUT", srv.URL+"/api/settings", string(payload))
	if got := store.Get().Camera.Password; got != "new" {
		t.Fatalf("password = %q, want new", got)
	}

	// Invalid settings are rejected.
	dto["camera"].(map[string]any)["port"] = 70000
	payload, _ = json.Marshal(dto)
	if resp, _ := do(t, "PUT", srv.URL+"/api/settings", string(payload)); resp.StatusCode != 400 {
		t.Fatalf("invalid port accepted: %d", resp.StatusCode)
	}
}

func TestRecordingsEndpoints(t *testing.T) {
	srv, store := newTestServer(t)
	dir := store.RecordingsDir()
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "2026-10-29_150000.mp4"), []byte("video"), 0o644)
	os.WriteFile(filepath.Join(store.BaseDir(), "secret.mp4"), []byte("outside"), 0o644)

	_, body := do(t, "GET", srv.URL+"/api/recordings", "")
	if !strings.Contains(body, "2026-10-29_150000.mp4") {
		t.Fatalf("list = %s", body)
	}

	resp, body := do(t, "GET", srv.URL+"/recordings/2026-10-29_150000.mp4?download=1", "")
	if resp.StatusCode != 200 || body != "video" || !strings.Contains(resp.Header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("download = %d %q %v", resp.StatusCode, body, resp.Header)
	}

	for _, path := range []string{"/recordings/..%2Fsecret.mp4", "/api/recordings/..%2Fsecret.mp4"} {
		method := "GET"
		if strings.HasPrefix(path, "/api") {
			method = "DELETE"
		}
		if resp, _ := do(t, method, srv.URL+path, ""); resp.StatusCode != 400 {
			t.Errorf("%s %s = %d, want 400", method, path, resp.StatusCode)
		}
	}
	if _, err := os.Stat(filepath.Join(store.BaseDir(), "secret.mp4")); err != nil {
		t.Fatal("file outside recordings dir was touched")
	}

	if resp, _ := do(t, "DELETE", srv.URL+"/api/recordings/2026-10-29_150000.mp4", ""); resp.StatusCode != 204 {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	if resp, _ := do(t, "DELETE", srv.URL+"/api/recordings/2026-10-29_150000.mp4", ""); resp.StatusCode != 404 {
		t.Fatalf("second delete = %d, want 404", resp.StatusCode)
	}
}

func TestStatusAndStatic(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, body := do(t, "GET", srv.URL+"/api/status", "")
	if resp.StatusCode != 200 || !strings.Contains(body, `"state":"idle"`) || !strings.Contains(body, `"configured":true`) {
		t.Fatalf("status = %d %s", resp.StatusCode, body)
	}
	resp, body = do(t, "GET", srv.URL+"/", "")
	if resp.StatusCode != 200 || !strings.Contains(body, "<title>") {
		t.Fatalf("index = %d", resp.StatusCode)
	}
	if resp, _ := do(t, "POST", srv.URL+"/api/recording/stop", ""); resp.StatusCode != 409 {
		t.Fatalf("stop while idle = %d, want 409", resp.StatusCode)
	}
}

func TestSettingsRequireAdmin(t *testing.T) {
	srv, store := newTestServer(t)
	anon := client()
	for _, r := range []struct{ method, path string }{
		{"GET", "/api/settings"}, {"PUT", "/api/settings"}, {"POST", "/api/settings/test"}, {"PUT", "/api/admin/password"},
		{"GET", "/api/logs"},
	} {
		if resp, _ := doWith(t, anon, r.method, srv.URL+r.path, `{"password":"hijacked!"}`); resp.StatusCode != 401 {
			t.Errorf("%s %s without login = %d, want 401", r.method, r.path, resp.StatusCode)
		}
	}
	if store.CheckAdminPassword(adminPassword) != nil {
		t.Fatal("admin password changed without login")
	}

	// A forged cookie is rejected.
	req, _ := http.NewRequest("GET", srv.URL+"/api/settings", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "99999999999.deadbeef"})
	if resp, _ := http.DefaultClient.Do(req); resp.StatusCode != 401 {
		t.Errorf("forged cookie = %d, want 401", resp.StatusCode)
	}

	if _, body := doWith(t, anon, "GET", srv.URL+"/api/admin", ""); body != `{"loggedIn":false,"passwordSet":true}`+"\n" {
		t.Errorf("admin status = %s", body)
	}
}

func TestLoginLockoutAndPasswordChange(t *testing.T) {
	srv, _ := newTestServer(t)
	c := client()
	for i := 0; i < maxFailures; i++ {
		if resp, _ := doWith(t, c, "POST", srv.URL+"/api/admin/login", `{"password":"wrong"}`); resp.StatusCode != 401 {
			t.Fatalf("wrong password = %d, want 401", resp.StatusCode)
		}
	}
	if resp, _ := doWith(t, c, "POST", srv.URL+"/api/admin/login", `{"password":"`+adminPassword+`"}`); resp.StatusCode != 429 {
		t.Fatalf("login after lockout = %d, want 429", resp.StatusCode)
	}

	// Changing the password logs out other sessions but keeps this one.
	srv2, _ := newTestServer(t)
	a, b := adminClient(t, srv2), adminClient(t, srv2)
	if resp, body := doWith(t, a, "PUT", srv2.URL+"/api/admin/password", `{"password":"short"}`); resp.StatusCode != 400 {
		t.Fatalf("weak password = %d %s, want 400", resp.StatusCode, body)
	}
	if resp, _ := doWith(t, a, "PUT", srv2.URL+"/api/admin/password", `{"password":"a new password"}`); resp.StatusCode != 204 {
		t.Fatalf("change password = %d", resp.StatusCode)
	}
	if resp, _ := doWith(t, a, "GET", srv2.URL+"/api/settings", ""); resp.StatusCode != 200 {
		t.Errorf("own session after change = %d, want 200", resp.StatusCode)
	}
	if resp, _ := doWith(t, b, "GET", srv2.URL+"/api/settings", ""); resp.StatusCode != 401 {
		t.Errorf("other session after change = %d, want 401", resp.StatusCode)
	}

	// Logout ends the session.
	doWith(t, a, "POST", srv2.URL+"/api/admin/logout", "")
	if resp, _ := doWith(t, a, "GET", srv2.URL+"/api/settings", ""); resp.StatusCode != 401 {
		t.Errorf("after logout = %d, want 401", resp.StatusCode)
	}
}

func TestSessionSurvivesRestart(t *testing.T) {
	srv, store := newTestServer(t)
	c := adminClient(t, srv)
	resp, _ := doWith(t, c, "GET", srv.URL+"/api/admin", "")
	var renewed *http.Cookie
	for _, ck := range resp.Cookies() {
		if ck.Name == sessionCookie {
			renewed = ck
		}
	}
	if renewed == nil || renewed.MaxAge != int(sessionDuration.Seconds()) {
		t.Errorf("session not renewed for 7 days: %+v", renewed)
	}

	// A restart (or update) builds a new handler; the key is read back from disk.
	restarted := &Server{Version: "test", Store: store, Log: logging.Discard(), Static: web.FS}
	srv2 := httptest.NewServer(restarted.Handler())
	t.Cleanup(srv2.Close)
	if resp, body := doWith(t, c, "GET", srv2.URL+"/api/logs", ""); resp.StatusCode != 200 {
		t.Fatalf("logs after restart = %d %s, want 200", resp.StatusCode, body)
	}
	if info, err := os.Stat(filepath.Join(store.BaseDir(), sessionKeyFile)); err != nil || info.Size() == 0 {
		t.Fatalf("session key not saved: %v", err)
	}
}

func TestLoginWithoutPasswordSet(t *testing.T) {
	srv, store := newTestServer(t)
	cfg := store.Get()
	cfg.AdminPasswordHash = ""
	store.Update(cfg)
	if resp, body := do(t, "POST", srv.URL+"/api/admin/login", `{"password":""}`); resp.StatusCode != 403 || !strings.Contains(body, "set-admin-password") {
		t.Fatalf("login with no password set = %d %s", resp.StatusCode, body)
	}
	if resp, _ := do(t, "GET", srv.URL+"/api/settings", ""); resp.StatusCode != 401 {
		t.Fatalf("settings with no password set = %d, want 401", resp.StatusCode)
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Cf-Connecting-Ip", "1.2.3.4")
	r.RemoteAddr = "127.0.0.1:5000"
	if got := clientIP(r); got != "1.2.3.4" {
		t.Errorf("via tunnel = %q", got)
	}
	r.RemoteAddr = "192.168.1.50:5000"
	if got := clientIP(r); got != "192.168.1.50" {
		t.Errorf("LAN client with forged header = %q", got)
	}
}

// The page must reference every asset by a content hash so that no cache
// (Cloudflare kept serving the 1.x style.css) can serve another version.
func TestStaticAssetsAreVersioned(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, page := do(t, "GET", srv.URL+"/", "")
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("index Cache-Control = %q, want no-store", cc)
	}
	for _, name := range assets {
		data, err := fs.ReadFile(web.FS, name)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		url := name + "?v=" + hex.EncodeToString(sum[:6])
		if !strings.Contains(page, `"`+url+`"`) {
			t.Errorf("index.html does not reference %s", url)
		}
		if strings.Contains(page, `"`+name+`"`) {
			t.Errorf("index.html still has an unversioned reference to %s", name)
		}
		resp, body := do(t, "GET", srv.URL+"/"+url, "")
		if resp.StatusCode != 200 || body != string(data) {
			t.Errorf("GET %s = %d", url, resp.StatusCode)
		}
		if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
			t.Errorf("%s Cache-Control = %q, want immutable", url, cc)
		}
		if resp, _ := do(t, "GET", srv.URL+"/"+name+"?v=stale", ""); resp.Header.Get("Cache-Control") != "no-cache" {
			t.Errorf("%s with a stale hash must not be cached long", name)
		}
	}
	if resp, _ := do(t, "GET", srv.URL+"/style.css", ""); !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/css") {
		t.Errorf("style.css Content-Type = %q", resp.Header.Get("Content-Type"))
	}
}
