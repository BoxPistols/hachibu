//go:build windows

// Package w32 は、HachibuのWindows版が使うWin32の関数と構造体だけを宣言する。
// cgoを使わず、DLLの関数を名前で引いて呼ぶ。
package w32

import (
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type (
	HWND      uintptr
	HMENU     uintptr
	HICON     uintptr
	HBITMAP   uintptr
	HDC       uintptr
	HGDIOBJ   uintptr
	HCURSOR   uintptr
	HBRUSH    uintptr
	HFONT     uintptr
	HMONITOR  uintptr
	HINSTANCE uintptr
)

type POINT struct{ X, Y int32 }

type RECT struct{ Left, Top, Right, Bottom int32 }

func (r RECT) Width() int32  { return r.Right - r.Left }
func (r RECT) Height() int32 { return r.Bottom - r.Top }

// Contains は点が矩形の中か（右端と下端は含めない）。
func (r RECT) Contains(p POINT) bool {
	return p.X >= r.Left && p.X < r.Right && p.Y >= r.Top && p.Y < r.Bottom
}

// Intersects は2つの矩形が重なるか。
func (r RECT) Intersects(o RECT) bool {
	return r.Left < o.Right && o.Left < r.Right && r.Top < o.Bottom && o.Top < r.Bottom
}

type SIZE struct{ CX, CY int32 }

type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
	Private uint32
}

type WNDCLASSEXW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   HINSTANCE
	Icon       HICON
	Cursor     HCURSOR
	Background HBRUSH
	MenuName   *uint16
	ClassName  *uint16
	IconSm     HICON
}

type BLENDFUNCTION struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type MONITORINFO struct {
	Size    uint32
	Monitor RECT
	Work    RECT
	Flags   uint32
}

type TRACKMOUSEEVENT struct {
	Size      uint32
	Flags     uint32
	HwndTrack HWND
	HoverTime uint32
}

type MENUITEMINFOW struct {
	Size         uint32
	Mask         uint32
	Type         uint32
	State        uint32
	ID           uint32
	SubMenu      HMENU
	BmpChecked   HBITMAP
	BmpUnchecked HBITMAP
	ItemData     uintptr
	TypeData     *uint16
	Cch          uint32
	BmpItem      HBITMAP
}

type NOTIFYICONDATAW struct {
	Size            uint32
	Wnd             HWND
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            HICON
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Version         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GuidItem        windows.GUID
	BalloonIcon     HICON
}

type TOOLINFOW struct {
	Size     uint32
	Flags    uint32
	Hwnd     HWND
	ID       uintptr
	Rect     RECT
	Inst     HINSTANCE
	Text     *uint16
	LParam   uintptr
	Reserved uintptr
}

type LOGFONTW struct {
	Height         int32
	Width          int32
	Escapement     int32
	Orientation    int32
	Weight         int32
	Italic         byte
	Underline      byte
	StrikeOut      byte
	CharSet        byte
	OutPrecision   byte
	ClipPrecision  byte
	Quality        byte
	PitchAndFamily byte
	FaceName       [32]uint16
}

type NONCLIENTMETRICSW struct {
	Size              uint32
	BorderWidth       int32
	ScrollWidth       int32
	ScrollHeight      int32
	CaptionWidth      int32
	CaptionHeight     int32
	CaptionFont       LOGFONTW
	SmCaptionWidth    int32
	SmCaptionHeight   int32
	SmCaptionFont     LOGFONTW
	MenuWidth         int32
	MenuHeight        int32
	MenuFont          LOGFONTW
	StatusFont        LOGFONTW
	MessageFont       LOGFONTW
	PaddedBorderWidth int32
}

type PAINTSTRUCT struct {
	Hdc         HDC
	Erase       int32
	Paint       RECT
	Restore     int32
	IncUpdate   int32
	RGBReserved [32]byte
}

