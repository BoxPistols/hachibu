#!/bin/bash
# Hachibu for Windowsをビルドする。GoがWindows向けに作るので、Linux、macOS、WindowsのGit Bashのどれでも動く（Go 1.26以降が要る）。
#
#   windows/build.sh        build/Hachibu.exe を作る
#   windows/build.sh --zip  配布用の build/Hachibu-windows-x64.zip とSHA-256も作る
#
# GOARCH=arm64 を付けるとArm向け（Hachibu-windows-arm64.zip）を作る。
# アイコンとマニフェスト（画面ごとの拡大率への対応、タスクダイアログに要るcomctl32 6.0）は、go-winresで.sysoにして埋め込む
set -euo pipefail

ZIP=0
for arg in "$@"; do
  case "$arg" in
    --zip) ZIP=1 ;;
    *) echo "unknown option: $arg" >&2; exit 2 ;;
  esac
done

cd "$(dirname "$0")"
# リリースのタグ（windows-v0.1.0）と揃える。新しいバージョンの確認はこの値と比べる
VERSION="0.1.0"
ARCH="${GOARCH:-amd64}"
case "$ARCH" in
  amd64) LABEL=x64 ;;
  arm64) LABEL=arm64 ;;
  *) echo "unsupported GOARCH: $ARCH" >&2; exit 2 ;;
esac

# go-winresはこのマシン向けに動かすので、GOOSとGOARCHを外して実行する
rm -f rsrc_windows_*.syso
env -u GOOS -u GOARCH go run github.com/tc-hib/go-winres@v0.3.3 simply \
  --arch "$ARCH" \
  --icon ../docs/images/app-icon.png \
  --manifest gui \
  --product-name Hachibu \
  --file-description "Hachibu" \
  --product-version "$VERSION" \
  --file-version "$VERSION" \
  --original-filename Hachibu.exe \
  --copyright "Copyright (c) 2026 BoxPistols"

mkdir -p build
# -H windowsgui: 起動しても黒い窓（コンソール）を出さない
GOOS=windows GOARCH="$ARCH" CGO_ENABLED=0 go build -trimpath \
  -ldflags "-H windowsgui -s -w -X github.com/BoxPistols/hachibu/windows/internal/app.Version=${VERSION}" \
  -o build/Hachibu.exe .
echo "built: windows/build/Hachibu.exe ($VERSION, $ARCH)"

if [ "$ZIP" = 1 ]; then
  # 添付する名前はバージョン番号を含めない（READMEのリンクが常に最新を指すように）
  ZIPFILE="build/Hachibu-windows-${LABEL}.zip"
  rm -f "$ZIPFILE"
  (cd build && zip -q -X "Hachibu-windows-${LABEL}.zip" Hachibu.exe)
  echo "zipped: windows/$ZIPFILE"
  if command -v sha256sum >/dev/null; then
    (cd build && sha256sum "Hachibu-windows-${LABEL}.zip")
  else
    (cd build && shasum -a 256 "Hachibu-windows-${LABEL}.zip")
  fi
fi
