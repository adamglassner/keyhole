#!/usr/bin/env bash
# Builds Keyhole.exe and the Windows installer (dist/Keyhole-Setup-<version>.exe).
# Needs Go 1.24+ and makensis (NSIS 3) on the PATH. Works in Git Bash on Windows, or on Linux/macOS.
# The Android APK is built by build-android.sh.
set -euo pipefail
cd "$(dirname "$0")"
VERSION=$(grep -oE 'Version = "[0-9.]+"' game/game.go | grep -oE '[0-9.]+')
echo "Building Keyhole $VERSION"
grep -q "^## Version $VERSION" CHANGELOG.md || { echo "CHANGELOG.md has no '## Version $VERSION' section"; exit 1; }
rm -rf dist && mkdir -p dist/Keyhole

# exe icon and version info
MKRES="dist/mkres$(go env GOEXE)"
( cd tools/mkres && GOFLAGS=-mod=vendor go build -o "../../$MKRES" . )
"./$MKRES" -version "$VERSION" -icon branding/icon_1024.png -out rsrc_windows_amd64.syso \
  -name Keyhole -exe Keyhole.exe -internal Keyhole -identity Ironite.Keyhole -desc "Keyhole - The Daily Word Vault"
rm -f "$MKRES"

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H windowsgui" -o dist/Keyhole/Keyhole.exe .
cp LICENSE.txt CHANGELOG.md THIRD-PARTY-NOTICES.txt dist/Keyhole/
makensis -V2 -DVERSION="$VERSION" -DSRCDIR="../dist/Keyhole" -DOUTDIR="../dist" installer/keyhole.nsi
# release notes = newest "## " section of the changelog
awk '/^## /{n++} n==1' CHANGELOG.md | tail -n +2 > dist/RELEASE_NOTES.md
echo "Done: dist/Keyhole-Setup-$VERSION.exe"
