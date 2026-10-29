# CLAUDE.md

Simple RTSP Recorder records one RTSP camera (a futsal pitch) to MP4 files on a
Windows PC on the same LAN as the camera. It is controlled from a web page that is
published at https://futsal.retorta.com through a Cloudflare Tunnel. The owner
can't easily get to the PC, so **auto-update is critical**: never break it.

## Stack

- Go (stdlib only, plus `golang.org/x/sys` for the Windows service and disk APIs), one static `.exe`.
- ffmpeg does the recording (`-c copy`, no re-encode). A Windows ffmpeg.exe is shipped next to the exe in the release zip.
- The web UI is plain HTML/CSS/JS with no build step, embedded with `go:embed`. UI text is **Portuguese (pt-PT)**; code, comments and logs are English.

## Layout

```
main.go                     commands (run/install/uninstall/version), wiring, restart after update
internal/config             config.json load/save/validate; migration from 1.x prod.env
internal/recorder           ffmpeg process: start/stop, reconnect, finalize (faststart remux), snapshot, list/prune
internal/server             HTTP API + static files (endpoint list in the package doc); auth.go = admin login
internal/updater            GitHub release check → download + sha256 verify → swap exe when idle → restart
internal/service            Windows service run/install/uninstall (stub on other OSes)
internal/logging            slog → rotating file logs/recorder.log + stdout + in-memory tail for /api/logs
internal/sysinfo            free disk space (windows / unix)
web/                        index.html, app.js, style.css, icon.svg (embedded)
scripts/fake-camera.sh      local RTSP test camera (mediamtx + ffmpeg test pattern)
install.ps1                 one-time installer/upgrader (irm | iex)
.github/workflows           ci.yml (fmt, vet linux+windows, test -race, build), release.yml
```

Runtime files in the install folder: `simple-rtsp-recorder.exe`, `ffmpeg.exe`,
`config.json`, `recording.state` (present while a recording should be running, so it
resumes after a restart), `logs/`, `recordings/`.

## Contracts that must not change

Installed PCs depend on these. Changing any of them strands the PC on the old version:

- Release assets are named `simple-rtsp-recorder.exe`, `checksums.txt` (sha256sum format) and `simple-rtsp-recorder-win-x64.zip`. See `updater.AssetName` and `release.yml`.
- The repo is `joaoopereira/simple-rtsp-recorder` and releases use semver tags `vX.Y.Z` (`updater.Newer`).
- The service name is `SimpleRTSPRecorder`. Restart after an update relies on the SCM recovery actions set in `service.Install`, together with a non-zero exit from the service.
- `config.json` stays backward compatible: only add fields, with defaults in `config.Default()`.
- Recording file names follow `2006-01-02_150405.mp4`. The UI parses the date from the name.

## Commands

```bash
gofmt -l .                          # must print nothing
go vet ./... && GOOS=windows go vet ./...   # always vet the windows build too
go test -race ./...                 # recorder tests need ffmpeg on PATH (skipped otherwise)
node --check web/app.js
GOOS=windows GOARCH=amd64 go build -o simple-rtsp-recorder.exe .
```

`go.mod` requires Go 1.26 (needed by `golang.org/x/sys`). If the installed Go is older, run commands with `GOTOOLCHAIN=auto`, which downloads the right toolchain from the Go proxy.

To run the app against the fake camera, see `.claude/skills/run-app/SKILL.md`.

## Conventions

- Conventional commits (`feat:`, `fix:`, `chore:` …). The default branch is `development`. `CHANGELOG.md` and the release notes are generated from `feat`/`fix`/`perf` subjects and `BREAKING CHANGE:` footers by `scripts/changelog.py` during the release. Don't edit `CHANGELOG.md` by hand for new versions.
- Prefer the standard library. Don't add a frontend framework or build step.
- The recorder's state machine is `idle → connecting → recording ⇄ reconnecting → stopping → idle`. All state lives under `Recorder.mu`. Don't call ffmpeg while holding the lock.
- Never log credentials: use `config.Redact(url)`. The camera password is write-only in the API: `GET /api/settings` returns `hasPassword`, and `PUT` keeps the stored password when `password` is absent.
- Validate file names with `recorder.ValidName` before touching the disk (prevents path traversal).
- Windows-only code goes in `_windows.go` files, with a `_other.go` counterpart (`//go:build !windows`).
- User-facing error messages are Portuguese, because they are shown in the UI as they are.
- A Windows service has **no console**: writes to stdout/stderr fail. Never let a stdout failure stop other output (that is why `logging` uses `allWriters` instead of `io.MultiWriter`; in v2.0.x the "Registos" page was empty because of it), and never rely on console output for anything the service needs.
- The site sits behind Cloudflare, which caches `.css`, `.js` and `.svg` by extension. `index.html` is served with `Cache-Control: no-store`, and every asset it references is rewritten to `name?v=<content hash>` (`internal/server/static.go`). A new file in `web/` must be added both to `//go:embed` in `web/embed.go` and to `assets` in `static.go`. Never reference an asset without the hash: in v2.0.0, Cloudflare kept serving the 1.x `style.css`, and the public page rendered unstyled.

## Adding things

- **API endpoint**: register a handler in `server.Handler()`, add it to the package doc list, and add a test in `server_test.go`. Then call it from `web/app.js` through `api()`.
- **Setting**: add the field to `config.Config` with a default and validation, then to `settingsDTO` and its `toDTO`/`apply` in `server.go`, then to the settings form in `index.html` and to `openSettings`/`formSettings` in `app.js`. Settings are admin-only automatically. `apply` must keep fields that aren't in the DTO, such as `adminPasswordHash`.
- **ffmpeg arguments**: `recordArgs` / `inputArgs` in `recorder.go`. Cover the change in `recorder_test.go`, which uses a real ffmpeg and a generated test file.

## Security model

- **Admin-only: settings.** `GET/PUT /api/settings`, `POST /api/settings/test` and `PUT /api/admin/password` are wrapped in `requireAdmin` (`internal/server/auth.go`). Every new endpoint that changes configuration or exposes secrets must be wrapped too.
- **Public: everything else** (record/stop, list/play/download/delete recordings, logs, update check). Anyone who reaches https://futsal.retorta.com can use it, so Cloudflare Access is recommended in front of it.
- The admin password is stored in `config.json` as a PBKDF2-SHA256 hash (`adminPasswordHash`). It is **never** set from the UI without being logged in, because the site is public and the first visitor would become admin. It is set on the PC, either by `install.ps1` (which prompts when none exists) or with `echo <pw> | simple-rtsp-recorder.exe set-admin-password`. Once logged in, it can be changed in the UI.
- Sessions use an HMAC-signed cookie (`srr_admin`, HttpOnly, SameSite=Strict, Secure behind the tunnel). The key is random per process, so a restart or update logs everyone out. Changing the password invalidates other sessions.
- Failed logins are locked out per IP: 5 failures in 15 minutes. The IP comes from `Cf-Connecting-Ip`, trusted only when the connection comes from loopback (cloudflared).
- `config.Store` reloads `config.json` when the file changes on disk, so edits from the CLI or by hand apply without restarting the service.
- Never return the camera password or the admin hash from the API.
