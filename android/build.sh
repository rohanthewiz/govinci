#!/bin/sh
# Build the Govinci AAR with gomobile.
#
# Binds the `mobile` bridge package (the Kotlin runtime's call surface) plus
# the demo app package, whose init() registers the root view with the bridge.
# To ship your own app instead, replace the mobileapp path with your app's
# package — any Go package whose init calls mobile.Register works.
set -e

# gomobile/gobind live in GOPATH/bin, which isn't always on PATH.
PATH="$PATH:$(go env GOPATH)/bin"

if ! command -v gomobile >/dev/null; then
  echo "gomobile is required: go install golang.org/x/mobile/cmd/gomobile@latest golang.org/x/mobile/cmd/gobind@latest" >&2
  exit 1
fi

# gomobile locates the SDK/NDK through these; default to the standard macOS
# install location when the caller hasn't exported them.
[ -n "$ANDROID_HOME" ] || export ANDROID_HOME="$HOME/Library/Android/sdk"
if [ -z "$ANDROID_NDK_HOME" ] && [ -d "$ANDROID_HOME/ndk" ]; then
  export ANDROID_NDK_HOME="$ANDROID_HOME/ndk/$(ls "$ANDROID_HOME/ndk" | sort -V | tail -1)"
fi

cd "$(dirname "$0")/.."
mkdir -p android/app/libs
gomobile bind -target=android -androidapi 24 \
  -o android/app/libs/govinci.aar \
  ./mobile ./examples/mobileapp
