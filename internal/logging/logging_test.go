package logging

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// brokenWriter mimics stdout in a Windows service, which has no console.
type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("invalid handle") }

func TestLogsReachFileAndMemoryWhenStdoutFails(t *testing.T) {
	dir := t.TempDir()
	log, err := newLogger(dir, brokenWriter{})
	if err != nil {
		t.Fatal(err)
	}
	log.Info("hello from the service")

	recent := log.Recent()
	if len(recent) != 1 || !strings.Contains(recent[0], "hello from the service") {
		t.Fatalf("Recent() = %q, want the logged line", recent)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "recorder.log"))
	if !strings.Contains(string(data), "hello from the service") {
		t.Fatalf("recorder.log = %q", data)
	}
}

func TestRecentIsNeverNil(t *testing.T) {
	// Encoded as JSON for /api/logs: must be [] rather than null.
	if got := Discard().Recent(); got == nil {
		t.Fatal("Recent() returned nil")
	}
}
