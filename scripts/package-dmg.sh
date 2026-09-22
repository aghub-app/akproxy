#!/bin/bash
# hdiutil -srcfolder images a directory's children. Stage the .app so the
# volume root contains akproxy.app rather than the bundle's Contents.
set -euo pipefail

source_dir="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$source_dir/.." && pwd)"
app="$root/build/bin/akproxy.app"

if [[ $# -gt 1 ]]; then
  echo "usage: package-dmg.sh [output.dmg]" >&2
  exit 2
fi

dmg="${1:-build/bin/akproxy.dmg}"
if [[ "$dmg" != /* ]]; then
  dmg="$root/$dmg"
fi

if [[ ! -d "$app" ]]; then
  echo "missing $app (run wails build first)" >&2
  exit 1
fi
if [[ ! -f "$app/Contents/MacOS/akproxy" ]]; then
  echo "missing $app/Contents/MacOS/akproxy" >&2
  exit 1
fi

stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT
ditto "$app" "$stage/akproxy.app"

mkdir -p "$(dirname "$dmg")"
rm -f "$dmg"
hdiutil create -volname akproxy -srcfolder "$stage" -ov -format UDZO "$dmg"
