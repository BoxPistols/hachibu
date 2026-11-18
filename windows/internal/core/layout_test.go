package core

import (
	"reflect"
	"strings"
	"testing"
)

func status(text string) Slot {
	return Slot{ID: StatusID, Text: text, Background: IdleBackground, Foreground: IdleForeground, Help: "reset info"}
}

func TestExpandingAlwaysShowsTheStandardStrip(t *testing.T) {
	for _, rest := range AllLayouts {
		if EffectiveLayout(rest, true) != LayoutFull || EffectiveLayout(rest, false) != rest {
			t.Errorf("%v", rest)
		}
	}
}

func TestUsageModeReplacesOnlyTheText(t *testing.T) {
	shown := VisibleSlots([]Slot{status("Opus5 1M xhigh · S12 W76")}, LayoutUsage, "S12 W76")
	if len(shown) != 1 || shown[0].Text != "S12 W76" || shown[0].Help != "reset info" || shown[0].Background != IdleBackground {
		t.Fatalf("%+v", shown)
	}
	if got := VisibleSlots([]Slot{status("Opus5")}, LayoutUsage, "")[0].Text; got != "Opus5" {
		t.Fatalf("%q", got)
	}
}

func TestTabAndHidden(t *testing.T) {
	s := []Slot{status("Opus5 1M xhigh · S12 W76")}
	if got := VisibleSlots(s, LayoutTab, ""); len(got) != 1 || got[0].ID != StatusID {
		t.Fatalf("%+v", got)
	}
	if got := VisibleSlots(s, LayoutHidden, ""); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
	if got := VisibleSlots(s, LayoutFull, ""); !reflect.DeepEqual(got, s) {
		t.Fatalf("%+v", got)
	}
	if _, ok := ParseLayout("compact"); ok {
		t.Fatal("compact is not a mode")
	}
}

func TestCyclingSkipsHidden(t *testing.T) {
	want := map[StripLayout]StripLayout{LayoutFull: LayoutUsage, LayoutUsage: LayoutTab, LayoutTab: LayoutFull, LayoutHidden: LayoutFull}
	for from, to := range want {
		if NextLayout(from) != to {
			t.Errorf("%v → %v", from, NextLayout(from))
		}
	}
}

func TestUsageOnlyText(t *testing.T) {
	limits := []UsageLimit{{Kind: KindFiveHour, Percent: 11}, {Kind: KindWeekly, Percent: 75}}
	if got := UsageOnlyText(limits); got != "S11 W75" {
		t.Fatalf("%q", got)
	}
	if got := UsageOnlyText(nil); got != "" {
		t.Fatalf("%q", got)
	}
}

func TestTheGaugeReadsTheWeeklyLimit(t *testing.T) {
	limits := []UsageLimit{{Kind: KindFiveHour, Percent: 38}, {Kind: KindWeekly, Percent: 68}, {Kind: KindModel, Model: "Examplemodel", Percent: 90}}
	if g, ok := GaugeFrom(limits); !ok || g != (GaugeReading{Label: "68", Percent: 68}) {
		t.Fatalf("%+v", g)
	}
	// 週枠が無ければ最初の枠。何も無ければ出さない
	if g, ok := GaugeFrom(limits[:1]); !ok || g != (GaugeReading{Label: "38", Percent: 38}) {
		t.Fatalf("%+v", g)
	}
	if _, ok := GaugeFrom(nil); ok {
		t.Fatal("no limits")
	}
}

func TestContextMarkTakesTheLengthOutOfTheStatusText(t *testing.T) {
	if s, ok := SplitContext("Examplemodel5 1M xhigh · S2 W81"); !ok || s != (ContextSplit{"Examplemodel5", "1M", " xhigh · S2 W81"}) {
		t.Fatalf("%+v", s)
	}
	if s, ok := SplitContext("Examplemodel5 200K"); !ok || s != (ContextSplit{"Examplemodel5", "200K", ""}) {
		t.Fatalf("%+v", s)
	}
	// 使用率の語（"M12"のような枠の頭文字＋数字）をコンテキスト長と取り違えない
	for _, text := range []string{"Examplemodel5 xhigh · S2", "S2 W81 F55", "Examplemodel5 M12 W81", "Examplemodel5 1Mx"} {
		if _, ok := SplitContext(text); ok {
			t.Errorf("%q", text)
		}
	}
	if got, ok := SpelledOut("Examplemodel5 1M xhigh · S2 W81"); !ok || got != "Examplemodel5 · 1M context · xhigh" {
		t.Fatalf("%q", got)
	}
	if _, ok := SpelledOut("Examplemodel5 xhigh · S2"); ok {
		t.Fatal("no context")
	}
}

func TestLegendExplainsEveryMarkOnTheStrip(t *testing.T) {
	limits := []UsageLimit{{Kind: KindFiveHour, Percent: 6}, {Kind: KindWeekly, Percent: 85}, {Kind: KindModel, Model: "Examplemodel", Percent: 57}}
	lines := Legend("Examplemodel5 1M xhigh · S6 W85 E57", limits, Thresholds{70, 90}, En)
	var texts []string
	for _, l := range lines {
		texts = append(texts, l.Text)
	}
	want := []string{
		"Dot after the model name: 1M context",
		"S: 5-hour limit",
		"W: Weekly limit",
		"E: weekly limit for Examplemodel",
		"70% or more",
		"90% or more",
	}
	if strings.Join(texts, "|") != strings.Join(want, "|") {
		t.Fatalf("%q", texts)
	}
	if lines[0].Mark != MarkContextDot || lines[len(lines)-1].Mark != MarkLevel || lines[len(lines)-1].Level != LevelCritical {
		t.Fatalf("%+v", lines)
	}
	lines = Legend("Examplemodel5 xhigh · S6", nil, Thresholds{50, 80}, Ja)
	if len(lines) != 2 || lines[0].Text != "50%以上" || lines[1].Text != "80%以上" {
		t.Fatalf("%+v", lines)
	}
}
