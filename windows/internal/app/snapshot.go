//go:build windows

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

// 開発用: --snapshot 出力先（またはHACHIBU_SNAPSHOT=出力先）で起動すると、窓を画面に出さずに帯とトレイのアイコンをPNGへ、
// メニューの項目を文字へ書き出して終わる。見た目の確認のたびに利用者の作業中の画面へ窓を出したり、カーソルを動かしたりしないためのもの
func snapshotDir() string {
	for i, arg := range os.Args {
		if arg == "--snapshot" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return os.Getenv("HACHIBU_SNAPSHOT")
}

// noRegistrar は書き出しでショートカットを実際には登録しない登録器
type noRegistrar struct{}

func (noRegistrar) Register(core.Shortcut) bool { return true }
func (noRegistrar) Unregister()                 {}

func runSnapshot(dir string) int {
	w32.SetPerMonitorDPIAware()
	if err := w32.StartGDIPlus(); err != nil {
		return 1
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 1
	}
	now := time.Now()
	limits := []core.UsageLimit{
		{Kind: core.KindFiveHour, Percent: 42, ResetsAt: now.Add(2 * time.Hour)},
		{Kind: core.KindWeekly, Percent: 76, ResetsAt: now.Add(3 * 24 * time.Hour)},
		{Kind: core.KindModel, Model: "Fable", Percent: 93, ResetsAt: now.Add(2 * 24 * time.Hour)},
	}
	slot := core.Slot{ID: core.StatusID, Text: core.StatusText("Opus5 1M", "xhigh", limits),
		Background: core.IdleBackground, Foreground: core.IdleForeground}
	usage := core.UsageOnlyText(limits)
	th := core.DefaultThresholds

	// 帯。拡大率100%と200%の両方で、モードごとに描く
	type variant struct {
		name  string
		slot  core.Slot
		modes []core.StripLayout
	}
	dimmed := core.Slot{ID: core.StatusID, Text: "Opus5 xhigh", Background: core.IdleBackground, Foreground: core.IdleForeground, Dimmed: true}
	placeholder := core.Slot{ID: core.StatusID, Text: core.StatusPlaceholder, Background: core.IdleBackground, Foreground: core.IdleForeground, Dimmed: true}
	critical := slot
	critical.Text = core.StatusText("Sonnet5", "high", []core.UsageLimit{{Kind: core.KindFiveHour, Percent: 95}, {Kind: core.KindWeekly, Percent: 81}})
	variants := []variant{
		{"basic", slot, []core.StripLayout{core.LayoutFull, core.LayoutUsage, core.LayoutTab}},
		{"no-usage", dimmed, []core.StripLayout{core.LayoutFull}},
		{"placeholder", placeholder, []core.StripLayout{core.LayoutFull}},
		{"critical", critical, []core.StripLayout{core.LayoutFull, core.LayoutTab}},
	}
	for _, scale := range []float32{1, 2} {
		for _, v := range variants {
			for _, mode := range v.modes {
				view := stripView{layout: mode, slots: core.VisibleSlots([]core.Slot{v.slot}, mode, usage), thresholds: th}
				c, _ := paintStrip(view, scale)
				writePNG(c, filepath.Join(dir, fmt.Sprintf("strip-%s-%s@%gx.png", v.name, mode, scale)))
			}
		}
	}

	// トレイのアイコン。暗いタスクバーと明るいタスクバーに、絵の種類と使用率ごとに並べる
	for _, size := range []int32{16, 32} {
		percents := []int{8, 42, 76, 93, 100}
		cell := size + size/2
		cols := int32(len(core.AllTrayStyles) * len(percents))
		sheet := w32.NewCanvas(cols*cell+cell/2, 2*cell+cell/2)
		for row, dark := range []bool{true, false} {
			bg := w32.Color(32, 32, 32, 1)
			if !dark {
				bg = w32.Color(238, 238, 238, 1)
			}
			sheet.FillRect(0, float32(int32(row)*cell), float32(sheet.W), float32(cell+cell/4), bg)
			col := int32(0)
			for _, style := range core.AllTrayStyles {
				for _, p := range percents {
					icon := paintTrayIcon(style, []core.UsageLimit{{Kind: core.KindWeekly, Percent: p}}, th, size, dark)
					sheet.DrawCanvas(icon, cell/2+col*cell, int32(row)*cell+cell/4)
					icon.Free()
					col++
				}
			}
		}
		writePNG(sheet, filepath.Join(dir, fmt.Sprintf("tray-%d.png", size)))
	}

	// メニューの項目
	for _, lang := range core.AllLanguages {
		L = core.StringsFor(lang)
		a := snapshotApp(limits, slot)
		var b strings.Builder
		for _, part := range []struct {
			title string
			build func(*menu)
		}{{"strip (right-click)", a.buildStripMenu}, {"tray icon", a.buildTrayMenu}} {
			set := newMenuSet(1)
			set.dump = &strings.Builder{}
			part.build(set.root())
			b.WriteString("== " + part.title + "\n" + set.dump.String() + "\n")
		}
		os.WriteFile(filepath.Join(dir, "menus-"+string(lang)+".txt"), []byte(b.String()), 0o644)
	}
	return 0
}

func writePNG(c *w32.Canvas, path string) {
	c.SavePNG(path)
	c.Free()
}

// snapshotApp はメニューを書き出すための、窓を持たないアプリ
func snapshotApp(limits []core.UsageLimit, slot core.Slot) *App {
	prefs := &Prefs{path: filepath.Join(os.TempDir(), "hachibu-snapshot-settings.json"), values: map[string]any{"usageAPIConsent": true}}
	a := &App{prefs: prefs}
	a.strip = &Strip{prefs: prefs, RestLayout: core.LayoutFull, Opacity: 1, Thresholds: core.DefaultThresholds, allSlots: []core.Slot{slot}}
	a.display = core.Display{Slot: slot, Limits: limits}
	a.hotKeys = core.NewHotKeyCenter(noRegistrar{}, core.SummonSettings(prefs))
	a.cycleKeys = core.NewHotKeyCenter(noRegistrar{}, core.LayoutCycleSettings(prefs))
	a.updates = &Updater{prefs: prefs, Available: &core.Release{Version: "0.2.0", URL: "https://example.com"}}
	return a
}
