package config

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// migrateLegacy builds settings from the prod.env / .env file used by the
// Node.js version (1.x). It returns defaults when neither file exists.
func migrateLegacy(baseDir string) (Config, bool, error) {
	cfg := Default()
	for _, name := range []string{"prod.env", ".env"} {
		env, err := readEnvFile(filepath.Join(baseDir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return cfg, false, err
		}
		applyEnv(&cfg, env)
		if err := cfg.Validate(); err != nil {
			// Bad legacy values should not stop the app from starting:
			// fall back to defaults and let the user fix it in the UI.
			return Default(), false, nil
		}
		return cfg, true, nil
	}
	return cfg, false, nil
}

func applyEnv(cfg *Config, env map[string]string) {
	set := func(key string, dst *string) {
		if v, ok := env[key]; ok && v != "" {
			*dst = v
		}
	}
	set("RTSP_IP", &cfg.Camera.Host)
	set("RTSP_SDP", &cfg.Camera.Path)
	set("RTSP_USER", &cfg.Camera.User)
	set("RTSP_PASSWORD", &cfg.Camera.Password)
	set("OUTPUT_DIR", &cfg.RecordingsDir)
	set("LABEL_TITLE", &cfg.Title)
	if p, err := strconv.Atoi(env["RTSP_PORT"]); err == nil {
		cfg.Camera.Port = p
	}
	if p, err := strconv.Atoi(env["PORT"]); err == nil {
		cfg.Port = p
	}
}

// readEnvFile parses KEY=VALUE lines, ignoring comments and blank lines.
func readEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	env := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(sc.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		}
		env[strings.TrimSpace(key)] = val
	}
	return env, sc.Err()
}
