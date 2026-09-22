#!/bin/bash
set -euo pipefail
umask 077
cd "$(dirname "$0")/.."

: "${VERSION:?release version required}"
: "${AKPROXY_CERTIFICATE_BASE64:?akproxy certificate required}"
: "${AKPROXY_CERTIFICATE_PASSWORD:?akproxy certificate password required}"
: "${AKPROXY_SIGNING_IDENTITY:?akproxy certificate SHA-1 required}"
: "${AKPROXY_APPLE_ID:?Apple ID required}"
: "${AKPROXY_APPLE_TEAM_ID:?Apple team ID required}"
: "${AKPROXY_APPLE_APP_PASSWORD:?akproxy notarization password required}"

# Use an isolated keychain; never import release keys into the login keychain.
original_keychains=()
while read -r existing_keychain; do
  original_keychains+=("${existing_keychain//\"/}")
done < <(security list-keychains -d user)
signing_dir="$(mktemp -d)"
keychain="$signing_dir/akproxy.keychain-db"
keychain_password="$(openssl rand -hex 32)"
cleanup() {
  security list-keychains -d user -s "${original_keychains[@]}" >/dev/null 2>&1 || true
  security delete-keychain "$keychain" >/dev/null 2>&1 || true
  rm -f "$signing_dir/akproxy.p12" "$signing_dir/DeveloperIDCA.cer" "$signing_dir/DeveloperIDG2CA.cer"
  rmdir "$signing_dir" 2>/dev/null || true
}
trap cleanup EXIT
printf '%s' "$AKPROXY_CERTIFICATE_BASE64" | base64 --decode > "$signing_dir/akproxy.p12"
security create-keychain -p "$keychain_password" "$keychain"
security set-keychain-settings -lut 21600 "$keychain"
security unlock-keychain -p "$keychain_password" "$keychain"
security import "$signing_dir/akproxy.p12" -k "$keychain" -P "$AKPROXY_CERTIFICATE_PASSWORD" -T /usr/bin/codesign
security set-key-partition-list -S apple-tool:,apple:,codesign: -s -k "$keychain_password" "$keychain" >/dev/null
# --keychain selects the identity, but certificate-chain resolution still uses
# the user search list. Fresh runners may also lack the Developer ID intermediates.
security list-keychains -d user -s "$keychain" "${original_keychains[@]}"
for intermediate in DeveloperIDCA DeveloperIDG2CA; do
  curl --fail --silent --show-error --location "https://www.apple.com/certificateauthority/$intermediate.cer" -o "$signing_dir/$intermediate.cer"
  security import "$signing_dir/$intermediate.cer" -k "$keychain"
done
signing_identities="$(security find-identity -v -p codesigning "$keychain")"
if [[ "$signing_identities" != *"$AKPROXY_SIGNING_IDENTITY"* ]]; then
  echo 'The configured certificate is not a valid signing identity in the release keychain.' >&2
  security find-identity -p codesigning "$keychain" >&2
  exit 1
fi

# The normal build already produced the host architecture.
other_arch=amd64
if [[ "$(go env GOARCH)" == amd64 ]]; then other_arch=arm64; fi
GOARCH="$other_arch" CGO_ENABLED=1 CGO_CFLAGS=-mmacosx-version-min=12.0 CGO_LDFLAGS=-mmacosx-version-min=12.0 \
  go build -tags production -ldflags "-s -w -X main.version=$VERSION" -o build/bin/akproxy-other .
lipo -create build/bin/akproxy build/bin/akproxy-other -output build/bin/akproxy-universal
cp build/bin/akproxy-universal build/bin/akproxy
bash scripts/package-app.sh "$VERSION"

app=build/bin/akproxy.app
codesign --force --options runtime --timestamp --keychain "$keychain" --sign "$AKPROXY_SIGNING_IDENTITY" "$app"
codesign --verify --strict "$app"
ditto -c -k --keepParent "$app" build/bin/akproxy-notary.zip
xcrun notarytool submit build/bin/akproxy-notary.zip --apple-id "$AKPROXY_APPLE_ID" \
  --team-id "$AKPROXY_APPLE_TEAM_ID" --password "$AKPROXY_APPLE_APP_PASSWORD" --wait --timeout 30m
xcrun stapler staple "$app"
xcrun stapler validate "$app"
spctl --assess --type execute "$app"
# Omit __MACOSX resource-fork entries: Wails expects one root .app in the ZIP.
ditto -c -k --norsrc --keepParent "$app" build/bin/akproxy-darwin-universal.zip
