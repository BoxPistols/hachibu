//go:build windows

package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/BoxPistols/hachibu/windows/internal/core"
)

func utf16le(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = append(b, byte(u), byte(u>>8))
	}
	return b
}

func TestDecodesWSLOutputInBothEncodings(t *testing.T) {
	// 既定のUTF-16LE（BOM付きと無し）と、WSL_UTF8=1のときのUTF-8
	for _, in := range [][]byte{
		utf16le("Ubuntu\r\ndocker-desktop\r\n"),
		append([]byte{0xFF, 0xFE}, utf16le("Ubuntu\r\ndocker-desktop\r\n")...),
		[]byte("Ubuntu\r\ndocker-desktop\r\n"),
	} {
		if got := decodeWSLOutput(in); !strings.HasPrefix(got, "Ubuntu\r\n") || !strings.Contains(got, "docker-desktop") {
			t.Errorf("%q", got)
		}
	}
}

func TestSetUTF16TruncatesAndTerminates(t *testing.T) {
	buf := make([]uint16, 5)
	setUTF16(buf, "Hachibu")
	if string(utf16.Decode(buf[:4])) != "Hach" || buf[4] != 0 {
		t.Fatalf("%v", buf)
	}
	setUTF16(buf, "ab")
	if buf[2] != 0 {
		t.Fatalf("%v", buf)
	}
}

func TestPrefsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	p := &Prefs{path: path, values: map[string]any{}}
	if p.UsageAPIConsent() != nil || !p.UpdateAutoCheck() || p.RestLayout() != core.LayoutFull || p.Opacity() != 1 {
		t.Fatal("defaults")
	}
	p.Set("usageAPIConsent", false)
	p.Set("restLayout", "tab")
	p.Set("opacity", 0.7)
	p.SetThresholds(core.Thresholds{Warning: 60, Critical: 85})
	p.SetStripAnchor(&stripAnchor{Edge: "right", X: 1900, Y: -20})
	p.SetLastUpdateCheck(time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC))

	// 書いた直後に同じプロセスで読み直せること（int32で入れた帯の位置を読めず、初期位置へ跳ね戻ったことがある）
	if a, ok := p.StripAnchor(); !ok || a != (stripAnchor{Edge: "right", X: 1900, Y: -20}) {
		t.Errorf("anchor in memory %+v %v", a, ok)
	}
	if p.Thresholds() != (core.Thresholds{Warning: 60, Critical: 85}) || p.RestLayout() != core.LayoutTab || p.Opacity() != 0.7 {
		t.Error("values in memory")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	q := &Prefs{path: path, values: map[string]any{}}
	if err := json.Unmarshal(data, &q.values); err != nil {
		t.Fatal(err)
	}
	if c := q.UsageAPIConsent(); c == nil || *c {
		t.Error("consent")
	}
	if q.RestLayout() != core.LayoutTab || q.Opacity() != 0.7 || q.Thresholds() != (core.Thresholds{Warning: 60, Critical: 85}) {
		t.Error("layout/opacity/thresholds")
	}
	if a, ok := q.StripAnchor(); !ok || a != (stripAnchor{Edge: "right", X: 1900, Y: -20}) {
		t.Errorf("anchor %+v", a)
	}
	if !q.LastUpdateCheck().Equal(time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Error("last update check")
	}
	// 読めない値は既定に戻す
	q.values["opacity"] = 0.42
	q.values["restLayout"] = "compact"
	if q.Opacity() != 1 || q.RestLayout() != core.LayoutFull {
		t.Error("fallbacks")
	}
}

func TestTrayTipFitsIn127Characters(t *testing.T) {
	L = core.Ja
	defer func() { L = core.En }()
	now := time.Now()
	limits := []core.UsageLimit{
		{Kind: core.KindFiveHour, Percent: 42, ResetsAt: now.Add(2 * time.Hour)},
		{Kind: core.KindWeekly, Percent: 76, ResetsAt: now.Add(72 * time.Hour)},
		{Kind: core.KindModel, Model: "Examplemodel with a long name", Percent: 93, ResetsAt: now.Add(48 * time.Hour)},
	}
	slot := core.Slot{ID: core.StatusID, Text: core.StatusText("Opus5 1M", "xhigh", limits)}
	a := &App{strip: &Strip{allSlots: []core.Slot{slot}}, display: core.Display{Slot: slot, Limits: limits}}
	tip := a.trayTip()
	if n := len([]rune(tip)); n > 127 || n == 0 {
		t.Fatalf("%d: %q", n, tip)
	}
	lines := strings.Split(tip, "\n")
	if lines[0] != core.AppName || lines[1] != "Opus5 · 1M context · xhigh · S42 W76 E93" {
		t.Fatalf("%q", lines)
	}
	// 入り切らない行は途中で切らずに落とす
	for _, line := range lines[2:] {
		if !strings.HasSuffix(line, "）") && !strings.HasSuffix(line, "%") {
			t.Errorf("cut line %q", line)
		}
	}
}

func TestExpandsKnownFolderPaths(t *testing.T) {
	// FOLDERID_LocalAppData
	got := expandKnownFolder(`{F1B32785-6FBA-4FCF-9D55-7B8E7F157091}\Programs\Hachibu\Hachibu.exe`)
	if strings.HasPrefix(got, "{") || !strings.HasSuffix(got, `\Programs\Hachibu\Hachibu.exe`) {
		t.Fatalf("%q", got)
	}
	if got := expandKnownFolder(`C:\Tools\Hachibu.exe`); got != `C:\Tools\Hachibu.exe` {
		t.Fatalf("%q", got)
	}
}

func TestMenusListEveryActionInBothLanguages(t *testing.T) {
	now := time.Now()
	limits := []core.UsageLimit{{Kind: core.KindFiveHour, Percent: 42, ResetsAt: now}, {Kind: core.KindWeekly, Percent: 95, ResetsAt: now}}
	slot := core.Slot{ID: core.StatusID, Text: core.StatusText("Opus5 1M", "xhigh", limits)}
	defer func() { L = core.En }()
	for _, lang := range core.AllLanguages {
		L = core.StringsFor(lang)
		a := snapshotApp(limits, slot)
		for _, build := range []func(*menu){a.buildStripMenu, a.buildTrayMenu} {
			set := newMenuSet(1)
			set.dump = &strings.Builder{}
			build(set.root())
			text := set.dump.String()
			for _, want := range []string{L.MenuQuit, L.MenuChangeShortcut, L.MenuLaunchAtLogin, L.MenuOpenLog, L.MenuLanguage, L.MenuLegend, L.MenuUsageAPI} {
				if !strings.Contains(text, want) {
					t.Errorf("%s menu is missing %q", lang, want)
				}
			}
		}
	}
}
