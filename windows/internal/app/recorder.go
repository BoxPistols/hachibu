//go:build windows

package app

import (
	"math"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

const recorderClass = "HachibuWindowsRecorder"

// 保存できたことを読めるだけ表示してから閉じる
const recorderCloseDelay = 900 * time.Millisecond

const (
	btnDisable = 1
	btnCancel  = 2
	btnSave    = 3
)

// Recorder は「ショートカットを変更…」で開く小さな窓。押している修飾キーをその場で表示し、キーを押した瞬間に登録を試す。
//
// 登録できても押したときに届くとは限らない（Windowsや他のアプリが先に受け取ることがある）。そのため
// Windowsが使う組み合わせは登録の前に弾き、登録後はもう一度押してもらって、届いたら「保存」を押せるようにする。
// 状態の判断はcore.ShortcutRecordingが持つ。この窓はキー入力を渡し、結果を表示するだけにする。
// キー入力を受けるためにこの窓だけは前面に出す。閉じたら元の窓へ戻す
type Recorder struct {
	hotKeys       *core.HotKeyCenter
	title         func() string
	usedElsewhere func() *core.Shortcut

	recording core.ShortcutRecording
	hwnd      w32.HWND
	buttons   [4]w32.HWND
	previous  w32.HWND
	live      string
	message   string
	closing   Delay

	scale   float32
	dark    bool
	font    w32.HFONT
	bigFont w32.HFONT
	bgBrush w32.HBRUSH
	layout  recorderLayout
}

type recorderLayout struct {
	width, height int32
	current       w32.RECT
	box           w32.RECT
	message       w32.RECT
	buttonsTop    int32
}

func newRecorder(hotKeys *core.HotKeyCenter, title func() string, usedElsewhere func() *core.Shortcut) *Recorder {
	registerClass(recorderClass, w32.LoadCursor(w32.IDC_ARROW))
	return &Recorder{hotKeys: hotKeys, title: title, usedElsewhere: usedElsewhere}
}

// Show は窓を開く。previousは閉じたときに前面へ戻す窓（メニューを開く前に使っていた窓）
func (r *Recorder) Show(previous w32.HWND) {
	if r.hwnd != 0 {
		w32.SetForegroundWindow(r.hwnd)
		return
	}
	r.previous = previous
	r.recording = core.ShortcutRecording{}
	r.hotKeys.Suspend()
	// 窓を開いている間に届いたショートカットは、帯の呼び出しではなく確認に回す
	r.hotKeys.Interceptor = func() bool {
		if r.recording.Arrived() {
			r.refresh()
		}
		return true
	}
	r.live = ""

	cursor := w32.GetCursorPos()
	monitor := w32.MonitorFromPoint(cursor, w32.MONITOR_DEFAULTTONEAREST)
	r.scale = float32(w32.DpiForMonitor(monitor)) / 96
	r.dark = appsAreDark()
	r.makeFonts()

	style := uint32(w32.WS_POPUP | w32.WS_CAPTION | w32.WS_SYSMENU | w32.WS_CLIPCHILDREN)
	r.hwnd = newWindow(w32.WS_EX_TOPMOST, recorderClass, r.title(), style, 0, 0, 100, 100, 0, r.handle)
	w32.SetDarkTitleBar(r.hwnd, r.dark)
	s := L
	for id, text := range map[int]string{btnDisable: s.RecorderDisable, btnCancel: s.RecorderCancel, btnSave: s.RecorderSave} {
		b := w32.CreateWindow(0, "BUTTON", text, w32.WS_CHILD|w32.WS_VISIBLE|w32.BS_PUSHBUTTON, 0, 0, 10, 10, r.hwnd, uintptr(id))
		w32.SendMessage(b, w32.WM_SETFONT, uintptr(r.font), 1)
		if r.dark {
			w32.SetDarkControl(b)
		}
		r.buttons[id] = b
	}
	keyHook = r.hook
	r.refresh()

	// 画面の中央に置く
	if mi, ok := w32.GetMonitorInfo(monitor); ok {
		rc := w32.GetWindowRect(r.hwnd)
		x := mi.Work.Left + (mi.Work.Width()-rc.Width())/2
		y := mi.Work.Top + (mi.Work.Height()-rc.Height())/2
		w32.SetWindowPos(r.hwnd, 0, x, y, 0, 0, w32.SWP_NOSIZE|w32.SWP_NOZORDER)
	}
	w32.ShowWindow(r.hwnd, w32.SW_SHOW)
	w32.SetForegroundWindow(r.hwnd)
	w32.SetFocus(r.hwnd)
}

func (r *Recorder) px(v float64) int32 { return int32(math.Round(v * float64(r.scale))) }

func (r *Recorder) makeFonts() {
	r.freeFonts()
	dpi := uint32(math.Round(float64(r.scale) * 96))
	lf := w32.MessageFont(dpi)
	if lf.Height == 0 {
		lf.Height = -r.px(12)
		copy(lf.FaceName[:], []uint16{'S', 'e', 'g', 'o', 'e', ' ', 'U', 'I', 0})
	}
	r.font = w32.CreateFont(&lf)
	big := lf
	big.Height = -r.px(24)
	big.Weight = 600
	r.bigFont = w32.CreateFont(&big)
	r.bgBrush = w32.CreateSolidBrush(colorRef(r.background()))
}

func (r *Recorder) freeFonts() {
	for _, h := range []uintptr{uintptr(r.font), uintptr(r.bigFont), uintptr(r.bgBrush)} {
		if h != 0 {
			w32.DeleteObject(w32.HGDIOBJ(h))
		}
	}
	r.font, r.bigFont, r.bgBrush = 0, 0, 0
}

func (r *Recorder) background() [3]byte {
	if r.dark {
		return [3]byte{32, 32, 32}
	}
	return [3]byte{249, 249, 249}
}

func colorRef(c [3]byte) uint32 { return uint32(c[0]) | uint32(c[1])<<8 | uint32(c[2])<<16 }

// currentModifiers はいま押している修飾キー。キー入力のメッセージを処理している時点の状態を返す
func currentModifiers() core.Modifiers {
	var m core.Modifiers
	if w32.KeyDown(w32.VK_CONTROL) {
		m |= core.ModCtrl
	}
	if w32.KeyDown(w32.VK_MENU) {
		m |= core.ModAlt
	}
	if w32.KeyDown(w32.VK_SHIFT) {
		m |= core.ModShift
	}
	return m
}

// hook はメッセージループから、記録の窓へ向かうキー入力を先に受け取る。ボタンやAltのメニューには渡さない
func (r *Recorder) hook(msg *w32.MSG) bool {
	if r.hwnd == 0 || (msg.Hwnd != r.hwnd && w32.GetAncestor(msg.Hwnd, w32.GA_ROOT) != r.hwnd) {
		return false
	}
	switch msg.Message {
	case w32.WM_KEYDOWN, w32.WM_SYSKEYDOWN:
		if r.buttonKey(uint32(msg.WParam)) {
			return true
		}
		r.keyDown(uint32(msg.WParam))
		return true
	case w32.WM_KEYUP, w32.WM_SYSKEYUP:
		r.showHeld()
		return true
	case w32.WM_CHAR, w32.WM_SYSCHAR:
		return true
	}
	return false
}

// buttonKey はキーボードだけでボタンを使うためのキー。修飾キーなしのTabとShift+Tabでボタンを移り、
// ボタンにいる間はSpaceかEnterでそのボタンを押す。どれも修飾キーが要るショートカットにはならないので、記録の邪魔をしない
func (r *Recorder) buttonKey(vk uint32) bool {
	mods := currentModifiers()
	focused := w32.GetFocus()
	switch {
	case vk == core.VKTab && (mods == 0 || mods == core.ModShift):
		order := []w32.HWND{r.buttons[btnDisable], r.buttons[btnCancel]}
		if r.recording.CanSave() {
			order = append(order, r.buttons[btnSave])
		}
		idx := -1
		for i, b := range order {
			if b == focused {
				idx = i
			}
		}
		var next int
		switch {
		case idx < 0 && mods == core.ModShift:
			next = len(order) - 1
		case idx < 0:
			next = 0
		case mods == core.ModShift:
			next = (idx - 1 + len(order)) % len(order)
		default:
			next = (idx + 1) % len(order)
		}
		// フォーカスの枠を見えるようにする（マウスで開いた窓では、Windowsが既定で隠している）
		w32.SendMessage(r.hwnd, wmChangeUIState, uisClearHideFocus, 0)
		w32.SetFocus(order[next])
		return true
	case (vk == core.VKSpace || vk == core.VKReturn) && mods == 0:
		for _, b := range r.buttons[1:] {
			if b == focused {
				w32.SendMessage(b, bmClick, 0, 0)
				return true
			}
		}
	}
	return false
}

const (
	bmClick         = 0x00F5
	wmChangeUIState = 0x0127
	// MAKEWPARAM(UIS_CLEAR, UISF_HIDEFOCUS)
	uisClearHideFocus = 2 | 1<<16
)

func isModifierKey(vk uint32) bool {
	return (vk >= 0x10 && vk <= 0x12) || (vk >= 0xA0 && vk <= 0xA5)
}

// showHeld は押している修飾キーだけを表示する（キーを押す前の途中経過）
func (r *Recorder) showHeld() {
	switch r.recording.Phase.Kind {
	case core.Verifying, core.Verified, core.Saved:
		// 確かめている間は、確かめている組み合わせを出したままにする（別の組み合わせはキーを押した時点で切り替わる）
		return
	}
	r.live = currentModifiers().Prefix()
	w32.InvalidateRect(r.hwnd)
}

func (r *Recorder) keyDown(vk uint32) {
	if r.recording.IsSaved() {
		return
	}
	if isModifierKey(vk) {
		r.showHeld()
		return
	}
	// Windowsキー、IMEが処理中のキー、Caps Lockは組み合わせに使わない
	if vk == w32.VK_LWIN || vk == w32.VK_RWIN || vk == w32.VK_PROCESSKEY || vk == 0x14 {
		return
	}
	held := currentModifiers()
	// 修飾キーなしのEscはキャンセルと同じ
	if vk == w32.VK_ESCAPE && held == 0 {
		r.cancel()
		return
	}
	// 修飾キーなしのEnterは「保存」と同じ
	if vk == w32.VK_RETURN && held == 0 && r.recording.CanSave() {
		r.save()
		return
	}
	candidate := core.Shortcut{VK: vk, Mods: held, Label: core.KeyLabel(vk, w32.KeyChar(vk))}
	reservedID, _ := core.Conflict(candidate)
	other := r.usedElsewhere()
	r.recording.Press(candidate, reservedID, other != nil && other.Same(candidate), r.hotKeys)
	r.refresh()
}

func (r *Recorder) save() {
	if !r.recording.Save(r.hotKeys) {
		return
	}
	r.refresh()
	r.closing.Schedule(recorderCloseDelay, r.close)
}

func (r *Recorder) cancel() {
	r.recording.Cancel(r.hotKeys)
	r.close()
}

// Close は開いていれば、キャンセルと同じく閉じる（もう一方の記録の窓を開く前に呼ぶ）
func (r *Recorder) Close() {
	if r.hwnd == 0 {
		return
	}
	if r.recording.IsSaved() {
		r.close()
	} else {
		r.cancel()
	}
}

func (r *Recorder) turnOff() {
	r.recording.TurnOff(r.hotKeys)
	r.close()
}

func (r *Recorder) close() {
	if r.hwnd == 0 {
		return
	}
	r.closing.Cancel()
	r.hotKeys.Interceptor = nil
	keyHook = nil
	hwnd := r.hwnd
	r.hwnd = 0
	forgetWindow(hwnd)
	w32.DestroyWindow(hwnd)
	r.freeFonts()
	if r.previous != 0 && w32.IsWindow(r.previous) {
		w32.SetForegroundWindow(r.previous)
	}
	r.previous = 0
}

// refresh は状態から表示を作る。「現在」は保存済みの値なので、確かめ終わるまでは窓を開く前のまま
func (r *Recorder) refresh() {
	s := L
	phase := r.recording.Phase
	switch phase.Kind {
	case core.Waiting:
		r.message = s.RecorderPrompt
	case core.Rejected:
		r.live = phase.Candidate.Display()
		switch phase.Rejection {
		case core.NeedsModifier:
			r.message = s.RecorderRejected
		case core.Reserved:
			r.message = s.RecorderReserved(phase.Candidate.Display(), s.SystemShortcutName(phase.ReservedID))
		default:
			r.message = s.RecorderTaken(phase.Candidate.Display())
		}
	case core.Verifying:
		r.live = phase.Candidate.Display()
		r.message = s.RecorderVerify(phase.Candidate.Display())
		if core.IsAppShortcutProne(phase.Candidate) {
			r.message += "\n" + s.RecorderAppProne
		}
	case core.Verified:
		r.live = phase.Candidate.Display()
		r.message = s.RecorderVerified(phase.Candidate.Display())
	case core.Saved:
		r.live = phase.Candidate.Display()
		r.message = s.RecorderSaved(phase.Candidate.Display())
	}
	w32.EnableWindow(r.buttons[btnSave], r.recording.CanSave())
	r.relayout()
	w32.InvalidateRect(r.hwnd)
}

func (r *Recorder) currentText() string {
	current := ""
	if r.hotKeys.Shortcut != nil {
		current = r.hotKeys.Shortcut.Display()
	}
	return L.RecorderCurrent(current)
}

// relayout は文言の長さに合わせて窓の高さを決め、ボタンを置く
func (r *Recorder) relayout() {
	if r.hwnd == 0 {
		return
	}
	pad := r.px(20)
	width := r.px(420)
	inner := width - 2*pad
	dc := w32.GetDC(r.hwnd)
	old := w32.SelectObject(dc, w32.HGDIOBJ(r.font))
	measure := func(text string) int32 {
		rc := w32.RECT{Right: inner}
		w32.DrawText(dc, text, &rc, 0x10|0x400|0x800) // DT_WORDBREAK|DT_CALCRECT|DT_NOPREFIX
		return rc.Height()
	}
	lineH := measure("Ag")
	msgH := measure(r.message)
	// ボタンの幅は文言に合わせ、最低でも標準の大きさにする
	widths := map[int]int32{}
	for _, id := range []int{btnDisable, btnCancel, btnSave} {
		rc := w32.RECT{Right: 1000}
		w32.DrawText(dc, buttonText(id), &rc, 0x400|0x800|0x20) // DT_CALCRECT|DT_NOPREFIX|DT_SINGLELINE
		widths[id] = max(rc.Width()+r.px(24), r.px(88))
	}
	w32.SelectObject(dc, old)
	w32.ReleaseDC(r.hwnd, dc)

	y := pad
	l := recorderLayout{width: width}
	l.current = w32.RECT{Left: pad, Top: y, Right: pad + inner, Bottom: y + lineH}
	y += lineH + r.px(10)
	l.box = w32.RECT{Left: pad, Top: y, Right: pad + inner, Bottom: y + r.px(52)}
	y += r.px(52) + r.px(12)
	l.message = w32.RECT{Left: pad, Top: y, Right: pad + inner, Bottom: y + msgH}
	y += msgH + r.px(18)
	l.buttonsTop = y
	bh := r.px(30)
	y += bh + pad
	l.height = y
	r.layout = l

	x := width - pad
	for _, id := range []int{btnSave, btnCancel, btnDisable} {
		x -= widths[id]
		w32.MoveWindow(r.buttons[id], x, l.buttonsTop, widths[id], bh)
		x -= r.px(8)
	}
	style := uint32(w32.WS_POPUP | w32.WS_CAPTION | w32.WS_SYSMENU | w32.WS_CLIPCHILDREN)
	frame := w32.AdjustWindowRect(w32.RECT{Right: width, Bottom: l.height}, style, w32.WS_EX_TOPMOST, uint32(math.Round(float64(r.scale)*96)))
	w32.SetWindowPos(r.hwnd, 0, 0, 0, frame.Width(), frame.Height(), w32.SWP_NOMOVE|w32.SWP_NOZORDER|w32.SWP_NOACTIVATE)
}

func buttonText(id int) string {
	switch id {
	case btnDisable:
		return L.RecorderDisable
	case btnCancel:
		return L.RecorderCancel
	}
	return L.RecorderSave
}

func (r *Recorder) paint(hwnd w32.HWND) {
	var ps w32.PAINTSTRUCT
	dc := w32.BeginPaint(hwnd, &ps)
	defer w32.EndPaint(hwnd, &ps)
	client := w32.GetClientRect(hwnd)
	c := w32.NewCanvas(client.Width(), client.Height())
	defer c.Free()
	bg := r.background()
	fg := w32.Color(0, 0, 0, 1)
	secondary := w32.Color(0, 0, 0, 0.62)
	if r.dark {
		fg = w32.Color(255, 255, 255, 1)
		secondary = w32.Color(255, 255, 255, 0.62)
	}
	c.FillColor(w32.NewPath().Polygon(w32.PointF{X: 0, Y: 0}, w32.PointF{X: float32(client.Right), Y: 0},
		w32.PointF{X: float32(client.Right), Y: float32(client.Bottom)}, w32.PointF{X: 0, Y: float32(client.Bottom)}),
		w32.Color(float64(bg[0]), float64(bg[1]), float64(bg[2]), 1))

	// 押しているキーをその場で出す枠。段階に合わせて縁の色を変える
	b := r.layout.box
	radius := 8 * r.scale
	line := float32(math.Max(1.5*float64(r.scale), 1))
	bx, by, bw, bh := float32(b.Left), float32(b.Top), float32(b.Width()), float32(b.Height())
	c.FillColor(w32.NewPath().RoundRect(bx, by, bw, bh, radius), fg.WithAlpha(0.06))
	c.FillColor(w32.NewPath().RoundRect(bx, by, bw, bh, radius).RoundRect(bx+line, by+line, bw-2*line, bh-2*line, radius-line), r.frameColor(fg))
	c.Finish()

	mem := w32.CreateCompatibleDC(dc)
	defer w32.DeleteDC(mem)
	oldBmp := w32.SelectObject(mem, w32.HGDIOBJ(c.Bitmap))
	defer w32.SelectObject(mem, oldBmp)
	w32.SetBkMode(mem, w32.TRANSPARENT)
	oldFont := w32.SelectObject(mem, w32.HGDIOBJ(r.font))

	w32.SetTextColor(mem, argbToColorRef(secondary, bg))
	rc := r.layout.current
	w32.DrawText(mem, r.currentText(), &rc, 0x800|0x20) // DT_NOPREFIX|DT_SINGLELINE

	w32.SelectObject(mem, w32.HGDIOBJ(r.bigFont))
	w32.SetTextColor(mem, argbToColorRef(fg, bg))
	rc = r.layout.box
	live := r.live
	if live == "" {
		live = " "
	}
	w32.DrawText(mem, live, &rc, 0x800|0x20|0x1|0x4) // DT_NOPREFIX|DT_SINGLELINE|DT_CENTER|DT_VCENTER

	w32.SelectObject(mem, w32.HGDIOBJ(r.font))
	w32.SetTextColor(mem, argbToColorRef(r.messageColor(fg), bg))
	rc = r.layout.message
	w32.DrawText(mem, r.message, &rc, 0x10|0x800) // DT_WORDBREAK|DT_NOPREFIX
	w32.SelectObject(mem, oldFont)

	w32.BitBlt(dc, 0, 0, client.Width(), client.Height(), mem)
}

func (r *Recorder) frameColor(fg w32.ARGB) w32.ARGB {
	switch r.recording.Phase.Kind {
	case core.Verifying:
		return fg.WithAlpha(0.45)
	case core.Rejected:
		c, _ := levelBackground(core.LevelCritical)
		return c
	case core.Verified, core.Saved:
		return w32.Color(80, 190, 120, 1)
	}
	return fg.WithAlpha(0.18)
}

func (r *Recorder) messageColor(fg w32.ARGB) w32.ARGB {
	if r.recording.Phase.Kind != core.Rejected {
		return fg
	}
	// 赤の札と同じ色は暗い面では文字として読みにくいので、明るめの赤にする
	if r.dark {
		return w32.Color(255, 140, 128, 1)
	}
	return w32.Color(196, 43, 28, 1)
}

// argbToColorRef は不透明度付きの色を地の色に重ねた、GDIの文字色にする
func argbToColorRef(c w32.ARGB, bg [3]byte) uint32 {
	a := float64(c>>24) / 255
	mix := func(v uint32, b byte) uint32 {
		return uint32(math.Round(float64(v)*a + float64(b)*(1-a)))
	}
	return mix(uint32(c>>16)&0xFF, bg[0]) | mix(uint32(c>>8)&0xFF, bg[1])<<8 | mix(uint32(c)&0xFF, bg[2])<<16
}

func (r *Recorder) handle(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) (uintptr, bool) {
	switch msg {
	case w32.WM_PAINT:
		r.paint(hwnd)
		return 0, true
	case w32.WM_ERASEBKGND:
		return 1, true
	case w32.WM_CTLCOLORBTN:
		return uintptr(r.bgBrush), true
	case w32.WM_COMMAND:
		if w32.HiWord(wParam) == w32.BN_CLICKED {
			switch int(w32.LoWord(wParam)) {
			case btnDisable:
				r.turnOff()
			case btnCancel:
				r.cancel()
			case btnSave:
				r.save()
			}
			return 0, true
		}
	case w32.WM_CLOSE:
		// 閉じるボタンはキャンセルと同じ（保存した後なら閉じるだけ）
		if r.recording.IsSaved() {
			r.close()
		} else {
			r.cancel()
		}
		return 0, true
	case w32.WM_DPICHANGED:
		r.scale = float32(w32.HiWord(wParam)) / 96
		r.makeFonts()
		for _, b := range r.buttons[1:] {
			w32.SendMessage(b, w32.WM_SETFONT, uintptr(r.font), 1)
		}
		r.relayout()
		w32.InvalidateRect(hwnd)
		return 0, true
	}
	return 0, false
}
