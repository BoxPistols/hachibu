//go:build windows

// Hachibu for Windows: Claude Codeのモデル、effort、使用率を、画面の最前面の小さなバーに出し続ける。
// ビルドはbuild.sh（アイコンとマニフェストを埋め込み、バージョン番号を入れる）。
package main

import (
	"os"

	"github.com/BoxPistols/hachibu/windows/internal/app"
)

func main() {
	os.Exit(app.Run())
}
