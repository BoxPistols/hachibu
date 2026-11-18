//go:build windows

package app

import (
	"sync"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

// 窓とメッセージはすべて1つのスレッド（mainのゴルーチン、OSのスレッドに固定）で扱う。
// 通信やファイルの読み取りは別のゴルーチンで行い、結果はPostでUIのスレッドへ渡す。

const (
	// msgRun はPostで積んだ処理を流す合図
	msgRun = w32.WM_APP + 1
	// msgTray はトレイのアイコンからの知らせ
	msgTray = w32.WM_APP + 2
	// msgSummon は後から開いたコピーから、帯を出すようにという知らせ
	msgSummon = w32.WM_APP + 3
)

// mainClass はアプリの主な窓（見えない）の種類の名前。後から開いたコピーはこの名前で先に動いている方を探す
const mainClass = "HachibuWindowsMain"

var (
	mainHwnd w32.HWND
	queueMu  sync.Mutex
	queue    []func()
	handlers = map[w32.HWND]func(msg uint32, wParam, lParam uintptr) (uintptr, bool){}
	wndProc  uintptr
)

// Post はUIのスレッドで処理を動かす。どのゴルーチンから呼んでもよい
func Post(fn func()) {
	queueMu.Lock()
	queue = append(queue, fn)
	queueMu.Unlock()
	w32.PostMessage(mainHwnd, msgRun, 0, 0)
}

func drainQueue() {
	queueMu.Lock()
	pending := queue
	queue = nil
	queueMu.Unlock()
	for _, fn := range pending {
		fn()
	}
}

// Delay は後で1回だけ動かす処理。もう一度Scheduleするか、Cancelすると前の予定は消える。UIのスレッドからだけ使う
type Delay struct{ gen int }

func (d *Delay) Schedule(after time.Duration, fn func()) {
	d.gen++
	gen := d.gen
	time.AfterFunc(after, func() {
		Post(func() {
			if d.gen == gen {
				fn()
			}
		})
	})
}

func (d *Delay) Cancel() { d.gen++ }

// After は後で1回だけ動かす（取り消さない処理）。
func After(after time.Duration, fn func()) {
	time.AfterFunc(after, func() { Post(fn) })
}

// Every は一定の間隔で動かし続ける。
func Every(interval time.Duration, fn func()) {
	go func() {
		t := time.NewTicker(interval)
		for range t.C {
			Post(fn)
		}
	}()
}

// windowProc はすべての窓のメッセージを受け、窓ごとの受け手に渡す。
func windowProc(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if h, ok := handlers[hwnd]; ok {
		if r, handled := h(msg, wParam, lParam); handled {
			return r
		}
	}
	return w32.DefWindowProc(hwnd, msg, wParam, lParam)
}

// registerClass は窓の種類を登録する。どの種類も同じwindowProcを使う
func registerClass(name string, cursor w32.HCURSOR) {
	if wndProc == 0 {
		wndProc = w32.NewWndProc(windowProc)
	}
	w32.RegisterClass(name, wndProc, cursor, appIcon())
}

// newWindow は窓を作り、受け手を結び付ける。作っている最中に届くメッセージ（WM_CREATEなど）は既定の処理に任せる
func newWindow(exStyle uint32, class, title string, style uint32, x, y, w, h int32, parent w32.HWND,
	handler func(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) (uintptr, bool)) w32.HWND {
	hwnd := w32.CreateWindow(exStyle, class, title, style, x, y, w, h, parent, 0)
	if hwnd != 0 && handler != nil {
		handlers[hwnd] = func(msg uint32, wParam, lParam uintptr) (uintptr, bool) { return handler(hwnd, msg, wParam, lParam) }
	}
	return hwnd
}

func forgetWindow(hwnd w32.HWND) {
	delete(handlers, hwnd)
}

// keyHook は記録の窓が開いている間、その窓へ向かうキー入力を先に受け取る（ボタンやAltのメニューに渡さない）
var keyHook func(msg *w32.MSG) bool

func messageLoop() {
	var msg w32.MSG
	for w32.GetMessage(&msg) > 0 {
		if keyHook != nil && keyHook(&msg) {
			continue
		}
		w32.TranslateMessage(&msg)
		w32.DispatchMessage(&msg)
	}
}

var cachedIcon w32.HICON

// appIcon は実行ファイルに埋め込んだアイコン（リソースの番号は go-winres が付ける 1 番）
func appIcon() w32.HICON {
	if cachedIcon == 0 {
		cachedIcon = w32.LoadAppIcon()
	}
	return cachedIcon
}