// 窓とメッセージ
const (
	WS_POPUP        = 0x80000000
	WS_CHILD        = 0x40000000
	WS_VISIBLE      = 0x10000000
	WS_CAPTION      = 0x00C00000
	WS_SYSMENU      = 0x00080000
	WS_CLIPCHILDREN = 0x02000000
	WS_TABSTOP      = 0x00010000
	WS_DISABLED     = 0x08000000

	WS_EX_TOPMOST     = 0x00000008
	WS_EX_TOOLWINDOW  = 0x00000080
	WS_EX_LAYERED     = 0x00080000
	WS_EX_NOACTIVATE  = 0x08000000
	WS_EX_TRANSPARENT = 0x00000020

	WM_NULL            = 0x0000
	WM_DESTROY         = 0x0002
	WM_ACTIVATE        = 0x0006
	WM_SETFOCUS        = 0x0007
	WM_KILLFOCUS       = 0x0008
	WM_PAINT           = 0x000F
	WM_CLOSE           = 0x0010
	WM_QUERYENDSESSION = 0x0011
	WM_ERASEBKGND      = 0x0014
	WM_ENDSESSION      = 0x0016
	WM_SETTINGCHANGE   = 0x001A
	WM_SETCURSOR       = 0x0020
	WM_MOUSEACTIVATE   = 0x0021
	WM_SETFONT         = 0x0030
	WM_NOTIFY          = 0x004E
	WM_CONTEXTMENU     = 0x007B
	WM_DISPLAYCHANGE   = 0x007E
	WM_NCHITTEST       = 0x0084
	WM_KEYDOWN         = 0x0100
	WM_KEYUP           = 0x0101
	WM_CHAR            = 0x0102
	WM_SYSKEYDOWN      = 0x0104
	WM_SYSKEYUP        = 0x0105
	WM_SYSCHAR         = 0x0106
	WM_COMMAND         = 0x0111
	WM_SYSCOMMAND      = 0x0112
	WM_TIMER           = 0x0113
	WM_MENUSELECT      = 0x011F
	WM_CTLCOLORBTN     = 0x0135
	WM_CTLCOLORSTATIC  = 0x0138
	WM_MOUSEMOVE       = 0x0200
	WM_LBUTTONDOWN     = 0x0201
	WM_LBUTTONUP       = 0x0202
	WM_RBUTTONDOWN     = 0x0204
	WM_RBUTTONUP       = 0x0205
	WM_CAPTURECHANGED  = 0x0215
	WM_EXITMENULOOP    = 0x0212
	WM_DPICHANGED      = 0x02E0
	WM_MOUSELEAVE      = 0x02A3
	WM_HOTKEY          = 0x0312
	WM_USER            = 0x0400
	WM_APP             = 0x8000

	SW_HIDE           = 0
	SW_SHOWNORMAL     = 1
	SW_SHOWNOACTIVATE = 4
	SW_SHOW           = 5

	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010
	SWP_SHOWWINDOW = 0x0040

	ULW_ALPHA    = 0x02
	AC_SRC_OVER  = 0x00
	AC_SRC_ALPHA = 0x01

	MA_NOACTIVATE = 3
	TME_LEAVE     = 0x02
	HTCLIENT      = 1

	IDC_ARROW   = 32512
	IDC_SIZEALL = 32646

	MOD_NOREPEAT = 0x4000

	MONITOR_DEFAULTTOPRIMARY = 1
	MONITOR_DEFAULTTONEAREST = 2

	SM_CXSMICON    = 49
	SM_CXMENUCHECK = 71

	GA_ROOT = 2

	VK_SHIFT      = 0x10
	VK_CONTROL    = 0x11
	VK_MENU       = 0x12
	VK_ESCAPE     = 0x1B
	VK_RETURN     = 0x0D
	VK_LWIN       = 0x5B
	VK_RWIN       = 0x5C
	VK_PROCESSKEY = 0xE5

	MAPVK_VK_TO_CHAR = 2

	BS_PUSHBUTTON           = 0x00000000
	BS_DEFPUSHBUTTON        = 0x00000001
	BN_CLICKED              = 0
	SPI_GETNONCLIENTMETRICS = 0x0029
	TRANSPARENT             = 1

	COLOR_WINDOW = 5
)

// メニュー
const (
	MF_STRING    = 0x0000
	MF_GRAYED    = 0x0001
	MF_CHECKED   = 0x0008
	MF_POPUP     = 0x0010
	MF_SEPARATOR = 0x0800

	MIIM_STATE   = 0x0001
	MIIM_ID      = 0x0002
	MIIM_SUBMENU = 0x0004
	MIIM_STRING  = 0x0040
	MIIM_BITMAP  = 0x0080
	MIIM_FTYPE   = 0x0100

	MFT_STRING     = 0x0000
	MFT_RADIOCHECK = 0x0200
	MFT_SEPARATOR  = 0x0800

	MFS_GRAYED  = 0x0003
	MFS_CHECKED = 0x0008

	TPM_RIGHTBUTTON = 0x0002
	TPM_RETURNCMD   = 0x0100
	TPM_NONOTIFY    = 0x0080
	TPM_BOTTOMALIGN = 0x0020
)

// トレイ
const (
	NIM_ADD        = 0
	NIM_MODIFY     = 1
	NIM_DELETE     = 2
	NIM_SETVERSION = 4

	NIF_MESSAGE  = 0x01
	NIF_ICON     = 0x02
	NIF_TIP      = 0x04
	NIF_INFO     = 0x10
	NIF_SHOWTIP  = 0x80
	NIIF_INFO    = 0x01
	NIIF_NOSOUND = 0x10

	NOTIFYICON_VERSION_4 = 4

	NIN_SELECT           = WM_USER + 0
	NIN_KEYSELECT        = WM_USER + 1
	NIN_BALLOONUSERCLICK = WM_USER + 5
)

