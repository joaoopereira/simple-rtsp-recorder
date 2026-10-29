# 🎥 Simple RTSP Recorder

Records an RTSP camera to MP4 files on a Windows PC, controlled from a web page.
A single executable that runs as a Windows service and **updates itself** from GitHub releases. Updates are only installed when nothing is recording.

- One-click record and stop, with a live camera preview and a timer
- Recordings list with an in-browser player, download and delete
- Camera settings in the UI, with a "test connection" button
- Recordings survive power cuts: files are fragmented MP4 while recording and finalized when stopped
- Reconnects automatically if the camera drops, and resumes recording after a PC or service restart
- Optional automatic deletion of old recordings, plus a disk space indicator
- Logs viewable from the UI

## Install (Windows)

Run PowerShell **as Administrator**:

```powershell
irm https://raw.githubusercontent.com/joaoopereira/simple-rtsp-recorder/development/install.ps1 | iex
```

Then open `http://localhost:8080` and set up the camera (⚙️).

The same command upgrades an existing installation, including 1.x (Node.js) installs. It installs into the existing folder, converts `prod.env` to `config.json` and keeps the recordings. After that, updates are automatic.

<details>
<summary>Manual install</summary>

1. Download `simple-rtsp-recorder-win-x64.zip` from [Releases](https://github.com/joaoopereira/simple-rtsp-recorder/releases) and extract it.
2. As Administrator: `simple-rtsp-recorder.exe install`
3. To remove the service: `simple-rtsp-recorder.exe uninstall`

</details>

## Configuration

Most settings are edited in the UI and stored in `config.json` next to the executable:

| Key | Description |
|---|---|
| `camera.host`, `camera.port`, `camera.path` | RTSP address, e.g. `192.168.1.100`, `554`, `live1.sdp` |
| `camera.user`, `camera.password` | Credentials (optional). The password is never sent back to the browser |
| `camera.url` | *(file only)* Full RTSP URL; overrides the fields above |
| `recordingsDir` | Folder for recordings, relative to the install folder or absolute |
| `retentionDays` | Delete recordings older than N days (`0` = keep forever) |
| `autoUpdate` | Install new releases automatically when idle |
| `title` | Page title |
| `port` | *(file only)* HTTP port, default `8080`. Restart the service after changing it |
| `ffmpegPath` | *(file only)* Custom ffmpeg. By default the app uses the `ffmpeg.exe` shipped next to it |

Logs are written to `logs/recorder.log`.

## Releasing

GitHub → **Actions → Release → Run workflow**, then pick `patch`, `minor` or `major`. This also works from the GitHub mobile app. Installed apps update within the hour, once nothing is recording.

## Development

Requires Go and ffmpeg.

```bash
go test ./...                              # unit + ffmpeg integration tests
scripts/fake-camera.sh &                   # fake camera at rtsp://127.0.0.1:8554/live1.sdp
go run . -dir ./tmp                        # http://localhost:8080
GOOS=windows go build -o simple-rtsp-recorder.exe .
```

See [CLAUDE.md](CLAUDE.md) for the architecture.

## License

MIT. See [LICENSE](LICENSE).
