#!/bin/bash
# SwiftPMでビルドし、build/Slashstrip.appに組み立てる。Xcodeは不要（Command Line Toolsだけで通る）。
#
#   scripts/build-app.sh：ビルドのみ
#   scripts/build-app.sh --install：~/Applicationsに置き換えて起動し直す
#   scripts/build-app.sh --zip：配布用のbuild/Slashstrip-macos.zipとSHA-256も作る（組み合わせてよい）
set -euo pipefail

INSTALL=0
ZIP=0
for arg in "$@"; do
  case "$arg" in
    --install) INSTALL=1 ;;
    --zip) ZIP=1 ;;
    *) echo "unknown option: $arg" >&2; exit 2 ;;
  esac
done

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP="$ROOT/build/Slashstrip.app"
VERSION="0.1.0"

cd "$ROOT"
# アプリのリリースビルドはテスト（swift test）と別の作業ディレクトリで行い、構成の違うビルドが.buildを取り合わないようにする
SCRATCH="$ROOT/.build-release"
swift build -c release --product Slashstrip --scratch-path "$SCRATCH"

rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$(swift build -c release --scratch-path "$SCRATCH" --show-bin-path)/Slashstrip" "$APP/Contents/MacOS/Slashstrip"

cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleIdentifier</key><string>dev.local.slashstrip</string>
  <key>CFBundleName</key><string>Slashstrip</string>
  <key>CFBundleDisplayName</key><string>Slashstrip</string>
  <key>CFBundleExecutable</key><string>Slashstrip</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleVersion</key><string>1</string>
  <key>LSMinimumSystemVersion</key><string>14.0</string>
  <key>LSUIElement</key><true/>
  <key>NSHighResolutionCapable</key><true/>
  <key>NSAppleEventsUsageDescription</key>
  <string>Claude Codeのセッションへコマンドと許可応答を送るため、iTerm2を操作します。</string>
</dict>
</plist>
PLIST

# 署名用の証明書が無い環境ではアドホック署名になる。
# その場合、作り直すたびにmacOSの「iTerm2の操作」許可を聞き直される
codesign --force --sign - "$APP" >/dev/null
echo "built: $APP"

if [ "$ZIP" = 1 ]; then
  # リリースに添付する名前は版の番号を含めない（releases/latest/download/…が常に最新を指すように）
  ZIPFILE="$ROOT/build/Slashstrip-macos.zip"
  rm -f "$ZIPFILE"
  ditto -c -k --sequesterRsrc --keepParent "$APP" "$ZIPFILE"
  echo "zipped: $ZIPFILE"
  shasum -a 256 "$ZIPFILE"
fi

if [ "$INSTALL" = 1 ]; then
  DEST="$HOME/Applications/Slashstrip.app"
  pkill -x Slashstrip 2>/dev/null || true
  mkdir -p "$HOME/Applications"
  rm -rf "$DEST"
  cp -R "$APP" "$DEST"
  open "$DEST"
  echo "installed: $DEST"
fi