// ツールチップ
const (
	TTS_ALWAYSTIP      = 0x01
	TTS_NOPREFIX       = 0x02
	TTF_IDISHWND       = 0x0001
	TTF_SUBCLASS       = 0x0010
	TTM_SETDELAYTIME   = WM_USER + 3
	TTM_SETMAXTIPWIDTH = WM_USER + 24
	TTM_POP            = WM_USER + 28
	TTM_ADDTOOLW       = WM_USER + 50
	TTM_UPDATETIPTEXTW = WM_USER + 57
	TTDT_AUTOPOP       = 2
)

const DWMWA_USE_IMMERSIVE_DARK_MODE = 20

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	uxtheme  = windows.NewLazySystemDLL("uxtheme.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	// comctl32はマニフェストで6.0を指定しているので、名前で読ませてWinSxSの版を受け取る
	comctl32 = windows.NewLazyDLL("comctl32.dll")

	pRegisterClassExW              = user32.NewProc("RegisterClassExW")
	pCreateWindowExW               = user32.NewProc("CreateWindowExW")
	pDefWindowProcW                = user32.NewProc("DefWindowProcW")
	pDestroyWindow                 = user32.NewProc("DestroyWindow")
	pShowWindow                    = user32.NewProc("ShowWindow")
	pIsWindowVisible               = user32.NewProc("IsWindowVisible")
	pSetWindowPos                  = user32.NewProc("SetWindowPos")
	pGetWindowRect                 = user32.NewProc("GetWindowRect")
	pGetClientRect                 = user32.NewProc("GetClientRect")
	pGetCursorPos                  = user32.NewProc("GetCursorPos")
	pGetMessageW                   = user32.NewProc("GetMessageW")
	pTranslateMessage              = user32.NewProc("TranslateMessage")
	pDispatchMessageW              = user32.NewProc("DispatchMessageW")
	pPostMessageW                  = user32.NewProc("PostMessageW")
	pSendMessageW                  = user32.NewProc("SendMessageW")
	pPostQuitMessage               = user32.NewProc("PostQuitMessage")
	pLoadCursorW                   = user32.NewProc("LoadCursorW")
	pSetCursor                     = user32.NewProc("SetCursor")
	pUpdateLayeredWindow           = user32.NewProc("UpdateLayeredWindow")
	pGetDC                         = user32.NewProc("GetDC")
	pReleaseDC                     = user32.NewProc("ReleaseDC")
	pCreatePopupMenu               = user32.NewProc("CreatePopupMenu")
	pInsertMenuItemW               = user32.NewProc("InsertMenuItemW")
	pGetMenuItemCount              = user32.NewProc("GetMenuItemCount")
	pTrackPopupMenuEx              = user32.NewProc("TrackPopupMenuEx")
	pDestroyMenu                   = user32.NewProc("DestroyMenu")
	pSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	pGetForegroundWindow           = user32.NewProc("GetForegroundWindow")
	pRegisterHotKey                = user32.NewProc("RegisterHotKey")
	pUnregisterHotKey              = user32.NewProc("UnregisterHotKey")
	pMonitorFromPoint              = user32.NewProc("MonitorFromPoint")
	pMonitorFromRect               = user32.NewProc("MonitorFromRect")
	pGetMonitorInfoW               = user32.NewProc("GetMonitorInfoW")
	pEnumDisplayMonitors           = user32.NewProc("EnumDisplayMonitors")
	pGetDpiForWindow               = user32.NewProc("GetDpiForWindow")
	pGetDpiForSystem               = user32.NewProc("GetDpiForSystem")
	pSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	pGetSystemMetricsForDpi        = user32.NewProc("GetSystemMetricsForDpi")
	pSetCapture                    = user32.NewProc("SetCapture")
	pReleaseCapture                = user32.NewProc("ReleaseCapture")
	pGetCapture                    = user32.NewProc("GetCapture")
	pTrackMouseEvent               = user32.NewProc("TrackMouseEvent")
	pRegisterWindowMessageW        = user32.NewProc("RegisterWindowMessageW")
	pFindWindowW                   = user32.NewProc("FindWindowW")
	pGetKeyState                   = user32.NewProc("GetKeyState")
	pMapVirtualKeyW                = user32.NewProc("MapVirtualKeyW")
	pInvalidateRect                = user32.NewProc("InvalidateRect")
	pBeginPaint                    = user32.NewProc("BeginPaint")
	pEndPaint                      = user32.NewProc("EndPaint")
	pFillRect                      = user32.NewProc("FillRect")
	pSetWindowTextW                = user32.NewProc("SetWindowTextW")
	pEnableWindow                  = user32.NewProc("EnableWindow")
	pMessageBoxW                   = user32.NewProc("MessageBoxW")
	pGetAncestor                   = user32.NewProc("GetAncestor")
	pSystemParametersInfoW         = user32.NewProc("SystemParametersInfoForDpi")
	pSetFocus                      = user32.NewProc("SetFocus")
	pMoveWindow                    = user32.NewProc("MoveWindow")
	pDestroyIcon                   = user32.NewProc("DestroyIcon")
	pAdjustWindowRectExForDpi      = user32.NewProc("AdjustWindowRectExForDpi")
	pGetWindowThreadProcessId      = user32.NewProc("GetWindowThreadProcessId")
	pIsWindow                      = user32.NewProc("IsWindow")
	pDrawTextW                     = user32.NewProc("DrawTextW")

	pCreateCompatibleDC  = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection    = gdi32.NewProc("CreateDIBSection")
	pSelectObject        = gdi32.NewProc("SelectObject")
	pDeleteObject        = gdi32.NewProc("DeleteObject")
	pDeleteDC            = gdi32.NewProc("DeleteDC")
	pCreateSolidBrush    = gdi32.NewProc("CreateSolidBrush")
	pCreateFontIndirectW = gdi32.NewProc("CreateFontIndirectW")
	pSetBkMode           = gdi32.NewProc("SetBkMode")
	pSetTextColor        = gdi32.NewProc("SetTextColor")

	pShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")

	pDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	pSetWindowTheme        = uxtheme.NewProc("SetWindowTheme")

	pTaskDialogIndirect   = comctl32.NewProc("TaskDialogIndirect")
	pInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

// UTF16 はWin32に渡す、NULで終わる文字列。途中のNULは落とす（ファイルから読んだ文字列に混ざっていても落ちないように）
func UTF16(s string) *uint16 {
	return &UTF16Slice(s)[0]
}

// UTF16Slice はNULで終わるUTF-16の配列。長さは文字数＋1で、空にはならない
func UTF16Slice(s string) []uint16 {
	u, err := windows.UTF16FromString(strings.ReplaceAll(s, "\x00", ""))
	if err != nil || len(u) == 0 {
		return []uint16{0}
	}
	return u
}

func boolArg(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

func ModuleHandle() HINSTANCE {
	r, _, _ := pGetModuleHandleW.Call(0)
	return HINSTANCE(r)
}

// NewWndProc はウインドウプロシージャの呼び口を作る。作れる数に上限があるので、窓の種類ごとに1回だけ呼ぶ
func NewWndProc(fn func(hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr) uintptr {
	return syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
		return fn(HWND(hwnd), uint32(msg), wParam, lParam)
	})
}

func RegisterClass(name string, wndProc uintptr, cursor HCURSOR, icon HICON) error {
	wc := WNDCLASSEXW{
		WndProc:   wndProc,
		Instance:  ModuleHandle(),
		Cursor:    cursor,
		Icon:      icon,
		IconSm:    icon,
		ClassName: UTF16(name),
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	r, _, err := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if r == 0 {
		return err
	}
	return nil
}

func CreateWindow(exStyle uint32, class, title string, style uint32, x, y, w, h int32, parent HWND, id uintptr) HWND {
	r, _, _ := pCreateWindowExW.Call(uintptr(exStyle), uintptr(unsafe.Pointer(UTF16(class))), uintptr(unsafe.Pointer(UTF16(title))),
		uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(parent), id, uintptr(ModuleHandle()), 0)
	return HWND(r)
}

func DefWindowProc(hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := pDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func DestroyWindow(hwnd HWND)         { pDestroyWindow.Call(uintptr(hwnd)) }
func ShowWindow(hwnd HWND, cmd int32) { pShowWindow.Call(uintptr(hwnd), uintptr(cmd)) }
func IsWindow(hwnd HWND) bool         { r, _, _ := pIsWindow.Call(uintptr(hwnd)); return r != 0 }
func IsWindowVisible(hwnd HWND) bool  { r, _, _ := pIsWindowVisible.Call(uintptr(hwnd)); return r != 0 }
func SetForegroundWindow(hwnd HWND) bool {
	r, _, _ := pSetForegroundWindow.Call(uintptr(hwnd))
	return r != 0
}
func GetForegroundWindow() HWND           { r, _, _ := pGetForegroundWindow.Call(); return HWND(r) }
func SetFocus(hwnd HWND)                  { pSetFocus.Call(uintptr(hwnd)) }
func EnableWindow(hwnd HWND, enable bool) { pEnableWindow.Call(uintptr(hwnd), boolArg(enable)) }
func SetWindowText(hwnd HWND, text string) {
	pSetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(UTF16(text))))
}
func PostQuitMessage(code int32) { pPostQuitMessage.Call(uintptr(code)) }
func InvalidateRect(hwnd HWND)   { pInvalidateRect.Call(uintptr(hwnd), 0, 1) }
func SetCapture(hwnd HWND)       { pSetCapture.Call(uintptr(hwnd)) }
func ReleaseCapture()            { pReleaseCapture.Call() }
func GetCapture() HWND           { r, _, _ := pGetCapture.Call(); return HWND(r) }
func DestroyMenu(m HMENU)        { pDestroyMenu.Call(uintptr(m)) }
func DestroyIcon(h HICON)        { pDestroyIcon.Call(uintptr(h)) }
func DeleteObject(h HGDIOBJ)     { pDeleteObject.Call(uintptr(h)) }
func SetCursor(c HCURSOR)        { pSetCursor.Call(uintptr(c)) }
func GetAncestor(hwnd HWND, flags uint32) HWND {
	r, _, _ := pGetAncestor.Call(uintptr(hwnd), uintptr(flags))
	return HWND(r)
}

func SetWindowPos(hwnd, after HWND, x, y, w, h int32, flags uint32) {
	pSetWindowPos.Call(uintptr(hwnd), uintptr(after), uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(flags))
}

func MoveWindow(hwnd HWND, x, y, w, h int32) {
	pMoveWindow.Call(uintptr(hwnd), uintptr(x), uintptr(y), uintptr(w), uintptr(h), 1)
}

// HWND_TOPMOST は(HWND)-1
const HWND_TOPMOST = ^HWND(0)

func GetWindowRect(hwnd HWND) RECT {
	var r RECT
	pGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r)))
	return r
}

