#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."
app="build/bin/akproxy.app"
test -f build/bin/akproxy
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
wails3 generate icons -input build/appicon.png -macfilename build/bin/iconfile.icns -windowsfilename build/bin/icon.ico
cp build/bin/akproxy "$app/Contents/MacOS/akproxy"
cp build/bin/iconfile.icns "$app/Contents/Resources/iconfile.icns"
cp build/darwin/Info.plist "$app/Contents/Info.plist"
codesign --force --sign - "$app"
