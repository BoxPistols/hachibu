package core

import (
	"reflect"
	"strings"
	"testing"
)

func TestDefaultIsSeventyAndNinety(t *testing.T) {
	if DefaultThresholds != (Thresholds{70, 90}) {
		t.Fatal(DefaultThresholds)
	}
}

func TestLevelsAtBoundaries(t *testing.T) {
	th := DefaultThresholds
	for p, want := range map[int]UsageLevel{69: LevelNormal, 70: LevelWarning, 89: LevelWarning, 90: LevelCritical, 100: LevelCritical} {
		if th.Level(p) != want {
			t.Errorf("%d → %v", p, th.Level(p))
		}
	}
}

func TestRejectsRedBelowOrEqualYellow(t *testing.T) {
	for _, c := range [][2]int{{80, 80}, {85, 80}, {0, 80}, {50, 101}} {
		if _, ok := NewThresholds(c[0], c[1]); ok {
			t.Errorf("%v", c)
		}
	}
	if _, ok := NewThresholds(50, 80); !ok {
		t.Error("50/80")
	}
	for _, w := range WarningChoices {
		for _, c := range CriticalChoices {
			if c > w {
				if _, ok := NewThresholds(w, c); !ok {
					t.Errorf("%d/%d", w, c)
				}
			}
		}
	}
}

func TestWorstOfLimits(t *testing.T) {
	limits := []UsageLimit{{Kind: KindFiveHour, Percent: 12}, {Kind: KindWeekly, Percent: 76}}
	if DefaultThresholds.Worst(limits) != LevelWarning || DefaultThresholds.Worst(nil) != LevelNormal {
		t.Fatal("worst")
	}
}

func TestMarksOnlyUsageTokensOverThreshold(t *testing.T) {
	got := Segments("Opus5 1M xhigh · S12 W76 F95", DefaultThresholds)
	want := []TextSegment{
		{"Opus5 1M xhigh · S12 ", LevelNormal},
		{"W76", LevelWarning},
		{" ", LevelNormal},
		{"F95", LevelCritical},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
}

func TestKeepsTextIntactWhenJoinedBack(t *testing.T) {
	text := "Opus5 1M xhigh · S72 W91 F55 ctx91%"
	var b strings.Builder
	for _, s := range Segments(text, DefaultThresholds) {
		b.WriteString(s.Text)
	}
	if b.String() != text {
		t.Fatalf("%q", b.String())
	}
}

func TestLeavesNonUsageTokensAlone(t *testing.T) {
	// モデル名の数字、コンテキスト使用率は使用率ではない
	for _, text := range []string{"Opus5 1M xhigh", "ctx95%", "1 Def", "CC …", "許可待ち"} {
		if got := Segments(text, DefaultThresholds); !reflect.DeepEqual(got, []TextSegment{{text, LevelNormal}}) {
			t.Errorf("%q → %+v", text, got)
		}
	}
}

func TestUsageOnlySegments(t *testing.T) {
	got := Segments("S95 W50", DefaultThresholds)
	if !reflect.DeepEqual(got, []TextSegment{{"S95", LevelCritical}, {" W50", LevelNormal}}) {
		t.Fatalf("%+v", got)
	}
}
