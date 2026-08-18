#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP_NAME="GopherTchi"
DIST="$ROOT/dist"
APP_DIR="$DIST/$APP_NAME.app"
CONTENTS="$APP_DIR/Contents"
MACOS="$CONTENTS/MacOS"
RESOURCES="$CONTENTS/Resources"

mkdir -p "$MACOS" "$RESOURCES"

echo "→ building $APP_NAME"
cd "$ROOT"
go build -o "$MACOS/$APP_NAME" .

cp "$ROOT/packaging/macos/Info.plist" "$CONTENTS/Info.plist"

# Optional app icon if present; menu-bar icons stay embedded in the binary.
if [[ -f "$ROOT/packaging/macos/AppIcon.icns" ]]; then
  cp "$ROOT/packaging/macos/AppIcon.icns" "$RESOURCES/AppIcon.icns"
fi

echo "✓ created $APP_DIR"
echo "  Custom packs live in ~/Library/Application Support/GopherTchi/packs/"
echo "  Run: open \"$APP_DIR\""
