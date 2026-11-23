//go:build windows

package app

import (
	"math"
	"strconv"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

// 寸法はmacOS向けのMetricsと同じ値（96dpiでの画素）。描くときに画面の拡大率を掛ける
const (
	chipHeight  = 22.0
	inset       = 3.0
	cornerRad   = 8.0
	tagRadius   = 3.0
	fontSize    = 13.0
	tabWidth    = 20.0
	handleWidth = 12.0
	// 札の高さ。文字の行の高さに上下1画素ずつの余白を足した大きさ
	tagHeight     = 18.0
	tagPadding    = 3.0
	contextDotDia = 4.0
)

func rgba(c core.RGBA) w32.ARGB { return w32.Color(c.R, c.G, c.B, c.A/255) }

// 使用率の段階ごとの札の色。黄は濃い文字（10.8:1）、赤は白い文字（4.8:1）
func levelBackground(l core.UsageLevel) (w32.ARGB, bool) {
	switch l {
	case core.LevelWarning:
		return w32.Color(242, 201, 76, 1), true
	case core.LevelCritical:
		return w32.Color(217, 48, 37, 1), true
	}
	return 0, false
}

func levelForeground(l core.UsageLevel) w32.ARGB {
	if l == core.LevelWarning {
		return w32.Color(28, 28, 30, 1)
	}
	return w32.Color(255, 255, 255, 1)
}

// contextDotColor はコンテキスト長の点の色。使用率の黄と赤のどちらとも見分けられる橙
var contextDotColor = w32.Color(224, 124, 84, 1)

var accentYellow = w32.Color(242, 201, 76, 1)

// stripView は帯に描くもの。
type stripView struct {
	layout     core.StripLayout
	slots      []core.Slot
	thresholds core.Thresholds
}

// piece は状態の枠の中に横へ並べる部品。
type piece struct {
	kind  int // pieceText, pieceTag, pieceDot
	text  string
	level core.UsageLevel
	// shift は前の部品へ詰める量（96dpiでの画素）。点の後ろの文字を点に寄せる
	shift float32
	width float32
}

const (
	pieceText = iota
	pieceTag
	pieceDot
)

// fonts は拡大率ごとの字体。同じ拡大率なら作り直さない
type fontCache struct {
	scale float32
	strip *w32.Font
}

var fonts fontCache

func stripFont(scale float32) *w32.Font {
	if fonts.strip == nil || fonts.scale != scale {
		if fonts.strip != nil {
			fonts.strip.Free()
		}
		// macOS向けのSF 13ptの中太に近い太さ。数字はSegoe UIでも等幅
		fonts.strip = w32.NewFont("Segoe UI Semibold", fontSize*scale, 400)
		fonts.scale = scale
	}
	return fonts.strip
}

var measurer *w32.Canvas

// measure は文字の幅を測るための小さな画用紙（GDI+は描き先が無いと幅を測れない）
func measure() *w32.Canvas {
	if measurer == nil {
		measurer = w32.NewCanvas(1, 1)
	}
	return measurer
}

// statusPieces は状態の枠の文字を部品に分ける。使用率の語に段階を付けるのは状態の枠だけ
func statusPieces(slot core.Slot, t core.Thresholds, font *w32.Font, scale float32) []piece {
	segments := []core.TextSegment{{Text: slot.Text, Level: core.LevelNormal}}
	if slot.ID == core.StatusID {
		segments = core.Segments(slot.Text, t)
	}
	var out []piece
	m := measure()
	for _, s := range segments {
		if s.Level != core.LevelNormal {
			out = append(out, piece{kind: pieceTag, text: s.Text, level: s.Level, width: m.TextWidth(s.Text, font) + 2*tagPadding*scale})
		} else if split, ok := core.SplitContext(s.Text); ok {
			// コンテキスト長は文字で書かず、モデル名の肩に小さな点で示す（文字はメニューとツールチップに出す）
			out = append(out,
				piece{kind: pieceText, text: split.Model, width: m.TextWidth(split.Model, font)},
				piece{kind: pieceDot, width: (contextDotDia + 3) * scale},
				piece{kind: pieceText, text: split.Rest, shift: 1.5, width: m.TextWidth(split.Rest, font)})
		} else {
			out = append(out, piece{kind: pieceText, text: s.Text, width: m.TextWidth(s.Text, font)})
		}
	}
	return out
}

// worstLevel はつまみの輪の色に使う、状態の枠の中でいちばん高い段階
func worstLevel(v stripView) core.UsageLevel {
	worst := core.LevelNormal
	if len(v.slots) == 0 {
		return worst
	}
	for _, s := range core.Segments(v.slots[0].Text, v.thresholds) {
		worst = max(worst, s.Level)
	}
	return worst
}

// stripGeometry は描く前に決める大きさと、つまみの範囲（ドラッグを受ける幅）。
type stripGeometry struct {
	w, h        int32
	handleRight float32
	pieces      [][]piece
}

func layoutStrip(v stripView, scale float32) stripGeometry {
	h := int32(math.Ceil(float64((chipHeight + 2*inset) * scale)))
	if v.layout == core.LayoutTab {
		w := int32(math.Ceil(float64(tabWidth * scale)))
		return stripGeometry{w: w, h: h, handleRight: float32(w)}
	}
	font := stripFont(scale)
	x := (inset + handleWidth + 2) * scale
	g := stripGeometry{h: h, handleRight: x}
	for _, slot := range v.slots {
		pieces := statusPieces(slot, v.thresholds, font, scale)
		x += 4 * scale
		for _, p := range pieces {
			x += p.width - p.shift*scale
		}
		g.pieces = append(g.pieces, pieces)
	}
	x += (inset + 4) * scale
	// 端数を切り上げて1画素の余裕を足す。ちょうどの幅だと、描くときの丸めで文字が詰められることがある
	g.w = int32(math.Ceil(float64(x))) + 1
	return g
}

// paintStrip は帯を描いた画用紙を返す。片付け（Finish、Free）は呼んだ側で行う
func paintStrip(v stripView, scale float32) (*w32.Canvas, stripGeometry) {
	g := layoutStrip(v, scale)
	c := w32.NewCanvas(g.w, g.h)
	W, H := float32(g.w), float32(g.h)

	// ガラスの面の代わり。Windowsでは背景をぼかせないので、macOS向けより不透明にして文字を読みやすくする
	face := w32.NewPath().RoundRect(0, 0, W, H, cornerRad*scale)
	bg := w32.VerticalGradient(w32.RectF{X: 0, Y: 0, W: W, H: H}, w32.Color(43, 43, 45, 0.94), w32.Color(18, 18, 20, 0.96))
	c.Fill(face, bg)
	bg.Free()
	face.Free()
	// ガラスの縁。上の辺が光を受け、下へ向かって消える
	line := float32(math.Max(1, math.Round(float64(scale))))
	rim := w32.NewPath().RoundRect(0, 0, W, H, cornerRad*scale).RoundRect(line, line, W-2*line, H-2*line, cornerRad*scale-line)
	rimBrush := w32.VerticalGradient(w32.RectF{X: 0, Y: 0, W: W, H: H},
		w32.Color(255, 255, 255, 0.30), w32.Color(255, 255, 255, 0.08), w32.Color(255, 255, 255, 0.14))
	c.Fill(rim, rimBrush)
	rimBrush.Free()
	rim.Free()

	if v.layout == core.LayoutTab {
		paintTab(c, v, W, H, scale)
		return c, g
	}

	paintHandleDots(c, inset*scale, 0, handleWidth*scale, H, scale)
	centerY := H / 2
	font := stripFont(scale)
	x := (inset + handleWidth + 2) * scale
	for i, slot := range v.slots {
		alpha := 1.0
		if slot.Dimmed {
			alpha = 0.5
		}
		x += 4 * scale
		for _, p := range g.pieces[i] {
			x -= p.shift * scale
			switch p.kind {
			case pieceText:
				c.DrawText(p.text, font, rgba(slot.Foreground).WithAlpha(alpha), x, centerY)
			case pieceTag:
				color, _ := levelBackground(p.level)
				th := tagHeight * scale
				c.FillColor(w32.NewPath().RoundRect(x, centerY-th/2, p.width, th, tagRadius*scale), color.WithAlpha(alpha))
				c.DrawText(p.text, font, levelForeground(p.level).WithAlpha(alpha), x+tagPadding*scale, centerY)
			case pieceDot:
				d := contextDotDia * scale
				capTop := centerY - font.Size*0.7/2
				c.FillColor(w32.NewPath().Ellipse(x+1.5*scale, capTop-0.5*scale, d, d), contextDotColor.WithAlpha(alpha))
			}
			x += p.width
		}
	}
	return c, g
}

// paintHandleDots は帯を動かすつまみの点（2列3行）
func paintHandleDots(c *w32.Canvas, x, y, w, h, scale float32) {
	d := 2.5 * scale
	gap := 3.5 * scale
	cols, rows := float32(2), float32(3)
	gw := cols*d + (cols-1)*gap
	gh := rows*d + (rows-1)*gap
	x0 := x + (w-gw)/2
	y0 := y + (h-gh)/2
	p := w32.NewPath()
	for col := float32(0); col < cols; col++ {
		for row := float32(0); row < rows; row++ {
			p.Ellipse(x0+col*(d+gap), y0+row*(d+gap), d, d)
		}
	}
	c.FillColor(p, w32.Color(255, 255, 255, 0.32))
}

// paintTab は端に収納したときの姿。状態は点の色だけで示し、使用率が黄や赤の段階なら点の周りに輪を付ける
func paintTab(c *w32.Canvas, v stripView, W, H, scale float32) {
	cx, cy := W/2, H/2
	dot := rgba(core.IdleForeground).WithAlpha(0.7)
	if len(v.slots) > 0 && v.slots[0].Background.A > 0 {
		dot = rgba(v.slots[0].Background)
	}
	d := 8 * scale
	c.FillColor(w32.NewPath().Ellipse(cx-d/2, cy-d/2, d, d), dot)
	if ring, ok := levelBackground(worstLevel(v)); ok {
		outer := 14 * scale
		inner := outer - 4*scale
		c.FillColor(w32.NewPath().Ellipse(cx-outer/2, cy-outer/2, outer, outer).Ellipse(cx-inner/2, cy-inner/2, inner, inner), ring)
	}
}

// glyph はアプリのアイコンと同じ「斜めの切れ目で2つに分かれた帯」。
// gaugeが0以上なら切れ目を使用率の位置へ動かし、左（使った分）を段階の色にする
func paintGlyph(c *w32.Canvas, x0, x1, centerY, thickness float32, label w32.ARGB, gauge int, level core.UsageLevel) {
	t := thickness
	gap := t / 3
	slant := t * 0.6
	top, bottom := centerY-t/2, centerY+t/2
	fraction := float32(0.62)
	if gauge >= 0 {
		fraction = float32(min(max(gauge, 0), 100)) / 100
	}
	// 両端でも左右の部分が消えないよう、端から少し残す
	reach := slant/2 + gap/2 + t/4
	cut := min(max(x0+(x1-x0)*fraction, x0+reach), x1-reach)
	left := w32.NewPath().Polygon(
		w32.PointF{X: x0, Y: top}, w32.PointF{X: cut - gap/2 + slant/2, Y: top},
		w32.PointF{X: cut - gap/2 - slant/2, Y: bottom}, w32.PointF{X: x0, Y: bottom})
	right := w32.NewPath().Polygon(
		w32.PointF{X: cut + gap/2 + slant/2, Y: top}, w32.PointF{X: x1, Y: top},
		w32.PointF{X: x1, Y: bottom}, w32.PointF{X: cut + gap/2 - slant/2, Y: bottom})
	if gauge >= 0 {
		used := label.WithAlpha(0.85)
		if color, ok := levelBackground(level); ok {
			used = color
		}
		c.FillColor(left, used)
		c.FillColor(right, label.WithAlpha(0.32))
	} else {
		c.FillColor(left, label.WithAlpha(0.6))
		c.FillColor(right, accentYellow)
	}
}

// paintTrayIcon はトレイのアイコン。sizeは画素、darkはタスクバーが暗い色か
func paintTrayIcon(style core.TrayStyle, limits []core.UsageLimit, t core.Thresholds, size int32, dark bool) *w32.Canvas {
	c := w32.NewCanvas(size, size)
	s := float32(size)
	label := w32.Color(0, 0, 0, 1)
	if dark {
		label = w32.Color(255, 255, 255, 1)
	}
	reading, hasReading := core.GaugeFrom(limits)
	switch {
	case style == core.TrayGauge && hasReading:
		paintGlyph(c, s*0.03, s*0.97, s/2, s*0.42, label, reading.Percent, t.Level(reading.Percent))
	case style == core.TrayPercent && hasReading:
		paintTrayNumber(c, reading.Label, t.Level(reading.Percent), s, label)
	default:
		paintGlyph(c, s*0.03, s*0.97, s/2, s*0.42, label, -1, core.LevelNormal)
	}
	return c
}

// paintTrayNumber は週枠の使用率を数字1つで描く。黄や赤の段階では、帯と同じ色の札にする
func paintTrayNumber(c *w32.Canvas, text string, level core.UsageLevel, s float32, label w32.ARGB) {
	px := s * 0.78
	if len(text) >= 3 {
		px = s * 0.56
	}
	font := w32.NewFont("Segoe UI Semibold", px, 400)
	defer font.Free()
	color := label
	if bg, ok := levelBackground(level); ok {
		c.FillColor(w32.NewPath().RoundRect(0, 0, s, s, s*0.18), bg)
		color = levelForeground(level)
	}
	w := c.TextWidth(text, font)
	c.DrawText(text, font, color, (s-w)/2, s/2)
}

// dotBitmap はメニューの行の左に付ける色の点（帯の札と同じ色）。メニューは乗算済みアルファの32bppの絵をそのまま描く
func dotBitmap(color w32.ARGB, diameter float32, scale float32) w32.HBITMAP {
	size := int32(math.Round(float64(16 * scale)))
	c := w32.NewCanvas(size, size)
	d := diameter * scale
	s := float32(size)
	c.FillColor(w32.NewPath().Ellipse((s-d)/2, (s-d)/2, d, d), color)
	c.Finish()
	return c.Bitmap
}

func itoa(i int) string { return strconv.Itoa(i) }
