// Package web holds the static web UI, embedded in the executable.
package web

import "embed"

// FS contains index.html and its assets.
//
//go:embed index.html app.js style.css icon.svg
var FS embed.FS
