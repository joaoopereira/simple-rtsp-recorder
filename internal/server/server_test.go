package server

import (
	"encoding/json"
	"net/http"
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

func do(t *testing.T, method, url, body string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	resp, err := http.DefaultClient.Do(req)
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

	_, body := do(t, "GET", srv.URL+"/api/settings", "")
	if strings.Contains(body, "secret") || !strings.Contains(body, `"hasPassword":true`) {
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
	if got := store.Get(); got.Camera.Password != "secret" || got.Title != "Futsal" {
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
