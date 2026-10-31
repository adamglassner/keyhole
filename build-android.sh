#!/usr/bin/env bash
# Builds the Android APK: dist/Keyhole-<version>.apk
# Needs Go 1.24+, the Android SDK + NDK (ANDROID_HOME, ANDROID_NDK_HOME), JDK 17 and Gradle 8.7+.
# For a signed release set KEYHOLE_KEYSTORE, KEYHOLE_KEYSTORE_PASSWORD, KEYHOLE_KEY_ALIAS and KEYHOLE_KEY_PASSWORD.
set -euo pipefail
cd "$(dirname "$0")"
VERSION=$(grep -oE 'Version = "[0-9.]+"' game/game.go | grep -oE '[0-9.]+')
echo "Building Keyhole $VERSION for Android"
go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile
mkdir -p android/app/libs dist
"$(go env GOPATH)/bin/ebitenmobile" bind -target android -androidapi 21 -javapkg com.ironite.keyhole \
  -o android/app/libs/keyhole.aar ./mobile
( cd android && gradle --no-daemon assembleRelease -PappVersion="$VERSION" )
cp android/app/build/outputs/apk/release/app-release.apk "dist/Keyhole-$VERSION.apk"
echo "Done: dist/Keyhole-$VERSION.apk"
