---
name: run-app
description: Run Simple RTSP Recorder locally against a fake RTSP camera, drive it through the API and take UI screenshots. Use to verify a change end-to-end (recording, reconnect, resume, UI) before pushing.
---

# Run the app locally

Everything below works on a Linux sandbox. Windows-only parts (service install,
self-update swap) can't run here: cover them with unit tests and
`GOOS=windows go vet ./...`.

## 1. Fake camera

```bash
MEDIAMTX=/tmp/mediamtx setsid scripts/fake-camera.sh > /tmp/fakecam.log 2>&1 &
sleep 5; cat /tmp/fakecam.log   # should only print the "fake camera: rtsp://..." line
```

- On first use the script builds mediamtx from the Go module proxy, which takes about a minute. GitHub release downloads are usually blocked in the sandbox.
- Stop it with `pkill -x mediamtx; pkill -x ffmpeg`. **Never** run `pkill -f <pattern>` here: the pattern also appears in your own shell command line, so it kills that shell (exit 144).

## 2. App

```bash
GOTOOLCHAIN=local go build -o /tmp/srr .
D=/tmp/srrtest; rm -rf $D; mkdir -p $D
printf 'RTSP_IP=127.0.0.1\nRTSP_PORT=8554\nRTSP_SDP=live1.sdp\nOUTPUT_DIR=recordings\nLABEL_TITLE=Teste\n' > $D/prod.env   # exercises the 1.x migration
setsid /tmp/srr -dir $D > /tmp/srr.log 2>&1 &
```

Alternatively, skip `prod.env` and PUT the settings (see below).

## 3. Drive it

```bash
curl -s localhost:8080/api/status
curl -s -XPOST localhost:8080/api/recording/start; sleep 5
curl -s -XPOST localhost:8080/api/recording/stop;  sleep 3
curl -s localhost:8080/api/recordings
ffprobe -v error -show_entries format=duration:stream=codec_name -of compact $D/recordings/*.mp4
curl -s -o /tmp/snap.jpg -w '%{http_code}\n' localhost:8080/api/snapshot
curl -s -XPUT localhost:8080/api/settings -H 'Content-Type: application/json' \
  -d '{"title":"Teste","camera":{"host":"127.0.0.1","port":8554,"path":"live1.sdp","user":""},"recordingsDir":"recordings","retentionDays":0,"autoUpdate":true}'
```

Scenarios worth checking when touching the recorder:

- **Camera drop**: start a recording, `pkill -x mediamtx` → the state becomes `reconnecting`. Restart the camera and the state goes back to `recording` with a new file and the same `startedAt`.
- **Resume after restart**: while recording, `pkill -x srr` (SIGTERM) and start it again. It should resume, and the log shows "resuming recording interrupted by restart".
- **Logs**: `tail $D/logs/recorder.log`, or `curl localhost:8080/api/logs`.

## 4. Screenshots

```bash
NODE_PATH=$(npm root -g) node .claude/skills/run-app/screenshot.js <out-dir>
```

The script saves `desktop-light.png`, `phone-dark.png` and `settings.png`, and prints any page errors. Open the PNGs with the Read tool and check the layout.

## 5. Clean up

```bash
pkill -x srr; pkill -x mediamtx; pkill -x ffmpeg
```
