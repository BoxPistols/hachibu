package core

import (
	"strings"
	"testing"
	"time"
)

// 2026-09-18に実際の応答で確認した形。値は架空
const usageAPIResponse = `{
  "five_hour": {"utilization": 2.0, "resets_at": "2030-01-01T10:00:00.807700+00:00"},
  "seven_day": {"utilization": 81.0, "resets_at": "2030-01-03T00:00:00.807721+00:00"},
  "limits": [
    {"kind": "session", "percent": 2, "resets_at": "2030-01-01T10:00:00.807700+00:00", "scope": null},
    {"kind": "weekly_all", "percent": 81, "resets_at": "2030-01-03T00:00:00.807721+00:00", "scope": null},
    {"kind": "weekly_scoped", "percent": 55, "resets_at": "2030-01-02T23:59:59.807925+00:00",
     "scope": {"model": {"id": null, "display_name": "Examplemodel"}, "surface": null}},
    {"kind": "something_new", "percent": 3}
  ]
}`

func percents(limits []UsageLimit) []int {
	var out []int
	for _, l := range limits {
		out = append(out, l.Percent)
	}
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestUsageAPIReadsTheThreeLimits(t *testing.T) {
	limits := ParseUsageAPI(parseJSON(t, usageAPIResponse))
	if len(limits) != 3 || limits[0].Kind != KindFiveHour || limits[1].Kind != KindWeekly ||
		limits[2].Kind != KindModel || limits[2].Model != "Examplemodel" {
		t.Fatalf("%+v", limits)
	}
	if !equalInts(percents(limits), []int{2, 81, 55}) {
		t.Fatalf("%v", percents(limits))
	}
	if got := StatusText("Opus5 1M", "xhigh", limits); got != "Opus5 1M xhigh · S2 W81 E55" {
		t.Fatalf("%q", got)
	}
}

func TestUsageAPIFallsBackToUtilizationAsPercent(t *testing.T) {
	obj := parseJSON(t, usageAPIResponse)
	delete(obj, "limits")
	// utilizationは百分率。2.0を比率と見て200%にも、0.5を50%にもしない
	if got := percents(ParseUsageAPI(obj)); !equalInts(got, []int{2, 81}) {
		t.Fatalf("%v", got)
	}
	if got := percents(ParseUsageAPI(parseJSON(t, `{"five_hour": {"utilization": 0.5}}`))); !equalInts(got, []int{1}) {
		t.Fatalf("%v", got)
	}
}

func TestUsageAPISkipsScopedLimitWithoutName(t *testing.T) {
	obj := parseJSON(t, `{"limits": [{"kind": "weekly_scoped", "percent": 10, "scope": {"model": {"display_name": " "}}}]}`)
	if got := ParseUsageAPI(obj); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestCompactsModelIDs(t *testing.T) {
	for in, want := range map[string]string{
		"claude-opus-5":             "Opus5",
		"claude-opus-5[1m]":         "Opus5 1M",
		"claude-sonnet-5-20260101":  "Sonnet5",
		"claude-haiku-4-5-20251001": "Haiku4.5",
	} {
		if got, ok := CompactModelID(in); !ok || got != want {
			t.Errorf("CompactModelID(%q) = %q", in, got)
		}
	}
	if _, ok := CompactModelID("gpt-4"); ok {
		t.Error("gpt-4")
	}
}

func TestFindsTheLastModelInTheTranscript(t *testing.T) {
	tail := `{"type":"assistant","message":{"model":"claude-sonnet-5","content":[]}}
{"type":"user","message":{"content":"model is fine"}}
{"type":"assistant","message":{"model":"claude-opus-5","content":[]}}`
	if got := LastModelID(tail); got != "claude-opus-5" {
		t.Errorf("%q", got)
	}
	if got := LastModelID(`{"type":"user"}`); got != "" {
		t.Errorf("%q", got)
	}
}

func TestFindsTheLastEffortInTheTranscript(t *testing.T) {
	tail := `{"type":"assistant","message":{"model":"claude-opus-5-5","content":[]},"effort":"xhigh","perTurnEffort":"xhigh"}
{"type":"user","message":{"content":"effort is fine"}}
{"type":"assistant","message":{"model":"claude-opus-5-5","content":[]},"effort":"medium","perTurnEffort":"medium"}`
	if got := LastEffort(tail); got != "medium" {
		t.Errorf("%q", got)
	}
	if got := LastEffort(`{"type":"user"}`); got != "" {
		t.Errorf("%q", got)
	}
}

func TestCombinesTranscriptModelWithContextFromSettings(t *testing.T) {
	cases := []struct{ id, settings, want string }{
		{"claude-opus-5", "opus[1m]", "Opus5 1M"},
		{"claude-opus-5[1m]", "opus[1m]", "Opus5 1M"},
		{"claude-sonnet-5", "", "Sonnet5"},
		// 設定が別の系統のモデルを指しているときは、その1Mを付けない
		{"claude-sonnet-5", "opus[1m]", "Sonnet5"},
		{"claude-sonnet-5", "claude-sonnet-5[1m]", "Sonnet5 1M"},
		// 会話記録が無ければ設定の指定から
		{"", "opus[1m]", "Opus 1M"},
		{"", "sonnet", "Sonnet"},
		{"", "", ""},
	}
	for _, c := range cases {
		if got := ModelDisplayName(c.id, c.settings); got != c.want {
			t.Errorf("ModelDisplayName(%q, %q) = %q, want %q", c.id, c.settings, got, c.want)
		}
	}
}

func TestComposePrefersANewerStatusLine(t *testing.T) {
	now := time.Date(2030, 1, 2, 9, 0, 0, 0, time.UTC)
	st := SourceState{
		StatusLine:         &StatusLineSnapshot{Model: "Opus5 1M", Effort: "xhigh"},
		StatusLineModified: now.Add(-10 * time.Second),
		StatusLineLimits:   []UsageLimit{{Kind: KindFiveHour, Percent: 9}, {Kind: KindWeekly, Percent: 75}},
		StatusLineLimitsAt: now.Add(-10 * time.Second),
		TranscriptModelID:  "claude-sonnet-5",
		TranscriptEffort:   "low",
		TranscriptModified: now.Add(-20 * time.Second),
	}
	d := Compose(st, En, now)
	if d.Slot.Text != "Opus5 1M xhigh · S9 W75" || d.Slot.Dimmed {
		t.Fatalf("%+v", d.Slot)
	}
	// 会話記録の方が1分より新しければ、そちらのモデルとeffortを使う（使用率はstatusLineのまま）
	st.TranscriptModified = now.Add(2 * time.Minute)
	if got := Compose(st, En, now).Slot.Text; got != "Sonnet5 low · S9 W75" {
		t.Fatalf("%q", got)
	}
}

func TestComposeFallsBackToSettings(t *testing.T) {
	now := time.Now()
	d := Compose(SourceState{SettingsModel: "opus[1m]", SettingsEffort: "high"}, Ja, now)
	if d.Slot.Text != "Opus 1M high" || !d.Slot.Dimmed || d.Slot.Help != Ja.BasicNoUsage {
		t.Fatalf("%+v", d.Slot)
	}
}

func TestComposeUsesTheAPIOnlyWhenItHasValues(t *testing.T) {
	now := time.Now()
	st := SourceState{
		StatusLineLimits:   []UsageLimit{{Kind: KindFiveHour, Percent: 9}},
		StatusLineLimitsAt: now,
		UseAPI:             true,
		APIFailure:         "HTTP 401",
	}
	d := Compose(st, En, now)
	if d.Slot.Text != "S9" || !strings.Contains(d.Slot.Help, "HTTP 401") {
		t.Fatalf("%+v", d.Slot)
	}
	st.APILimits = []UsageLimit{{Kind: KindFiveHour, Percent: 20}, {Kind: KindWeekly, Percent: 40}}
	st.APIAt = now
	st.APIFailure = ""
	if got := Compose(st, En, now).Slot.Text; got != "S20 W40" {
		t.Fatalf("%q", got)
	}
	// 有効にしていなければ、取得済みの値があっても使わない
	st.UseAPI = false
	if got := Compose(st, En, now).Slot.Text; got != "S9" {
		t.Fatalf("%q", got)
	}
}

func TestComposeDimsStaleUsageAndSaysWhen(t *testing.T) {
	now := time.Date(2030, 1, 2, 9, 0, 0, 0, time.UTC)
	st := SourceState{
		StatusLineLimits:   []UsageLimit{{Kind: KindWeekly, Percent: 50}},
		StatusLineLimitsAt: now.Add(-45 * time.Minute),
	}
	d := Compose(st, En, now)
	if !d.Slot.Dimmed || !strings.HasSuffix(d.Slot.Help, "Last updated 8:15") {
		t.Fatalf("%+v", d.Slot)
	}
}
