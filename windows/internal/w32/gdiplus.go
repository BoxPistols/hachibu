//go:build windows

package w32

import (
	"errors"
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GDI+で描く。帯、トレイのアイコン、メニューの点はすべてここを通る。
//
// GDI+の関数のうち、座標や太さを浮動小数点の値で受け取るもの（GdipFillEllipse、GdipCreatePen1など）は使わない。
// Goのsyscallは引数を整数のレジスタに置くので、arm64では浮動小数点の引数が正しく届かない。
// 形はすべて点の配列（ポインタで渡す）からパスに組み、塗りつぶしで描く。線も「外側の形から内側の形を抜いた面」として塗る。

var (
	gdiplus = windows.NewLazySystemDLL("gdiplus.dll")

	pGdiplusStartup                        = gdiplus.NewProc("GdiplusStartup")
	pGdipCreateBitmapFromScan0             = gdiplus.NewProc("GdipCreateBitmapFromScan0")
	pGdipGetImageGraphicsContext           = gdiplus.NewProc("GdipGetImageGraphicsContext")
	pGdipDeleteGraphics                    = gdiplus.NewProc("GdipDeleteGraphics")
	pGdipDisposeImage                      = gdiplus.NewProc("GdipDisposeImage")
	pGdipSetSmoothingMode                  = gdiplus.NewProc("GdipSetSmoothingMode")
	pGdipSetPixelOffsetMode                = gdiplus.NewProc("GdipSetPixelOffsetMode")
	pGdipSetTextRenderingHint              = gdiplus.NewProc("GdipSetTextRenderingHint")
	pGdipCreatePath                        = gdiplus.NewProc("GdipCreatePath")
	pGdipDeletePath                        = gdiplus.NewProc("GdipDeletePath")
	pGdipAddPathBeziers                    = gdiplus.NewProc("GdipAddPathBeziers")
	pGdipAddPathPolygon                    = gdiplus.NewProc("GdipAddPathPolygon")
	pGdipClosePathFigure                   = gdiplus.NewProc("GdipClosePathFigure")
	pGdipStartPathFigure                   = gdiplus.NewProc("GdipStartPathFigure")
	pGdipFillPath                          = gdiplus.NewProc("GdipFillPath")
	pGdipCreateSolidFill                   = gdiplus.NewProc("GdipCreateSolidFill")
	pGdipCreateLineBrushFromRect           = gdiplus.NewProc("GdipCreateLineBrushFromRect")
	pGdipSetLinePresetBlend                = gdiplus.NewProc("GdipSetLinePresetBlend")
	pGdipDeleteBrush                       = gdiplus.NewProc("GdipDeleteBrush")
	pGdipCreateFontFromLogfontW            = gdiplus.NewProc("GdipCreateFontFromLogfontW")
	pGdipDeleteFont                        = gdiplus.NewProc("GdipDeleteFont")
	pGdipGetFamily                         = gdiplus.NewProc("GdipGetFamily")
	pGdipDeleteFontFamily                  = gdiplus.NewProc("GdipDeleteFontFamily")
	pGdipGetFontStyle                      = gdiplus.NewProc("GdipGetFontStyle")
	pGdipGetCellAscent                     = gdiplus.NewProc("GdipGetCellAscent")
	pGdipGetEmHeight                       = gdiplus.NewProc("GdipGetEmHeight")
	pGdipGetFontSize                       = gdiplus.NewProc("GdipGetFontSize")
	pGdipStringFormatGetGenericTypographic = gdiplus.NewProc("GdipStringFormatGetGenericTypographic")
	pGdipCloneStringFormat                 = gdiplus.NewProc("GdipCloneStringFormat")
	pGdipSetStringFormatFlags              = gdiplus.NewProc("GdipSetStringFormatFlags")
	pGdipDrawString                        = gdiplus.NewProc("GdipDrawString")
	pGdipMeasureString                     = gdiplus.NewProc("GdipMeasureString")
	pGdipSaveImageToFile                   = gdiplus.NewProc("GdipSaveImageToFile")
	pGdipCreateHICONFromBitmap             = gdiplus.NewProc("GdipCreateHICONFromBitmap")
)

// ARGB はGDI+の色（0xAARRGGBB）。
type ARGB uint32

// Color は0〜255の色と0〜1の不透明度から作る。
func Color(r, g, b, a float64) ARGB {
	c := func(v float64) uint32 { return uint32(math.Round(math.Max(0, math.Min(255, v)))) }
	return ARGB(c(a*255)<<24 | c(r)<<16 | c(g)<<8 | c(b))
}

// WithAlpha は不透明度だけを掛ける。
func (c ARGB) WithAlpha(a float64) ARGB {
	alpha := math.Round(float64(c>>24) * math.Max(0, math.Min(1, a)))
	return ARGB(uint32(alpha)<<24 | uint32(c)&0xFFFFFF)
}

type PointF struct{ X, Y float32 }
type RectF struct{ X, Y, W, H float32 }

var gdiplusStarted bool

// StartGDIPlus はGDI+を使えるようにする。プロセスで1回だけ呼ぶ
func StartGDIPlus() error {
	if gdiplusStarted {
		return nil
	}
	type startupInput struct {
		Version                  uint32
		DebugEventCallback       uintptr
		SuppressBackgroundThread int32
		SuppressExternalCodecs   int32
	}
	in := startupInput{Version: 1}
	var token uintptr
	r, _, _ := pGdiplusStartup.Call(uintptr(unsafe.Pointer(&token)), uintptr(unsafe.Pointer(&in)), 0)
	if r != 0 {
		return errors.New("GdiplusStartup failed")
	}
	gdiplusStarted = true
	return nil
}

// Canvas は32bppのDIBに直接描く画用紙。UpdateLayeredWindowとメニューの絵にそのまま渡せる（乗算済みアルファ）。
type Canvas struct {
	W, H   int32
	Bitmap HBITMAP
	img    uintptr
	g      uintptr
}

const pixelFormat32bppPARGB = 0x000E200B

func NewCanvas(w, h int32) *Canvas {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	bmp, bits := CreateDIB(w, h)
	c := &Canvas{W: w, H: h, Bitmap: bmp}
	pGdipCreateBitmapFromScan0.Call(uintptr(w), uintptr(h), uintptr(w*4), pixelFormat32bppPARGB, uintptr(bits), uintptr(unsafe.Pointer(&c.img)))
	pGdipGetImageGraphicsContext.Call(c.img, uintptr(unsafe.Pointer(&c.g)))
	pGdipSetSmoothingMode.Call(c.g, 4)     // SmoothingModeAntiAlias8x8相当（AntiAlias）
	pGdipSetPixelOffsetMode.Call(c.g, 4)   // PixelOffsetModeHalf: 座標を画素の中心でなく角に合わせる
	pGdipSetTextRenderingHint.Call(c.g, 4) // TextRenderingHintAntiAlias: 透明な地にはClearTypeを使えない
	return c
}

// Finish は描き終えたらGDI+の側だけを片付け、DIB（Bitmap）は残す。
func (c *Canvas) Finish() {
	if c.g != 0 {
		pGdipDeleteGraphics.Call(c.g)
		c.g = 0
	}
	if c.img != 0 {
		pGdipDisposeImage.Call(c.img)
		c.img = 0
	}
}

// Free はDIBも含めてすべて片付ける。
func (c *Canvas) Free() {
	c.Finish()
	if c.Bitmap != 0 {
		DeleteObject(HGDIOBJ(c.Bitmap))
		c.Bitmap = 0
	}
}

// SavePNG は描いた絵をPNGに書く（開発用の書き出し）。
func (c *Canvas) SavePNG(path string) error {
	// PNGのエンコーダ {557CF406-1A04-11D3-9A73-0000F81EF32E}
	clsid := windows.GUID{Data1: 0x557CF406, Data2: 0x1A04, Data3: 0x11D3, Data4: [8]byte{0x9A, 0x73, 0x00, 0x00, 0xF8, 0x1E, 0xF3, 0x2E}}
	r, _, _ := pGdipSaveImageToFile.Call(c.img, uintptr(unsafe.Pointer(UTF16(path))), uintptr(unsafe.Pointer(&clsid)), 0)
	if r != 0 {
		return errors.New("GdipSaveImageToFile failed")
	}
	return nil
}

// HIcon は描いた絵からアイコンを作る（トレイ用）。使い終えたらDestroyIconで消す
func (c *Canvas) HIcon() HICON {
	var h HICON
	pGdipCreateHICONFromBitmap.Call(c.img, uintptr(unsafe.Pointer(&h)))
	return h
}

// Brush は塗りの色か、上から下への階調。
type Brush struct{ h uintptr }

func SolidBrush(c ARGB) Brush {
	var b Brush
	pGdipCreateSolidFill.Call(uintptr(c), uintptr(unsafe.Pointer(&b.h)))
	return b
}

// VerticalGradient は矩形の上端から下端へ、colorsを等間隔に並べた階調。
func VerticalGradient(rc RectF, colors ...ARGB) Brush {
	var b Brush
	// 端の画素で色が折り返さないよう、矩形を少し広げてから作る
	r := RectF{rc.X, rc.Y - 0.5, rc.W, rc.H + 1}
	pGdipCreateLineBrushFromRect.Call(uintptr(unsafe.Pointer(&r)), uintptr(colors[0]), uintptr(colors[len(colors)-1]),
		1, 0, uintptr(unsafe.Pointer(&b.h))) // LinearGradientModeVertical, WrapModeTile
	if len(colors) > 2 {
		positions := make([]float32, len(colors))
		for i := range colors {
			positions[i] = float32(i) / float32(len(colors)-1)
		}
		pGdipSetLinePresetBlend.Call(b.h, uintptr(unsafe.Pointer(&colors[0])), uintptr(unsafe.Pointer(&positions[0])), uintptr(len(colors)))
	}
	return b
}

func (b Brush) Free() {
	if b.h != 0 {
		pGdipDeleteBrush.Call(b.h)
	}
}

// Path は塗りつぶす形。図形を足していき、Fillで描く。重なった部分は交互の規則で抜ける（輪や縁を作るため）
type Path struct{ h uintptr }

func NewPath() *Path {
	p := &Path{}
	pGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&p.h))) // FillModeAlternate
	return p
}

