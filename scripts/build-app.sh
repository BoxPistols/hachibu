#!/bin/bash
# SwiftPMでビルドし、build/Hachibu.appに組み立てる。Xcodeは不要（Command Line Toolsだけで通る）。
#
#   scripts/build-app.sh：ビルドのみ
#   scripts/build-app.sh --install：~/Applicationsに置き換えて起動し直す
#   scripts/build-app.sh --zip：配布用のbuild/Hachibu-macos.zipとSHA-256も作る（組み合わせてよい）
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
APP="$ROOT/build/Hachibu.app"
VERSION="0.3.2"

cd "$ROOT"
# アプリのリリースビルドはテスト（swift test）と別の作業ディレクトリで行い、構成の違うビルドが.buildを取り合わないようにする
SCRATCH="$ROOT/.build-release"
# Apple SiliconとIntelの両方で動くように、2つのCPU向けを1つのバイナリにまとめる
ARCHS=(--arch arm64 --arch x86_64)
swift build -c release --product Hachibu --scratch-path "$SCRATCH" "${ARCHS[@]}"

rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$(swift build -c release --scratch-path "$SCRATCH" "${ARCHS[@]}" --show-bin-path)/Hachibu" "$APP/Contents/MacOS/Hachibu"
# アイコンはscripts/make-icon.swiftで描いたもの（形を変えるときはそちらを直して作り直す）
cp "$ROOT/Resources/AppIcon.icns" "$APP/Contents/Resources/AppIcon.icns"

cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleIdentifier</key><string>io.github.boxpistols.hachibu</string>
  <key>CFBundleName</key><string>Hachibu</string>
  <key>CFBundleDisplayName</key><string>Hachibu</string>
  <key>CFBundleExecutable</key><string>Hachibu</string>
  <key>CFBundleIconFile</key><string>AppIcon</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleVersion</key><string>1</string>
  <key>LSMinimumSystemVersion</key><string>14.0</string>
  <key>LSUIElement</key><true/>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
PLIST

# 署名用の証明書が無い環境ではアドホック署名になる。
# その場合、作り直すたびにmacOSがキーチェーンやログイン項目の許可を聞き直すことがある
codesign --force --sign - "$APP" >/dev/null
echo "built: $APP"

if [ "$ZIP" = 1 ]; then
  # リリースに添付する名前は版の番号を含めない（releases/latest/download/…が常に最新を指すように）
  ZIPFILE="$ROOT/build/Hachibu-macos.zip"
  rm -f "$ZIPFILE"
  ditto -c -k --sequesterRsrc --keepParent "$APP" "$ZIPFILE"
  echo "zipped: $ZIPFILE"
  shasum -a 256 "$ZIPFILE"
fi

if [ "$INSTALL" = 1 ]; then
  DEST="$HOME/Applications/Hachibu.app"
  pkill -x Hachibu 2>/dev/null || true
  mkdir -p "$HOME/Applications"
  rm -rf "$DEST"
  cp -R "$APP" "$DEST"
  # ターミナルの中から開くと、macOSはメニューバーの項目をターミナルのものとして覚え、
  # ターミナルの許可が切られていると項目が出なくなる。launchd経由で、ターミナルを親に持たない形で開く
  launchctl remove io.github.boxpistols.hachibu.install 2>/dev/null || true
  launchctl submit -l io.github.boxpistols.hachibu.install -- /usr/bin/open "$DEST"
  sleep 2
  launchctl remove io.github.boxpistols.hachibu.install 2>/dev/null || true
  echo "installed: $DEST"
fi
