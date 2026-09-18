#!/bin/bash
# テストをビルドしてから実行する。
#
# Command Line Toolsだけの環境では、SwiftPMがSwift Testingのマクロのプラグイン（plugins/testing/libTestingMacros.dylib）を
# コンパイラに渡さないことがあり、テストのビルドが「plugin for module 'TestingMacros' not found」で失敗する。
# プラグインの置き場所を明示して渡す。Xcodeが入っている環境でも害はない。
set -euo pipefail
cd "$(dirname "$0")/.."

PLUGINS="$(xcode-select -p)/usr/lib/swift/host/plugins/testing"
if [ -d "$PLUGINS" ]; then
  swift build --build-tests -Xswiftc -plugin-path -Xswiftc "$PLUGINS" "$@"
else
  swift build --build-tests "$@"
fi
swift test --skip-build "$@"