func GetClientRect(hwnd HWND) RECT {
	var r RECT
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r)))
	return r
}

func GetCursorPos() POINT {
	var p POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p
}

func GetMessage(msg *MSG) int32 {
	r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(msg)), 0, 0, 0)
	return int32(r)
}

func TranslateMessage(msg *MSG) { pTranslateMessage.Call(uintptr(unsafe.Pointer(msg))) }
func DispatchMessage(msg *MSG)  { pDispatchMessageW.Call(uintptr(unsafe.Pointer(msg))) }

func PostMessage(hwnd HWND, msg uint32, wParam, lParam uintptr) bool {
	r, _, _ := pPostMessageW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r != 0
}

func SendMessage(hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := pSendMessageW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

// SendMessagePtr はlParamに構造体のポインタを渡すとき。ポインタを整数にするのはCallの引数の中でだけ行う。
// 呼ぶ前に整数にすると、Goは指す先を生かしておかず、スタックが動けば古い番地を渡してしまう
func SendMessagePtr(hwnd HWND, msg uint32, wParam uintptr, lParam unsafe.Pointer) uintptr {
	r, _, _ := pSendMessageW.Call(uintptr(hwnd), uintptr(msg), wParam, uintptr(lParam))
	return r
}

var pGetFocus = user32.NewProc("GetFocus")

func GetFocus() HWND { r, _, _ := pGetFocus.Call(); return HWND(r) }

func LoadCursor(id uintptr) HCURSOR {
	r, _, _ := pLoadCursorW.Call(0, id)
	return HCURSOR(r)
}

func RegisterWindowMessage(name string) uint32 {
	r, _, _ := pRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(UTF16(name))))
	return uint32(r)
}

