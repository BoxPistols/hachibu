//go:build windows

package w32

import (
	"encoding/binary"
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// TaskDialogButton はタスクダイアログに並べるボタン。
type TaskDialogButton struct {
	ID   int32
	Text string
}

// TaskDialogConfig はタスクダイアログ（Windows標準の確認の窓）の中身。Footerには<a href="...">文言</a>でリンクを書ける
type TaskDialogConfig struct {
	Owner           HWND
	Title           string
	MainInstruction string
	Content         string
	Footer          string
	Buttons         []TaskDialogButton
	DefaultButton   int32
	OnHyperlink     func(href string)
}

const (
	tdfEnableHyperlinks        = 0x0001
	tdfAllowDialogCancellation = 0x0008
	tdnHyperlinkClicked        = 3
	// IDCANCEL はEscや閉じるボタンで閉じたとき
	IDCANCEL = 2
)

var (
	taskDialogCallback uintptr
	onHyperlink        func(string)
)

// TaskDialog は窓を出し、押されたボタンの番号を返す（閉じるまで戻らない）。comctl32の6.0が無ければエラー
//
// TASKDIALOGCONFIGは1バイト詰め（pack(1)）の構造体なので、Goの構造体ではなくバイト列に位置を決めて書く
func TaskDialog(cfg TaskDialogConfig) (int32, error) {
	if pTaskDialogIndirect.Find() != nil {
		return 0, errors.New("TaskDialogIndirect is not available")
	}
	if unsafe.Sizeof(uintptr(0)) != 8 {
		return 0, errors.New("only 64-bit builds are supported")
	}
	if taskDialogCallback == 0 {
		taskDialogCallback = syscall.NewCallback(func(hwnd, msg, wParam, lParam, data uintptr) uintptr {
			if msg == tdnHyperlinkClicked && onHyperlink != nil {
				// lParamはリンクの文字列（WCHAR*）。数値のまま受け取ったので、変数の場所を経由してポインタに戻す
				onHyperlink(windows.UTF16PtrToString(*(**uint16)(unsafe.Pointer(&lParam))))
			}
			return 0
		})
	}
	onHyperlink = cfg.OnHyperlink

	// 文字列とボタンの配列は、呼び出しが終わるまでここで持っておく
	keep := []*uint16{}
	str := func(s string) uint64 {
		if s == "" {
			return 0
		}
		p := UTF16(s)
		keep = append(keep, p)
		return uint64(uintptr(unsafe.Pointer(p)))
	}
	buttons := make([]byte, 12*len(cfg.Buttons)+1)
	for i, b := range cfg.Buttons {
		binary.LittleEndian.PutUint32(buttons[12*i:], uint32(b.ID))
		binary.LittleEndian.PutUint64(buttons[12*i+4:], str(b.Text))
	}

	buf := make([]byte, 160)
	le := binary.LittleEndian
	le.PutUint32(buf[0:], 160)
	le.PutUint64(buf[4:], uint64(cfg.Owner))
	le.PutUint64(buf[12:], uint64(ModuleHandle()))
	le.PutUint32(buf[20:], tdfEnableHyperlinks|tdfAllowDialogCancellation)
	le.PutUint64(buf[28:], str(cfg.Title))
	le.PutUint64(buf[44:], str(cfg.MainInstruction))
	le.PutUint64(buf[52:], str(cfg.Content))
	le.PutUint32(buf[60:], uint32(len(cfg.Buttons)))
	if len(cfg.Buttons) > 0 {
		le.PutUint64(buf[64:], uint64(uintptr(unsafe.Pointer(&buttons[0]))))
	}
	le.PutUint32(buf[72:], uint32(cfg.DefaultButton))
	le.PutUint64(buf[132:], str(cfg.Footer))
	le.PutUint64(buf[140:], uint64(taskDialogCallback))

	var pressed int32
	r, _, _ := pTaskDialogIndirect.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&pressed)), 0, 0)
	onHyperlink = nil
	runtimeKeepAlive(keep, buttons, buf)
	if r != 0 {
		return 0, errors.New("TaskDialogIndirect failed")
	}
	return pressed, nil
}
