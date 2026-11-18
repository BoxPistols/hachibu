//go:build windows

package app

import (
	"math"
	"strings"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

const stripClass = "HachibuWindowsStrip"

// 乗せてから広げるまでと、離してから畳むまでの待ち。通りがかりで開閉しないため
const (
	expandDelay   = 150 * time.Millisecond
	collapseDelay = 600 * time.Millisecond
	// 帯の下端と作業領域（タスクバーの上）の間
	bottomMargin = 12.0
)

// Strip は帯の窓と表示の状態を持つ。普段のモード（利用者が選ぶ）に、マウスを乗せたときとショートカットで呼んだときの一時的な展開を重ねる。
//
// 窓は押してもアクティブにならない（WS_EX_NOACTIVATE）。帯を触っても、使っているアプリからキー入力が離れない。
type Strip struct {
	hwnd    w32.HWND
	tooltip w32.HWND
	prefs   *Prefs

	allSlots   []core.Slot
	hovering   bool
	Summoned   bool
	RestLayout core.StripLayout
	Opacity    float64
	Thresholds core.Thresholds

	// メニューの項目にカーソルを乗せている間だけ使う仮の値。選ばずに閉じたら消す
	previewLayout     *core.StripLayout
	previewOpacity    *float64
	previewThresholds *core.Thresholds
	// メニューで選んだ直後は、カーソルが帯の上に残っていても広げない（選んだモードをすぐ見せる）。カーソルが一度帯の外に出たら解く
	hoverSuppressed bool
	menuOpen        bool

	// OnChange は表示が変わったとき（トレイのアイコンを合わせるため）
	OnChange func()
	// UsageText は「使用率だけ」モードで状態の枠に出す文字
	UsageText func() string
	// ContextMenu は帯の右クリックメニューを開く（閉じるまで戻らない）
	ContextMenu func(at w32.POINT)

	hoverCheck    Delay
	suppressCheck Delay
	tracking      bool
	dragging      bool
	dragStart     w32.POINT
	dragFrame     w32.RECT

	visible     bool
	frame       w32.RECT
	handleRight int32
	tipText     string
}

func newStrip(prefs *Prefs) *Strip {
	s := &Strip{
		prefs:      prefs,
		RestLayout: prefs.RestLayout(),
		Opacity:    prefs.Opacity(),
		Thresholds: prefs.Thresholds(),
		OnChange:   func() {},
		UsageText:  func() string { return "" },
	}
	registerClass(stripClass, w32.LoadCursor(w32.IDC_ARROW))
	s.hwnd = newWindow(w32.WS_EX_LAYERED|w32.WS_EX_TOOLWINDOW|w32.WS_EX_TOPMOST|w32.WS_EX_NOACTIVATE,
		stripClass, core.AppName, w32.WS_POPUP, 0, 0, 1, 1, 0, s.handle)
	s.tooltip = w32.AddTooltip(s.hwnd, 520)
	return s
}

// StatusSlot は状態の枠（メニューの先頭とツールチップに使う）。
func (s *Strip) StatusSlot() (core.Slot, bool) {
	for _, slot := range s.allSlots {
		if slot.ID == core.StatusID {
			return slot, true
		}
	}
	return core.Slot{}, false
}

func (s *Strip) Update(slots []core.Slot) {
	s.allSlots = slots
	s.render()
}

// MARK: - 設定の変更（メニューから）

func (s *Strip) SetRestLayout(l core.StripLayout) {
	s.RestLayout = l
	s.prefs.Set("restLayout", string(l))
	s.previewLayout = nil
	s.Summoned = false
	s.suppressHover()
	s.render()
}

func (s *Strip) SetThresholds(t core.Thresholds) {
	s.Thresholds = t
	s.prefs.SetThresholds(t)
	s.previewThresholds = nil
	s.render()
}

func (s *Strip) SetOpacity(v float64) {
	s.Opacity = v
	s.prefs.Set("opacity", v)
	s.previewOpacity = nil
	s.suppressHover()
	s.render()
}

// MARK: - 仮の表示（メニューの項目に乗せている間）

func (s *Strip) PreviewLayout(l core.StripLayout) {
	s.previewLayout = &l
	s.render()
}

func (s *Strip) PreviewOpacity(v float64) {
	s.previewOpacity = &v
	s.render()
}

func (s *Strip) PreviewThresholds(t core.Thresholds) {
	s.previewThresholds = &t
	s.render()
}

func (s *Strip) ClearPreview() {
	if s.previewLayout == nil && s.previewOpacity == nil && s.previewThresholds == nil {
		return
	}
	s.previewLayout, s.previewOpacity, s.previewThresholds = nil, nil, nil
	s.render()
}

// MenuOpened とMenuClosed はメニューの開閉。開いている間はマウスの出入りで畳まない
func (s *Strip) MenuOpened() { s.menuOpen = true }

func (s *Strip) MenuClosed() {
	s.menuOpen = false
	s.ClearPreview()
	s.scheduleHoverCheck()
}

// ToggleSummon はショートカットとメニューから。呼び出し中ならもう一度で元のモードへ戻す
func (s *Strip) ToggleSummon() {
	s.Summoned = !s.Summoned
	s.render()
}

// CycleLayout はショートカットとメニューから。標準→使用率だけ→端に収納→標準の順に切り替える
func (s *Strip) CycleLayout() {
	s.SetRestLayout(core.NextLayout(s.RestLayout))
}

// ShowSummoned はアプリをもう一度開いたとき。隠すモードでも帯を出す（戻すのはショートカットかメニューから）
func (s *Strip) ShowSummoned() {
	s.Summoned = true
	s.render()
}

func (s *Strip) ResetPosition() {
	s.prefs.SetStripAnchor(nil)
	s.render()
}

// MARK: - 描画

// hoverExpands はマウスを乗せたことによる展開。メニューで選んだ直後と、仮の表示の間は広げない
func (s *Strip) hoverExpands() bool {
	return s.hovering && !s.hoverSuppressed && s.previewLayout == nil && s.previewOpacity == nil
}

func (s *Strip) currentView() stripView {
	rest := s.RestLayout
	if s.previewLayout != nil {
		rest = *s.previewLayout
	}
	layout := core.EffectiveLayout(rest, s.hoverExpands() || s.Summoned)
	t := s.Thresholds
	if s.previewThresholds != nil {
		t = *s.previewThresholds
	}
	return stripView{layout: layout, slots: core.VisibleSlots(s.allSlots, layout, s.UsageText()), thresholds: t}
}

func (s *Strip) render() {
	// ドラッグしている間は置き直さない（保存済みの位置へ跳ね戻るため）。離したときに描き直す
	if s.dragging {
		return
	}
	view := s.currentView()
	s.OnChange()
	if view.layout == core.LayoutHidden {
		s.hovering = false
		s.visible = false
		w32.ShowWindow(s.hwnd, w32.SW_HIDE)
		return
	}
	// 置く先の画面の拡大率で描く。先に今の大きさで置き場所を決め、その画面の拡大率で描き直す
	guess := s.frameFor(s.frame.Width(), s.frame.Height(), 1)
	scale := float32(w32.DpiForMonitor(w32.MonitorFromRect(guess, w32.MONITOR_DEFAULTTONEAREST))) / 96
	c, g := paintStrip(view, scale)
	defer c.Free()
	s.handleRight = int32(g.handleRight)
	frame := s.clamped(s.frameFor(g.w, g.h, scale))
	s.frame = frame
	// 触っている間と呼び出し中は読めるように不透明へ戻す。仮の表示の間はその値を見せる
	opacity := s.Opacity
	if s.previewOpacity != nil {
		opacity = *s.previewOpacity
	}
	if s.hoverExpands() || s.Summoned {
		opacity = 1
	}
	c.Finish()
	w32.UpdateLayered(s.hwnd, w32.POINT{X: frame.Left, Y: frame.Top}, w32.SIZE{CX: g.w, CY: g.h}, c.Bitmap, byte(math.Round(opacity*255)))
	if !s.visible {
		w32.ShowWindow(s.hwnd, w32.SW_SHOWNOACTIVATE)
		s.visible = true
	}
	w32.SetWindowPos(s.hwnd, w32.HWND_TOPMOST, 0, 0, 0, 0, w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOACTIVATE)
	s.updateTooltip()
}

// updateTooltip は見えている文字と同じ説明は出さない。出すのはコンテキスト長の綴りとリセット時刻など、帯に載っていない情報だけ
func (s *Strip) updateTooltip() {
	slot, ok := s.StatusSlot()
	if !ok {
		return
	}
	var lines []string
	if spelled, ok := core.SpelledOut(slot.Text); ok {
		lines = append(lines, spelled)
	}
	if slot.Help != "" {
		lines = append(lines, slot.Help)
	}
	text := strings.Join(lines, "\n")
	if text != s.tipText {
		s.tipText = text
		w32.SetTooltipText(s.tooltip, s.hwnd, text)
	}
}

// MARK: - 置き場所

// 置き場所は「左端・中央・右端のどれを基準にするか」と、その基準点の座標で覚える。
// 左上の座標で覚えると、標準で保存した位置に使用率だけで出したとき、中央寄せの帯が左へずれる

func (s *Strip) frameFor(w, h int32, scale float32) w32.RECT {
	if w <= 0 {
		w, h = 1, 1
	}
	if a, ok := s.prefs.StripAnchor(); ok {
		x := a.X
		switch a.Edge {
		case "center":
			x = a.X - w/2
		case "right":
			x = a.X - w
		}
		frame := w32.RECT{Left: x, Top: a.Y, Right: x + w, Bottom: a.Y + h}
		// 外部ディスプレイを外した後などで、保存位置がどの画面にも無ければ既定に戻す
		for _, area := range w32.WorkAreas() {
			if area.Intersects(frame) {
				return frame
			}
		}
	}
	areas := w32.WorkAreas()
	area := w32.RECT{Right: 1920, Bottom: 1080}
	if len(areas) > 0 {
		area = areas[0]
	}
	x := area.Left + (area.Width()-w)/2
	y := area.Bottom - int32(math.Round(bottomMargin*float64(scale))) - h
	return w32.RECT{Left: x, Top: y, Right: x + w, Bottom: y + h}
}

func (s *Strip) clamped(f w32.RECT) w32.RECT {
	areas := w32.WorkAreas()
	if len(areas) == 0 {
		return f
	}
	area := areas[0]
	for _, a := range areas {
		if a.Intersects(f) {
			area = a
			break
		}
	}
	w, h := f.Width(), f.Height()
	x := min(max(f.Left, area.Left), area.Right-w)
	y := min(max(f.Top, area.Top), area.Bottom-h)
	return w32.RECT{Left: x, Top: y, Right: x + w, Bottom: y + h}
}

// userMoved は利用者が動かした位置を覚える。画面の左3分の1に置いたら左端、右3分の1なら右端を基準にする
func (s *Strip) userMoved() {
	f := s.clamped(w32.GetWindowRect(s.hwnd))
	area := f
	for _, a := range w32.WorkAreas() {
		if a.Intersects(f) {
			area = a
			break
		}
	}
	mid := (f.Left + f.Right) / 2
	ratio := float64(mid-area.Left) / math.Max(float64(area.Width()), 1)
	a := stripAnchor{Edge: "center", X: mid, Y: f.Top}
	if ratio < 1.0/3 {
		a = stripAnchor{Edge: "left", X: f.Left, Y: f.Top}
	} else if ratio > 2.0/3 {
		a = stripAnchor{Edge: "right", X: f.Right, Y: f.Top}
	}
	s.prefs.SetStripAnchor(&a)
}

// MARK: - マウス

func (s *Strip) cursorInside() bool {
	return s.visible && w32.GetWindowRect(s.hwnd).Contains(w32.GetCursorPos())
}

func (s *Strip) suppressHover() {
	s.hoverSuppressed = true
	// 小さくなって帯がカーソルの下から外れた場合、外へ出た知らせが来ないことがあるので、置き直した後に確かめる
	s.suppressCheck.Schedule(300*time.Millisecond, func() {
		if !s.cursorInside() {
			s.hoverSuppressed = false
			s.hovering = false
		}
	})
}

// scheduleHoverCheck は出入りの知らせそのものではなく、待ったあとの実際のカーソル位置で決める。
// 広げると枠が動くため、出入りの知らせが実際と食い違うことがある
func (s *Strip) scheduleHoverCheck() {
	delay := collapseDelay
	if s.cursorInside() {
		delay = expandDelay
	}
	s.hoverCheck.Schedule(delay, func() {
		if s.menuOpen || s.dragging {
			return
		}
		now := s.cursorInside()
		if !now {
			s.hoverSuppressed = false
		}
		if now == s.hovering {
			return
		}
		s.hovering = now
		s.render()
	})
}

// mkLButton はマウスのメッセージのwParamで、左ボタンが押されていることを示す値
const mkLButton = 0x0001

// endDrag はドラッグを終え、置いた位置を覚える。releaseはマウスの捕捉を自分で外すか
// （捕捉がすでにほかへ移ったとき（WM_CAPTURECHANGED）に外すと、同じスレッドのメニューなどの捕捉まで外してしまう）
func (s *Strip) endDrag(release bool) {
	if !s.dragging {
		return
	}
	s.dragging = false
	if release {
		w32.ReleaseCapture()
	}
	s.userMoved()
	s.render()
}

func (s *Strip) onHandle(client w32.POINT) bool {
	return s.currentView().layout == core.LayoutTab || client.X < s.handleRight
}

func (s *Strip) handle(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) (uintptr, bool) {
	switch msg {
	case w32.WM_MOUSEACTIVATE:
		return w32.MA_NOACTIVATE, true
	case w32.WM_SETCURSOR:
		if w32.LoWord(lParam) == w32.HTCLIENT {
			p := w32.GetCursorPos()
			client := w32.POINT{X: p.X - s.frame.Left, Y: p.Y - s.frame.Top}
			if s.onHandle(client) {
				w32.SetCursor(w32.LoadCursor(w32.IDC_SIZEALL))
			} else {
				w32.SetCursor(w32.LoadCursor(w32.IDC_ARROW))
			}
			return 1, true
		}
	case w32.WM_MOUSEMOVE:
		// 離した知らせ（WM_LBUTTONUP）を受け取れなかった場合に備え、移動のたびに左ボタンの状態も見る
		if s.dragging && wParam&mkLButton == 0 {
			s.endDrag(true)
			return 0, true
		}
		if s.dragging {
			p := w32.GetCursorPos()
			x := s.dragFrame.Left + p.X - s.dragStart.X
			y := s.dragFrame.Top + p.Y - s.dragStart.Y
			w32.SetWindowPos(hwnd, 0, x, y, 0, 0, w32.SWP_NOSIZE|w32.SWP_NOZORDER|w32.SWP_NOACTIVATE)
			return 0, true
		}
		if !s.tracking {
			s.tracking = true
			w32.TrackMouseLeave(hwnd)
			s.scheduleHoverCheck()
		}
		return 0, true
	case w32.WM_MOUSELEAVE:
		s.tracking = false
		s.scheduleHoverCheck()
		return 0, true
	case w32.WM_LBUTTONDOWN:
		// 移動はつまみだけで行う。状態の枠を掴めると、読もうとして触っただけで帯ごと動く
		if s.onHandle(w32.PointFromLParam(lParam)) {
			s.dragging = true
			s.dragStart = w32.GetCursorPos()
			s.dragFrame = w32.GetWindowRect(hwnd)
			w32.SetCapture(hwnd)
		}
		return 0, true
	case w32.WM_LBUTTONUP:
		s.endDrag(true)
		return 0, true
	case w32.WM_CAPTURECHANGED:
		s.endDrag(false)
		return 0, true
	case w32.WM_RBUTTONUP:
		if s.ContextMenu != nil {
			s.ContextMenu(w32.GetCursorPos())
		}
		return 0, true
	case w32.WM_DPICHANGED:
		s.render()
		return 0, true
	}
	return 0, false
}
