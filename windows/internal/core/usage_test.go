package core

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
	"unicode"
)

// parseJSON は試験用のJSONを、アプリが読むときと同じ形（map[string]any、数はfloat64）にする
func parseJSON(t *testing.T, text string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// 公式ドキュメントのstatusLineのJSONの形。値は架空
const statusLinePayload = `{
  "model": {"id": "example-model-id", "display_name": "Opus 5 (1M context)"},
  "effort": {"level": "xhigh"},
  "rate_limits": {
    "five_hour": {"used_percentage": 9, "resets_at": 1900000000},
    "seven_day": {"used_percentage": 75.4, "resets_at": 1900100000}
  }
}`

func TestStatusLineReadsModelEffortAndLimits(t *testing.T) {
	s := ParseStatusLine(parseJSON(t, statusLinePayload))
	if s.Model != "Opus5 1M" || s.Effort != "xhigh" {
		t.Fatalf("model/effort = %q %q", s.Model, s.Effort)
	}
	if len(s.Limits) != 2 || s.Limits[0].Kind != KindFiveHour || s.Limits[1].Kind != KindWeekly {
		t.Fatalf("limits = %+v", s.Limits)
	}
	if s.Limits[0].Percent != 9 || s.Limits[1].Percent != 75 {
		t.Fatalf("percent = %d %d", s.Limits[0].Percent, s.Limits[1].Percent)
	}
	if !s.Limits[0].ResetsAt.Equal(time.Unix(1_900_000_000, 0)) {
		t.Fatalf("resets = %v", s.Limits[0].ResetsAt)
	}
}

func TestStatusLineLimitsAreEmptyBeforeTheFirstResponse(t *testing.T) {
	p := parseJSON(t, statusLinePayload)
	delete(p, "rate_limits")
	if len(ParseStatusLine(p).Limits) != 0 {
		t.Fatal("limits should be empty")
	}
	if got := ParseStatusLine(map[string]any{}); got.Model != "" || got.Effort != "" || got.Limits != nil {
		t.Fatalf("empty payload = %+v", got)
	}
}

func TestCompactsDisplayNames(t *testing.T) {
	for in, want := range map[string]string{
		"Opus 5 (1M context)":    "Opus5 1M",
		"Sonnet 5":               "Sonnet5",
		"Haiku 4.5":              "Haiku4.5",
		"Example (200k context)": "Example 200K",
	} {
		if got := CompactDisplayName(in); got != want {
			t.Errorf("CompactDisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildsTouchBarStyleStatus(t *testing.T) {
	limits := []UsageLimit{
		{Kind: KindFiveHour, Percent: 9},
		{Kind: KindWeekly, Percent: 75},
		{Kind: KindModel, Model: "Examplemodel", Percent: 55},
	}
	cases := []struct {
		model, effort string
		limits        []UsageLimit
		want          string
	}{
		{"Opus5 1M", "xhigh", limits, "Opus5 1M xhigh · S9 W75 E55"},
		{"", "", limits, "S9 W75 E55"},
		{"Opus5", "", nil, "Opus5"},
		{"", "", nil, StatusPlaceholder},
	}
	for _, c := range cases {
		if got := StatusText(c.model, c.effort, c.limits); got != c.want {
			t.Errorf("StatusText = %q, want %q", got, c.want)
		}
	}
}

func TestParsesFractionalSecondsOfAnyLength(t *testing.T) {
	a, okA := ParseISO("2030-01-02T03:04:05.865631+00:00")
	b, okB := ParseISO("2030-01-02T03:04:05.8+00:00")
	c, okC := ParseISO("2030-01-02T03:04:05Z")
	if !okA || !okB || !okC || !a.Equal(b) || !b.Equal(c) {
		t.Fatalf("%v %v %v", a, b, c)
	}
	if _, ok := ParseISO("not a date"); ok {
		t.Fatal("should not parse")
	}
}

func tokyo(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		// tzdataが無い環境でも同じ時差で試す
		loc = time.FixedZone("JST", 9*3600)
	}
	return loc
}

func TestFormatsResetTimeRelativeToToday(t *testing.T) {
	loc := tokyo(t)
	now := time.Date(2030, 1, 2, 9, 0, 0, 0, loc)
	sameDay := time.Date(2030, 1, 2, 23, 5, 0, 0, loc)
	later := time.Date(2030, 1, 5, 7, 30, 0, 0, loc)
	if got := ResetText(sameDay, now, Ja); got != "23:05" {
		t.Errorf("same day = %q", got)
	}
	if got := ResetText(later, now, Ja); got != "1/5(土) 7:30" {
		t.Errorf("later = %q", got)
	}
}

func TestNoSpaceBetweenJapaneseAndDigits(t *testing.T) {
	now := time.Now()
	if got := (UsageLimit{Kind: KindWeekly, Percent: 75}).Line(Ja, now); got != "週枠75%" {
		t.Errorf("%q", got)
	}
	if got := (UsageLimit{Kind: KindModel, Model: "Examplemodel", Percent: 55}).Line(Ja, now); got != "Examplemodel 55%" {
		t.Errorf("%q", got)
	}
}

func TestLimitLinesInEnglish(t *testing.T) {
	now := time.Now()
	if got := (UsageLimit{Kind: KindWeekly, Percent: 75}).Line(En, now); got != "Weekly 75%" {
		t.Errorf("%q", got)
	}
	if got := (UsageLimit{Kind: KindFiveHour, Percent: 9}).Line(En, now); got != "5-hour 9%" {
		t.Errorf("%q", got)
	}
}

func TestResetTimeInEnglish(t *testing.T) {
	now := time.Date(2030, 1, 2, 9, 0, 0, 0, time.UTC)
	later := time.Date(2030, 1, 5, 7, 30, 0, 0, time.UTC)
	limit := UsageLimit{Kind: KindWeekly, Percent: 80, ResetsAt: later}
	if got := limit.Line(En, now); got != "Weekly 80% (resets Sat 1/5 7:30)" {
		t.Errorf("%q", got)
	}
	if got := limit.Line(Ja, now); got != "週枠80%（1/5(土) 7:30にリセット）" {
		t.Errorf("%q", got)
	}
}

func TestPicksLanguageFromPreferencesAndOverride(t *testing.T) {
	cases := []struct {
		preferred       []string
		saved, override string
		want            Language
	}{
		{[]string{"ja-JP", "en-US"}, "", "", Japanese},
		{[]string{"en-US", "ja-JP"}, "", "", English},
		{[]string{"fr-FR"}, "", "", English},
		{[]string{"ja-JP"}, "", "en", English},
		{[]string{"en-US"}, "", "xx", English},
		// メニューで選んだ言語はWindowsの設定より優先する
		{[]string{"ja-JP"}, "en", "", English},
		{[]string{"en-US"}, "ja", "", Japanese},
		// 撮影用の環境変数はメニューの選択よりも優先する
		{[]string{"ja-JP"}, "ja", "en", English},
		// 読めない値が保存されていたらWindowsに合わせる
		{[]string{"ja-JP"}, "xx", "", Japanese},
		{nil, "", "", English},
	}
	for _, c := range cases {
		if got := DetectLanguage(c.preferred, c.saved, c.override); got != c.want {
			t.Errorf("DetectLanguage(%v, %q, %q) = %v", c.preferred, c.saved, c.override, got)
		}
	}
}

func TestReadsTheContextLengthFromTheWindowSize(t *testing.T) {
	cases := map[string]string{
		`{"model":{"display_name":"Examplemodel 5.1"},"context_window":{"context_window_size":1000000}}`:            "Examplemodel5.1 1M",
		`{"model":{"display_name":"Examplemodel 5.1"},"context_window":{"context_window_size":200000}}`:             "Examplemodel5.1",
		`{"model":{"display_name":"Examplemodel 5 (1M context)"},"context_window":{"context_window_size":1000000}}`: "Examplemodel5 1M",
	}
	for in, want := range cases {
		if got := ParseStatusLine(parseJSON(t, in)).Model; got != want {
			t.Errorf("%s → %q, want %q", in, got, want)
		}
	}
}

func hasCJK(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana) || (r >= 0x3000 && r <= 0x303F) {
			return true
		}
	}
	return false
}

func TestEnglishLogLinesHaveNoJapanese(t *testing.T) {
	s := En
	lines := []string{
		s.LogShortcutCommitted("Ctrl+Alt+/"), s.LogShortcutDisabled, s.LogShortcutFailed("Ctrl+Alt+/"),
		s.LogUsageAPIEnabled, s.LogUsageAPIDeclined, s.LogUsageAPIDisabled,
		s.LogLoginItemOn, s.LogLoginItemOff, s.LogOpened, s.LogLanguageChanged("English"),
		s.UsageAPIFailed("HTTP 500"), s.LoginItemFailed("example error"), s.LogClaudeDirs("Windows"),
	}
	for _, line := range lines {
		if hasCJK(line) {
			t.Errorf("%q", line)
		}
	}
}

func TestEveryModeAndStyleHasAnEnglishName(t *testing.T) {
	for _, l := range AllLayouts {
		if En.LayoutName(l) == "" || En.LayoutName(l) == Ja.LayoutName(l) {
			t.Errorf("layout %v", l)
		}
	}
	for _, st := range AllTrayStyles {
		if En.TrayStyleName(st) == Ja.TrayStyleName(st) {
			t.Errorf("tray style %v", st)
		}
	}
}

// 英語と日本語の表で、空の文言が無いこと（片方だけ書き忘れると、その言語で空の項目が出る）
func TestEveryStringIsFilledInBothLanguages(t *testing.T) {
	for _, s := range []*Strings{En, Ja} {
		v := reflect.ValueOf(*s)
		for i := range v.NumField() {
			f := v.Field(i)
			name := v.Type().Field(i).Name
			switch f.Kind() {
			case reflect.String:
				if f.String() == "" {
					t.Errorf("%s.%s is empty", s.Language, name)
				}
			case reflect.Func:
				if f.IsNil() {
					t.Errorf("%s.%s is nil", s.Language, name)
				}
			}
		}
	}
}

func TestStaleness(t *testing.T) {
	now := time.Unix(2_000_000_000, 0)
	if IsStale(time.Time{}, now) {
		t.Error("unknown time is not stale")
	}
	if !IsStale(now.Add(-31*time.Minute), now) || IsStale(now.Add(-29*time.Minute), now) {
		t.Error("thirty minutes")
	}
}