func FindWindow(class string) HWND {
	r, _, _ := pFindWindowW.Call(uintptr(unsafe.Pointer(UTF16(class))), 0)
	return HWND(r)
}

func GetDC(hwnd HWND) HDC           { r, _, _ := pGetDC.Call(uintptr(hwnd)); return HDC(r) }
func ReleaseDC(hwnd HWND, dc HDC)   { pReleaseDC.Call(uintptr(hwnd), uintptr(dc)) }
func CreateCompatibleDC(dc HDC) HDC { r, _, _ := pCreateCompatibleDC.Call(uintptr(dc)); return HDC(r) }
func DeleteDC(dc HDC)               { pDeleteDC.Call(uintptr(dc)) }
func SelectObject(dc HDC, obj HGDIOBJ) HGDIOBJ {
	r, _, _ := pSelectObject.Call(uintptr(dc), uintptr(obj))
	return HGDIOBJ(r)
}

// CreateDIB は上から下へ並ぶ32bppのDIBを作る。中身は0で埋まっている
func CreateDIB(w, h int32) (HBITMAP, unsafe.Pointer) {
	bi := BITMAPINFOHEADER{Width: w, Height: -h, Planes: 1, BitCount: 32}
	bi.Size = uint32(unsafe.Sizeof(bi))
	var bits unsafe.Pointer
	r, _, _ := pCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	return HBITMAP(r), bits
}

// UpdateLayered は乗算済みアルファのDIBで、層になった窓の中身と位置を置き換える。alphaは窓全体の不透明度（0〜255）
func UpdateLayered(hwnd HWND, pos POINT, size SIZE, bmp HBITMAP, alpha byte) bool {
	screen := GetDC(0)
	defer ReleaseDC(0, screen)
	mem := CreateCompatibleDC(screen)
	defer DeleteDC(mem)
	old := SelectObject(mem, HGDIOBJ(bmp))
	defer SelectObject(mem, old)
	src := POINT{}
	blend := BLENDFUNCTION{BlendOp: AC_SRC_OVER, SourceConstantAlpha: alpha, AlphaFormat: AC_SRC_ALPHA}
	r, _, _ := pUpdateLayeredWindow.Call(uintptr(hwnd), uintptr(screen), uintptr(unsafe.Pointer(&pos)), uintptr(unsafe.Pointer(&size)),
		uintptr(mem), uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&blend)), ULW_ALPHA)
	return r != 0
}

