//go:build windows

package w32

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// FileVersionStrings は実行ファイルに埋め込んだバージョン情報（StringFileInfo）から、namesの値を読む
// （ProductName、ProductVersionなど）。Hachibu.exeにはbuild.shがgo-winresで埋め込む。
// 情報が無いファイル、読めない名前はエラー
func FileVersionStrings(path string, names ...string) (map[string]string, error) {
	size, err := windows.GetFileVersionInfoSize(path, nil)
	if err != nil || size == 0 {
		return nil, errors.New("no version information")
	}
	buf := make([]byte, size)
	if err := windows.GetFileVersionInfo(path, 0, size, unsafe.Pointer(&buf[0])); err != nil {
		return nil, errors.New("no version information")
	}
	// 言語とコードページの組（2バイトずつ）。最初の組の表を読む
	var trans unsafe.Pointer
	var transLen uint32
	if err := windows.VerQueryValue(unsafe.Pointer(&buf[0]), `\VarFileInfo\Translation`, unsafe.Pointer(&trans), &transLen); err != nil || transLen < 4 {
		return nil, errors.New("no version translation")
	}
	pair := unsafe.Slice((*uint16)(trans), 2)
	out := map[string]string{}
	for _, name := range names {
		var p unsafe.Pointer
		var n uint32
		sub := fmt.Sprintf(`\StringFileInfo\%04x%04x\%s`, pair[0], pair[1], name)
		if err := windows.VerQueryValue(unsafe.Pointer(&buf[0]), sub, unsafe.Pointer(&p), &n); err != nil || n == 0 {
			return nil, fmt.Errorf("no %s", name)
		}
		// nは終わりのNULを含む文字数
		out[name] = windows.UTF16ToString(unsafe.Slice((*uint16)(p), n))
	}
	return out, nil
}
