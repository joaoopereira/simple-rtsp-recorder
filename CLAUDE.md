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
internal/server             HTTP API + static files (endpoint list in the package doc)
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

If the sandbox's Go wants to download a newer toolchain, prefix commands with `GOTOOLCHAIN=local`.

To run the app against the fake camera, see `.claude/skills/run-app/SKILL.md`.

## Conventions

- Conventional commits (`feat:`, `fix:`, `chore:` …). The default branch is `development`.
- Prefer the standard library. Don't add a frontend framework or build step.
- The recorder's state machine is `idle → connecting → recording ⇄ reconnecting → stopping → idle`. All state lives under `Recorder.mu`. Don't call ffmpeg while holding the lock.
- Never log credentials: use `config.Redact(url)`. The camera password is write-only in the API: `GET /api/settings` returns `hasPassword`, and `PUT` keeps the stored password when `password` is absent.
- Validate file names with `recorder.ValidName` before touching the disk (prevents path traversal).
- Windows-only code goes in `_windows.go` files, with a `_other.go` counterpart (`//go:build !windows`).
- User-facing error messages are Portuguese, because they are shown in the UI as they are.

## Adding things

- **API endpoint**: register a handler in `server.Handler()`, add it to the package doc list, and add a test in `server_test.go`. Then call it from `web/app.js` through `api()`.
- **Setting**: add the field to `config.Config` with a default and validation, then to `settingsDTO` and its `toDTO`/`apply` in `server.go`, then to the settings form in `index.html` and to `openSettings`/`formSettings` in `app.js`.
- **ffmpeg arguments**: `recordArgs` / `inputArgs` in `recorder.go`. Cover the change in `recorder_test.go`, which uses a real ffmpeg and a generated test file.

## Security note

The UI has no login of its own. Anyone who can reach https://futsal.retorta.com can start and stop recordings, delete them and change settings. It should sit behind Cloudflare Access. Don't expose anything more sensitive than what is there now (for example, never return the camera password).