func CreatePopupMenu() HMENU {
	r, _, _ := pCreatePopupMenu.Call()
	return HMENU(r)
}

func InsertMenuItem(menu HMENU, info *MENUITEMINFOW) {
	info.Size = uint32(unsafe.Sizeof(*info))
	n, _, _ := pGetMenuItemCount.Call(uintptr(menu))
	pInsertMenuItemW.Call(uintptr(menu), n, 1, uintptr(unsafe.Pointer(info)))
}

// TrackPopupMenu はメニューを開き、選ばれた項目の番号を返す（選ばれなければ0）。閉じるまで戻らない
func TrackPopupMenu(menu HMENU, flags uint32, x, y int32, owner HWND) uint32 {
	r, _, _ := pTrackPopupMenuEx.Call(uintptr(menu), uintptr(flags), uintptr(x), uintptr(y), uintptr(owner), 0)
	return uint32(r)
}

func RegisterHotKey(hwnd HWND, id int32, mods, vk uint32) bool {
	r, _, _ := pRegisterHotKey.Call(uintptr(hwnd), uintptr(id), uintptr(mods), uintptr(vk))
	return r != 0
}

func UnregisterHotKey(hwnd HWND, id int32) {
	pUnregisterHotKey.Call(uintptr(hwnd), uintptr(id))
}

func MonitorFromPoint(p POINT, flags uint32) HMONITOR {
	// POINTは値渡しで、64ビットでは1つのレジスタに詰める
	r, _, _ := pMonitorFromPoint.Call(uintptr(uint32(p.X))|uintptr(uint32(p.Y))<<32, uintptr(flags))
	return HMONITOR(r)
}

func MonitorFromRect(rc RECT, flags uint32) HMONITOR {
	r, _, _ := pMonitorFromRect.Call(uintptr(unsafe.Pointer(&rc)), uintptr(flags))
	return HMONITOR(r)
}

func GetMonitorInfo(m HMONITOR) (MONITORINFO, bool) {
	var mi MONITORINFO
	mi.Size = uint32(unsafe.Sizeof(mi))
	r, _, _ := pGetMonitorInfoW.Call(uintptr(m), uintptr(unsafe.Pointer(&mi)))
	return mi, r != 0
}

// 画面を数える呼び口は1つだけ作って使い回す（NewCallbackで作れる数には上限がある）。UIのスレッドからだけ呼ぶ
var (
	monitorsCallback uintptr
	collectedAreas   []RECT
)

// WorkAreas はすべての画面の作業領域（タスクバーを除いた範囲）。最初が主画面
func WorkAreas() []RECT {
	if monitorsCallback == 0 {
		monitorsCallback = syscall.NewCallback(func(m, dc, rc, data uintptr) uintptr {
			if mi, ok := GetMonitorInfo(HMONITOR(m)); ok {
				if mi.Flags&1 != 0 {
					collectedAreas = append([]RECT{mi.Work}, collectedAreas...)
				} else {
					collectedAreas = append(collectedAreas, mi.Work)
				}
			}
			return 1
		})
	}
	collectedAreas = nil
	pEnumDisplayMonitors.Call(0, 0, monitorsCallback, 0)
	areas := collectedAreas
	collectedAreas = nil
	return areas
}

func GetDpiForWindow(hwnd HWND) uint32 {
	if pGetDpiForWindow.Find() != nil {
		return 96
	}
	r, _, _ := pGetDpiForWindow.Call(uintptr(hwnd))
	if r == 0 {
		return 96
	}
	return uint32(r)
}

func GetDpiForSystem() uint32 {
	if pGetDpiForSystem.Find() != nil {
		return 96
	}
	r, _, _ := pGetDpiForSystem.Call()
	if r == 0 {
		return 96
	}
	return uint32(r)
}

func GetSystemMetricsForDpi(index int32, dpi uint32) int32 {
	r, _, _ := pGetSystemMetricsForDpi.Call(uintptr(index), uintptr(dpi))
	return int32(r)
}

// SetPerMonitorDPIAware はマニフェストが無い起動（go runなど）でも、画面ごとの拡大率で描けるようにする
func SetPerMonitorDPIAware() {
	if pSetProcessDpiAwarenessContext.Find() == nil {
		pSetProcessDpiAwarenessContext.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = -4
	}
}

func TrackMouseLeave(hwnd HWND) {
	t := TRACKMOUSEEVENT{Flags: TME_LEAVE, HwndTrack: hwnd}
	t.Size = uint32(unsafe.Sizeof(t))
	pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&t)))
}

