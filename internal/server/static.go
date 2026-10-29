package server

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"strings"
)

// assets are the files index.html references; each gets a content hash.
var assets = []string{"style.css", "app.js", "icon.svg"}

// staticHandler serves the embedded UI. index.html is rewritten so it
// references its assets as e.g. style.css?v=1a2b3c4d5e6f: any change to a
// file changes its URL, so no cache (browser or Cloudflare) can ever serve an
// asset from another version. 1.x also had a /style.css, and Cloudflare
// kept serving that stale copy after the upgrade to 2.0.0.
type staticHandler struct {
	files   http.Handler
	index   []byte
	version map[string]string // asset → content hash
}

func newStaticHandler(fsys fs.FS) *staticHandler {
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		panic("web: index.html missing from embedded files: " + err.Error())
	}
	h := &staticHandler{files: http.FileServerFS(fsys), version: map[string]string{}}
	page := string(index)
	for _, name := range assets {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			panic("web: " + name + " missing from embedded files: " + err.Error())
		}
		sum := sha256.Sum256(data)
		v := hex.EncodeToString(sum[:6])
		h.version[name] = v
		for _, attr := range []string{`href="`, `src="`} {
			page = strings.ReplaceAll(page, attr+name+`"`, attr+name+"?v="+v+`"`)
		}
	}
	h.index = []byte(page)
	return h
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" || name == "index.html" {
		// Never cache the page itself: it carries the asset hashes.
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(h.index)
		return
	}
	if v, ok := h.version[name]; ok && r.URL.Query().Get("v") == v {
		// This exact content never changes under this URL.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	h.files.ServeHTTP(w, r)
}
