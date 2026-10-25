#!/bin/sh
# macOS 13 (Intel) 向けに HachibuLite.app をビルドする（Hachibu の軽量版。lite/ の中で完結する）
set -e
cd "$(dirname "$0")"
app=HachibuLite.app
# リリースのタグ（lite-v0.3.0 など）と揃える。新しい版の確認はこの値と比べる。TokenBar の 0.2.0 から続ける
version=0.3.0
rm -rf "$app" && mkdir -p "$app/Contents/MacOS"
swiftc -O -target x86_64-apple-macos13 HachibuLite.swift -o "$app/Contents/MacOS/HachibuLite"
cat > "$app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>HachibuLite</string>
<key>CFBundleIdentifier</key><string>io.github.boxpistols.hachibu.lite</string>
<key>CFBundleName</key><string>Hachibu Lite</string>
<key>CFBundleShortVersionString</key><string>$version</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>LSMinimumSystemVersion</key><string>13.0</string>
<key>LSUIElement</key><true/>
</dict></plist>
PLIST
codesign --force -s - "$app"
echo "built $app"