func KeyDown(vk int32) bool {
	r, _, _ := pGetKeyState.Call(uintptr(vk))
	return int16(r) < 0
}

// KeyChar は仮想キーコードに対応する、今のキーボード配列での文字（修飾キーなし）。無ければ空
func KeyChar(vk uint32) string {
	r, _, _ := pMapVirtualKeyW.Call(uintptr(vk), MAPVK_VK_TO_CHAR)
	c := rune(r & 0x7FFFFFFF)
	if c == 0 {
		return ""
	}
	return string(c)
}

func BeginPaint(hwnd HWND, ps *PAINTSTRUCT) HDC {
	r, _, _ := pBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(ps)))
	return HDC(r)
}

func EndPaint(hwnd HWND, ps *PAINTSTRUCT) { pEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(ps))) }

func FillRect(dc HDC, rc RECT, brush HBRUSH) {
	pFillRect.Call(uintptr(dc), uintptr(unsafe.Pointer(&rc)), uintptr(brush))
}

func CreateSolidBrush(rgb uint32) HBRUSH {
	r, _, _ := pCreateSolidBrush.Call(uintptr(rgb))
	return HBRUSH(r)
}

func CreateFont(lf *LOGFONTW) HFONT {
	r, _, _ := pCreateFontIndirectW.Call(uintptr(unsafe.Pointer(lf)))
	return HFONT(r)
}

func SetBkMode(dc HDC, mode int32)    { pSetBkMode.Call(uintptr(dc), uintptr(mode)) }
func SetTextColor(dc HDC, rgb uint32) { pSetTextColor.Call(uintptr(dc), uintptr(rgb)) }

// DrawText のflags: DT_WORDBREAK=0x10, DT_CALCRECT=0x400, DT_NOPREFIX=0x800, DT_CENTER=1, DT_VCENTER=4, DT_SINGLELINE=0x20
func DrawText(dc HDC, text string, rc *RECT, flags uint32) int32 {
	s := UTF16Slice(text)
	r, _, _ := pDrawTextW.Call(uintptr(dc), uintptr(unsafe.Pointer(&s[0])), uintptr(len(s)-1), uintptr(unsafe.Pointer(rc)), uintptr(flags))
	return int32(r)
}

// MessageFont は画面の文言に使う標準の字体（「設定」と同じSegoe UI、日本語ならYu Gothic UIなど）を、指定のDPIの大きさで返す
func MessageFont(dpi uint32) LOGFONTW {
	var m NONCLIENTMETRICSW
	m.Size = uint32(unsafe.Sizeof(m))
	if pSystemParametersInfoW.Find() == nil {
		pSystemParametersInfoW.Call(SPI_GETNONCLIENTMETRICS, uintptr(m.Size), uintptr(unsafe.Pointer(&m)), 0, uintptr(dpi))
	}
	return m.MessageFont
}

func MessageBox(owner HWND, text, title string, flags uint32) int32 {
	r, _, _ := pMessageBoxW.Call(uintptr(owner), uintptr(unsafe.Pointer(UTF16(text))), uintptr(unsafe.Pointer(UTF16(title))), uintptr(flags))
	return int32(r)
}

func ShellNotifyIcon(msg uint32, data *NOTIFYICONDATAW) bool {
	data.Size = uint32(unsafe.Sizeof(*data))
	r, _, _ := pShellNotifyIconW.Call(uintptr(msg), uintptr(unsafe.Pointer(data)))
	return r != 0
}

// SetDarkTitleBar は窓の枠を暗い色にする（Windows 10 20H1以降）
func SetDarkTitleBar(hwnd HWND, dark bool) {
	if pDwmSetWindowAttribute.Find() != nil {
		return
	}
	v := int32(boolArg(dark))
	pDwmSetWindowAttribute.Call(uintptr(hwnd), DWMWA_USE_IMMERSIVE_DARK_MODE, uintptr(unsafe.Pointer(&v)), 4)
}

// SetDarkControl はボタンなどの部品を暗い外観で描かせる
func SetDarkControl(hwnd HWND) {
	if pSetWindowTheme.Find() != nil {
		return
	}
	pSetWindowTheme.Call(uintptr(hwnd), uintptr(unsafe.Pointer(UTF16("DarkMode_Explorer"))), 0)
}

func AdjustWindowRect(rc RECT, style, exStyle uint32, dpi uint32) RECT {
	pAdjustWindowRectExForDpi.Call(uintptr(unsafe.Pointer(&rc)), uintptr(style), 0, uintptr(exStyle), uintptr(dpi))
	return rc
}

// InitCommonControls はツールチップなどの部品の窓の種類を使えるようにする
func InitCommonControls() {
	type initCommonControlsEx struct{ Size, ICC uint32 }
	icc := initCommonControlsEx{Size: 8, ICC: 0x8 | 0x4000} // ICC_BAR_CLASSES（ツールチップ）、ICC_STANDARD_CLASSES
	if pInitCommonControlsEx.Find() == nil {
		pInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
	}
}