func (p *Path) Free() {
	if p.h != 0 {
		pGdipDeletePath.Call(p.h)
		p.h = 0
	}
}

// 4分の1の円を3次ベジェで近似するときの係数
const kappa = 0.5522847498

// RoundRect は角の丸い矩形を足す。
func (p *Path) RoundRect(x, y, w, h, r float32) *Path {
	r = min(r, w/2, h/2)
	if r <= 0 {
		return p.Polygon(PointF{x, y}, PointF{x + w, y}, PointF{x + w, y + h}, PointF{x, y + h})
	}
	x1, y1 := x+w, y+h
	k := r * kappa
	line := func(a, b PointF) []PointF { return []PointF{a, b, b} }
	pts := []PointF{{x + r, y}}
	pts = append(pts, line(PointF{x + r, y}, PointF{x1 - r, y})...)
	pts = append(pts, PointF{x1 - r + k, y}, PointF{x1, y + r - k}, PointF{x1, y + r})
	pts = append(pts, line(PointF{x1, y + r}, PointF{x1, y1 - r})...)
	pts = append(pts, PointF{x1, y1 - r + k}, PointF{x1 - r + k, y1}, PointF{x1 - r, y1})
	pts = append(pts, line(PointF{x1 - r, y1}, PointF{x + r, y1})...)
	pts = append(pts, PointF{x + r - k, y1}, PointF{x, y1 - r + k}, PointF{x, y1 - r})
	pts = append(pts, line(PointF{x, y1 - r}, PointF{x, y + r})...)
	pts = append(pts, PointF{x, y + r - k}, PointF{x + r - k, y}, PointF{x + r, y})
	pGdipStartPathFigure.Call(p.h)
	pGdipAddPathBeziers.Call(p.h, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))
	pGdipClosePathFigure.Call(p.h)
	return p
}

