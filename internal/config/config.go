// Package config loads, validates and persists the application settings
// (config.json next to the executable).
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FileName is the settings file, stored in the base directory.
const FileName = "config.json"

// Camera describes how to reach the RTSP stream.
type Camera struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Path     string `json:"path"`
	User     string `json:"user"`
	Password string `json:"password"`
	// URL, when set, is used as-is instead of host/port/path/user/password.
	// Meant for unusual cameras and for local testing.
	URL string `json:"url,omitempty"`
}

// Config is the full set of settings.
type Config struct {
	Title         string `json:"title"`
	Port          int    `json:"port"`
	Camera        Camera `json:"camera"`
	RecordingsDir string `json:"recordingsDir"`
	// RetentionDays deletes recordings older than this many days. 0 keeps everything.
	RetentionDays int  `json:"retentionDays"`
	AutoUpdate    bool `json:"autoUpdate"`
	// FFmpegPath overrides the ffmpeg binary. Empty means ffmpeg(.exe) next to
	// the executable, falling back to the one on PATH.
	FFmpegPath string `json:"ffmpegPath,omitempty"`
	// AdminPasswordHash protects the settings (see HashPassword). Set with
	// `simple-rtsp-recorder.exe set-admin-password` or from the UI once logged in.
	AdminPasswordHash string `json:"adminPasswordHash,omitempty"`
}

// Default returns the settings used when no config file exists.
func Default() Config {
	return Config{
		Title:         "Simple RTSP Recorder",
		Port:          8080,
		Camera:        Camera{Port: 554},
		RecordingsDir: "recordings",
		AutoUpdate:    true,
	}
}

// StreamURL builds the RTSP URL including credentials.
func (c Camera) StreamURL() string {
	if c.URL != "" {
		return c.URL
	}
	if c.Host == "" {
		return ""
	}
	u := url.URL{
		Scheme: "rtsp",
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   "/" + strings.TrimPrefix(c.Path, "/"),
	}
	if c.User != "" {
		u.User = url.UserPassword(c.User, c.Password)
	}
	return u.String()
}

// Configured reports whether there is enough information to connect.
func (c Camera) Configured() bool { return c.StreamURL() != "" }

// Redact removes credentials from a URL so it can be logged.
func Redact(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	u.User = url.User("***")
	return u.String()
}

// Validate checks the settings and fills in defaults for zero values.
func (c *Config) Validate() error {
	d := Default()
	if strings.TrimSpace(c.Title) == "" {
		c.Title = d.Title
	}
	if c.Port == 0 {
		c.Port = d.Port
	}
	if c.Camera.Port == 0 {
		c.Camera.Port = d.Camera.Port
	}
	if strings.TrimSpace(c.RecordingsDir) == "" {
		c.RecordingsDir = d.RecordingsDir
	}
	c.Camera.Host = strings.TrimSpace(c.Camera.Host)
	c.Camera.Path = strings.TrimSpace(c.Camera.Path)
	switch {
	case c.Port < 1 || c.Port > 65535:
		return errors.New("porta HTTP inválida")
	case c.Camera.Port < 1 || c.Camera.Port > 65535:
		return errors.New("porta da câmara inválida")
	case c.RetentionDays < 0:
		return errors.New("dias de retenção não pode ser negativo")
	case strings.ContainsAny(c.Camera.Host, "/@ "):
		return errors.New("endereço da câmara inválido (indique só o IP ou nome)")
	}
	return nil
}

// Store gives concurrent access to the settings and persists changes. Edits
// made to config.json by other processes (e.g. set-admin-password while the
// service runs, or by hand) are picked up automatically.
type Store struct {
	mu      sync.Mutex
	path    string
	baseDir string
	cfg     Config
	modTime time.Time
}

// Open loads config.json from baseDir. When it does not exist the settings are
// migrated from a legacy prod.env (or .env) if present, otherwise defaults are
// used, and the result is written to disk.
func Open(baseDir string) (*Store, error) {
	s := &Store{path: filepath.Join(baseDir, FileName), baseDir: baseDir}
	cfg, modTime, err := s.read()
	switch {
	case err == nil:
		s.cfg, s.modTime = cfg, modTime
		return s, nil
	case errors.Is(err, os.ErrNotExist):
		cfg, _, err := migrateLegacy(baseDir)
		if err != nil {
			return nil, err
		}
		s.cfg = cfg
		return s, s.save(cfg)
	default:
		return nil, err
	}
}

// Get returns a copy of the current settings.
func (s *Store) Get() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, err := os.Stat(s.path); err == nil && !st.ModTime().Equal(s.modTime) {
		// Changed on disk by someone else. Keep the old settings if the
		// new file is invalid (e.g. half-edited by hand).
		if cfg, modTime, err := s.read(); err == nil {
			s.cfg, s.modTime = cfg, modTime
		}
	}
	return s.cfg
}

func (s *Store) read() (Config, time.Time, error) {
	cfg := Default()
	st, err := os.Stat(s.path)
	if err != nil {
		return cfg, time.Time{}, err
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return cfg, time.Time{}, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, time.Time{}, fmt.Errorf("%s: %w", s.path, err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, time.Time{}, fmt.Errorf("%s: %w", s.path, err)
	}
	return cfg, st.ModTime(), nil
}

// Update validates and persists new settings.
func (s *Store) Update(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.save(cfg); err != nil {
		return err
	}
	s.cfg = cfg
	return nil
}

// BaseDir is the directory holding the executable, config and logs.
func (s *Store) BaseDir() string { return s.baseDir }

// RecordingsDir returns the absolute recordings directory.
func (s *Store) RecordingsDir() string {
	dir := s.Get().RecordingsDir
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(s.baseDir, dir)
	}
	return dir
}

func (s *Store) save(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	if st, err := os.Stat(s.path); err == nil {
		s.modTime = st.ModTime()
	}
	return nil
}