func ProcessIDOfWindow(hwnd HWND) uint32 {
	var pid uint32
	pGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	return pid
}

func LoWord(v uintptr) uint16 { return uint16(v & 0xFFFF) }
func HiWord(v uintptr) uint16 { return uint16((v >> 16) & 0xFFFF) }

// PointFromLParam はマウスのメッセージの座標（符号付き16ビットが2つ）を読む
func PointFromLParam(l uintptr) POINT {
	return POINT{X: int32(int16(l & 0xFFFF)), Y: int32(int16((l >> 16) & 0xFFFF))}
}

var pLoadImageW = user32.NewProc("LoadImageW")

// LoadAppIcon は実行ファイルに埋め込んだアイコン（番号1）。埋め込んでいなければ0
func LoadAppIcon() HICON {
	r, _, _ := pLoadImageW.Call(uintptr(ModuleHandle()), 1, 1, 0, 0, 0x0040|0x8000) // IMAGE_ICON, LR_DEFAULTSIZE|LR_SHARED
	return HICON(r)
}

var (
	shcore            = windows.NewLazySystemDLL("shcore.dll")
	pGetDpiForMonitor = shcore.NewProc("GetDpiForMonitor")
)

// DpiForMonitor は画面の拡大率（96が100%）。窓を置く前に、置く先の画面の大きさで描くために使う
func DpiForMonitor(m HMONITOR) uint32 {
	if pGetDpiForMonitor.Find() != nil {
		return GetDpiForSystem()
	}
	var x, y uint32
	if r, _, _ := pGetDpiForMonitor.Call(uintptr(m), 0, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y))); r != 0 || x == 0 {
		return GetDpiForSystem()
	}
	return x
}

// CW_USEDEFAULT は窓の位置と大きさをWindowsに任せる値
const CW_USEDEFAULT = -0x80000000

// AddTooltip はツールチップの部品を作り、窓全体に結び付ける
func AddTooltip(owner HWND, maxWidth int32) HWND {
	tip := CreateWindow(WS_EX_TOPMOST, "tooltips_class32", "", WS_POPUP|TTS_ALWAYSTIP|TTS_NOPREFIX,
		CW_USEDEFAULT, CW_USEDEFAULT, CW_USEDEFAULT, CW_USEDEFAULT, owner, 0)
	if tip == 0 {
		return 0
	}
	ti := TOOLINFOW{Flags: TTF_IDISHWND | TTF_SUBCLASS, Hwnd: owner, ID: uintptr(owner), Text: UTF16("")}
	ti.Size = uint32(unsafe.Sizeof(ti))
	SendMessagePtr(tip, TTM_ADDTOOLW, 0, unsafe.Pointer(&ti))
	SendMessage(tip, TTM_SETMAXTIPWIDTH, 0, uintptr(maxWidth))
	SendMessage(tip, TTM_SETDELAYTIME, TTDT_AUTOPOP, 30000)
	return tip
}

// SetTooltipText はツールチップの文言を差し替える（部品が文字列を写し取る）
func SetTooltipText(tip, owner HWND, text string) {
	ti := TOOLINFOW{Hwnd: owner, ID: uintptr(owner), Text: UTF16(text)}
	ti.Size = uint32(unsafe.Sizeof(ti))
	SendMessagePtr(tip, TTM_UPDATETIPTEXTW, 0, unsafe.Pointer(&ti))
}

var pBitBlt = gdi32.NewProc("BitBlt")

// BitBlt は描き終えた絵を窓へ写す
func BitBlt(dst HDC, x, y, w, h int32, src HDC) {
	pBitBlt.Call(uintptr(dst), uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(src), 0, 0, 0x00CC0020) // SRCCOPY
}

// runtimeKeepAlive は、ポインタを数値にして渡したものを、呼び出しが終わるまで回収させない
func runtimeKeepAlive(v ...any) {
	runtime.KeepAlive(v)
}

// AllowDarkMenus は、Windowsの「アプリモード」が暗いときにメニューも暗い色で描かせる。
// uxthemeの番号でしか引けない関数（SetPreferredAppMode、FlushMenuThemes）を使う。Windows 10 1903より前や、引けないときは何もしない
func AllowDarkMenus() {
	h, err := windows.LoadLibraryEx("uxtheme.dll", 0, windows.LOAD_LIBRARY_SEARCH_SYSTEM32)
	if err != nil {
		return
	}
	setPreferredAppMode, err := windows.GetProcAddressByOrdinal(h, 135)
	if err != nil {
		return
	}
	syscall.SyscallN(setPreferredAppMode, 1) // AllowDark: アプリモードに合わせる
	if flush, err := windows.GetProcAddressByOrdinal(h, 136); err == nil {
		syscall.SyscallN(flush)
	}
}