// Ellipse は矩形に内接する楕円を足す。
func (p *Path) Ellipse(x, y, w, h float32) *Path {
	rx, ry := w/2, h/2
	cx, cy := x+rx, y+ry
	kx, ky := rx*kappa, ry*kappa
	pts := []PointF{
		{cx + rx, cy},
		{cx + rx, cy + ky}, {cx + kx, cy + ry}, {cx, cy + ry},
		{cx - kx, cy + ry}, {cx - rx, cy + ky}, {cx - rx, cy},
		{cx - rx, cy - ky}, {cx - kx, cy - ry}, {cx, cy - ry},
		{cx + kx, cy - ry}, {cx + rx, cy - ky}, {cx + rx, cy},
	}
	pGdipStartPathFigure.Call(p.h)
	pGdipAddPathBeziers.Call(p.h, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))
	pGdipClosePathFigure.Call(p.h)
	return p
}

// Polygon は多角形を足す。
func (p *Path) Polygon(pts ...PointF) *Path {
	pGdipAddPathPolygon.Call(p.h, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))
	return p
}

// Fill はパスを塗る。
func (c *Canvas) Fill(p *Path, b Brush) {
	pGdipFillPath.Call(c.g, b.h, p.h)
}

// FillColor は色1つで塗り、パスとブラシを片付ける。
func (c *Canvas) FillColor(p *Path, color ARGB) {
	b := SolidBrush(color)
	c.Fill(p, b)
	b.Free()
	p.Free()
}

// Font はGDI+の字体と、文字を縦に揃えるための寸法（画素）。
type Font struct {
	h uintptr
	// Ascent はセルの上端から基準線まで、Size は1emの大きさ
	Ascent, Size float32
}

