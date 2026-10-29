---
name: release
description: Publish a new version of Simple RTSP Recorder so the Windows PC auto-updates. Use when the user asks to release, ship or deploy a change.
---

# Release

Installed PCs check GitHub `releases/latest` every hour (and one minute after
start-up). They download `simple-rtsp-recorder.exe`, verify it against
`checksums.txt`, and swap it in **only when nothing is recording or being
finalized**. The Windows service then restarts on the new version.

## Steps

1. Make sure the change is merged into `development` and CI is green there.
2. Trigger the **Release** workflow (`.github/workflows/release.yml`) on `development` with `bump` = `patch` (fixes), `minor` (features) or `major` (breaking):
   - with the GitHub MCP tools, use `actions_run_trigger` (workflow `release.yml`, ref `development`, inputs `{"bump":"patch"}`), or
   - ask the user to go to GitHub → Actions → Release → Run workflow (this also works in the mobile app).
3. Watch the run until it finishes (`actions_get` / `actions_list`). It tests, cross-compiles, bundles ffmpeg.exe and publishes the tag and release with three assets: `simple-rtsp-recorder.exe`, `simple-rtsp-recorder-win-x64.zip` and `checksums.txt`.
4. Tell the user the version number and that the PC will pick it up within about an hour, once it is idle. In the UI footer, the version changes, and "Verificar atualizações" forces an immediate check.

## If the release fails

- **"Bundle ffmpeg" step**: the BtbN download URL in `FFMPEG_URL` changed. Pick a current `win64-lgpl` zip from https://github.com/BtbN/FFmpeg-Builds/releases and update the env var.
- **Tests**: fix them on a branch first. Never publish with failing tests: a broken binary would auto-install on the PC.

## Never

- Rename the assets, the repo, the tag format or the service name. Installed PCs would stop updating (see "Contracts" in CLAUDE.md).
- Publish a release whose exe can't start. The PC would be stuck until someone goes there. If in doubt, run the `run-app` skill first.
