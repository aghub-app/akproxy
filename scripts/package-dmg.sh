#!/bin/bash
# Build a drag-install DMG. The volume root contains akproxy.app and an
# Applications symlink. Finder positions them on build/dmg-background.png.
set -euo pipefail

source_dir="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$source_dir/.." && pwd)"
app="$root/build/bin/akproxy.app"
background="$root/build/dmg-background.png"

# Finder icon centers, in points. Keep in sync with scripts/render-dmg-background.py.
window_width=720
window_height=680
titlebar=28
app_x=196
app_y=430
applications_x=524
applications_y=430
icon_size=128

if [[ $# -gt 1 ]]; then
  echo "usage: package-dmg.sh [output.dmg]" >&2
  exit 2
fi

dmg="${1:-build/bin/akproxy.dmg}"
if [[ "$dmg" != /* ]]; then
  dmg="$root/$dmg"
fi

if [[ ! -d "$app" ]]; then
  echo "missing $app (run wails3 task build first)" >&2
  exit 1
fi
if [[ ! -f "$app/Contents/MacOS/akproxy" ]]; then
  echo "missing $app/Contents/MacOS/akproxy" >&2
  exit 1
fi
if [[ ! -f "$background" ]]; then
  echo "missing $background" >&2
  exit 1
fi
if [[ -d /Volumes/akproxy ]]; then
  echo "/Volumes/akproxy is already mounted" >&2
  exit 1
fi

work="$(mktemp -d)"
rw="$work/rw.dmg"
mounted=0
cleanup() {
  if [[ "$mounted" -eq 1 ]]; then
    hdiutil detach /Volumes/akproxy >/dev/null 2>&1 || hdiutil detach -force /Volumes/akproxy >/dev/null 2>&1 || true
  fi
  rm -rf "$work"
}
trap cleanup EXIT

kb="$(du -sk "$app" | awk '{print $1}')"
mb="$((kb / 1024 + 48))"
hdiutil create -size "${mb}m" -fs APFS -volname akproxy -ov "$rw" >/dev/null
hdiutil attach -readwrite -noverify -noautoopen "$rw" >/dev/null
mounted=1

ditto "$app" /Volumes/akproxy/akproxy.app
ln -s /Applications /Volumes/akproxy/Applications
mkdir -p /Volumes/akproxy/.background
cp "$background" /Volumes/akproxy/.background/background.png
chflags hidden /Volumes/akproxy/.background

osascript <<EOF
tell application "Finder"
  tell disk "akproxy"
    open
    set current view of container window to icon view
    set toolbar visible of container window to false
    set statusbar visible of container window to false
    set the bounds of container window to {120, 80, 120 + $window_width, 80 + $titlebar + $window_height}
    set viewOptions to the icon view options of container window
    set arrangement of viewOptions to not arranged
    set icon size of viewOptions to $icon_size
    set text size of viewOptions to 13
    set background picture of viewOptions to file ".background:background.png"
    delay 1
    -- Closing once lets Finder commit the window frame. Positions set before
    -- that close are stored about 45 points lower, so set them again after reopen.
    close
    open
    delay 1
    set position of item "akproxy.app" of container window to {$app_x, $app_y}
    set position of item "Applications" of container window to {$applications_x, $applications_y}
    update without registering applications
    delay 1
    close
  end tell
end tell
EOF

sync
hdiutil detach /Volumes/akproxy >/dev/null
mounted=0

mkdir -p "$(dirname "$dmg")"
rm -f "$dmg"
hdiutil convert "$rw" -format UDZO -ov -o "$dmg" >/dev/null
echo "wrote $dmg"