// NewFont は字体の名前と1emの画素数から作る。weightは400が標準、600が中太
func NewFont(face string, px float32, weight int32) *Font {
	lf := LOGFONTW{Height: -int32(math.Round(float64(px))), Weight: weight, Quality: 4, CharSet: 1} // ANTIALIASED_QUALITY, DEFAULT_CHARSET
	name, _ := windows.UTF16FromString(face)
	copy(lf.FaceName[:31], name)
	screen := GetDC(0)
	defer ReleaseDC(0, screen)
	f := &Font{Size: float32(math.Round(float64(px)))}
	pGdipCreateFontFromLogfontW.Call(uintptr(screen), uintptr(unsafe.Pointer(&lf)), uintptr(unsafe.Pointer(&f.h)))
	if f.h == 0 {
		return f
	}
	var family uintptr
	var style int32
	var ascent, em uint16
	pGdipGetFamily.Call(f.h, uintptr(unsafe.Pointer(&family)))
	pGdipGetFontStyle.Call(f.h, uintptr(unsafe.Pointer(&style)))
	pGdipGetCellAscent.Call(family, uintptr(style), uintptr(unsafe.Pointer(&ascent)))
	pGdipGetEmHeight.Call(family, uintptr(style), uintptr(unsafe.Pointer(&em)))
	var size float32
	pGdipGetFontSize.Call(f.h, uintptr(unsafe.Pointer(&size)))
	if size > 0 {
		f.Size = size
	}
	if em > 0 {
		f.Ascent = f.Size * float32(ascent) / float32(em)
	}
	pGdipDeleteFontFamily.Call(family)
	return f
}

func (f *Font) Free() {
	if f.h != 0 {
		pGdipDeleteFont.Call(f.h)
		f.h = 0
	}
}

var typographic uintptr

// stringFormat は余白を足さず、末尾の空白も幅に数え、折り返さない書式。
func stringFormat() uintptr {
	if typographic == 0 {
		var generic uintptr
		pGdipStringFormatGetGenericTypographic.Call(uintptr(unsafe.Pointer(&generic)))
		pGdipCloneStringFormat.Call(generic, uintptr(unsafe.Pointer(&typographic)))
		// StringFormatFlagsMeasureTrailingSpaces | NoWrap | NoClip
		pGdipSetStringFormatFlags.Call(typographic, 0x0800|0x1000|0x4000)
	}
	return typographic
}

// TextWidth は文字列の幅（画素）。描く前に部品の幅を決めるのに使う
func (c *Canvas) TextWidth(text string, f *Font) float32 {
	if text == "" {
		return 0
	}
	s := UTF16Slice(text)
	layout := RectF{0, 0, 100000, 1000}
	var box RectF
	var fitted, lines int32
	pGdipMeasureString.Call(c.g, uintptr(unsafe.Pointer(&s[0])), uintptr(len(s)-1), f.h, uintptr(unsafe.Pointer(&layout)),
		stringFormat(), uintptr(unsafe.Pointer(&box)), uintptr(unsafe.Pointer(&fitted)), uintptr(unsafe.Pointer(&lines)))
	return box.W
}

// DrawText は大文字と数字の高さの中心がcenterYに来るように描く（Segoe UIの大文字の高さは約0.7em）。
func (c *Canvas) DrawText(text string, f *Font, color ARGB, x, centerY float32) {
	if text == "" {
		return
	}
	s := UTF16Slice(text)
	baseline := centerY + f.Size*0.7/2
	layout := RectF{x, baseline - f.Ascent, 100000, f.Size * 3}
	b := SolidBrush(color)
	defer b.Free()
	pGdipDrawString.Call(c.g, uintptr(unsafe.Pointer(&s[0])), uintptr(len(s)-1), f.h, uintptr(unsafe.Pointer(&layout)),
		stringFormat(), b.h)
}

var pGdipDrawImageI = gdiplus.NewProc("GdipDrawImageI")

// DrawCanvas は別の画用紙の絵を、整数の位置に重ねる（開発用の書き出しで並べるため）
func (c *Canvas) DrawCanvas(src *Canvas, x, y int32) {
	pGdipDrawImageI.Call(c.g, src.img, uintptr(x), uintptr(y))
}

// FillRect は矩形を色で塗る
func (c *Canvas) FillRect(x, y, w, h float32, color ARGB) {
	c.FillColor(NewPath().Polygon(PointF{x, y}, PointF{x + w, y}, PointF{x + w, y + h}, PointF{x, y + h}), color)
}
