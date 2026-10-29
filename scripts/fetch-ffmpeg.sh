#!/usr/bin/env bash
# Downloads a Windows ffmpeg.exe from BtbN/FFmpeg-Builds' "latest" release.
#
#   scripts/fetch-ffmpeg.sh <out-dir>   # writes <out-dir>/ffmpeg.exe (+ FFMPEG-LICENSE.txt)
#   scripts/fetch-ffmpeg.sh --check     # only checks a suitable asset exists (CI)
#
# Asset names change with each FFmpeg release branch (e.g.
# ffmpeg-n7.1-latest-win64-lgpl-7.1.zip), so the newest release-branch LGPL
# static build is looked up instead of hard-coding a name. LGPL is enough:
# we only stream-copy, never encode. Needs the gh CLI and GH_TOKEN.
set -euo pipefail

REPO=BtbN/FFmpeg-Builds
assets=$(gh release view latest --repo "$REPO" --json assets --jq '.assets[].name')
name=$(grep -E '^ffmpeg-n[0-9.]+-latest-win64-lgpl-[0-9.]+\.zip$' <<<"$assets" | sort -V | tail -1 || true)
if [[ -z "$name" ]]; then
  name=$(grep -Fx 'ffmpeg-master-latest-win64-lgpl.zip' <<<"$assets" || true)
fi
if [[ -z "$name" ]]; then
  echo "no win64-lgpl ffmpeg build found in $REPO 'latest'; assets were:" >&2
  echo "$assets" >&2
  exit 1
fi
echo "ffmpeg asset: $name"
[[ "${1:-}" == "--check" ]] && exit 0

out=${1:?usage: fetch-ffmpeg.sh <out-dir> | --check}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
gh release download latest --repo "$REPO" --pattern "$name" --dir "$work"
unzip -q "$work/$name" -d "$work/x"
mkdir -p "$out"
cp "$(find "$work/x" -name ffmpeg.exe | head -1)" "$out/ffmpeg.exe"
lic=$(find "$work/x" -name LICENSE.txt | head -1)
[[ -n "$lic" ]] && cp "$lic" "$out/FFMPEG-LICENSE.txt"
ls -la "$out/ffmpeg.exe"
