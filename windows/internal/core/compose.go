package core

import (
	"strings"
	"time"
)

// SourceState は、基本表示の供給元が集めた値。ファイルや通信から読むのはアプリ側で、ここでは組み立てだけを行う。
type SourceState struct {
	StatusLine         *StatusLineSnapshot
	StatusLineModified time.Time
	// StatusLineLimits はstatusLineから最後に読めた使用率。最初の応答の前には入らないので、空で上書きせずに残す
	StatusLineLimits   []UsageLimit
	StatusLineLimitsAt time.Time

	TranscriptModelID  string
	TranscriptEffort   string
	TranscriptModified time.Time

	SettingsModel  string
	SettingsEffort string

	// UseAPI は利用者が使用率APIを有効にしているか
	UseAPI     bool
	APILimits  []UsageLimit
	APIAt      time.Time
	APIFailure string
}

// Display は帯とメニューに出すもの。
type Display struct {
	Slot      Slot
	Limits    []UsageLimit
	UsageAsOf time.Time
}

// Compose は状態の枠1つ（"Opus5 1M xhigh · S2 W81 F55"）を組み立てる。
//
//   - モデル名とeffort: statusLineのJSONが会話記録より新しければそれ、無ければ最後の会話記録、それも無ければsettings.json
//   - 使用率: 利用者が有効にしていれば使用率API、そうでなければstatusLineのrate_limits
func Compose(st SourceState, s *Strings, now time.Time) Display {
	// statusLineの方が会話記録より新しければ、その時点のモデルとeffortを使う
	statusLineIsNewer := st.StatusLine != nil && !st.StatusLineModified.IsZero() &&
		(st.TranscriptModified.IsZero() || !st.StatusLineModified.Before(st.TranscriptModified.Add(-time.Minute)))
	model, effort := "", ""
	if statusLineIsNewer {
		model, effort = st.StatusLine.Model, st.StatusLine.Effort
	}
	if model == "" {
		model = ModelDisplayName(st.TranscriptModelID, st.SettingsModel)
	}
	for _, e := range []string{st.TranscriptEffort, st.SettingsEffort} {
		if effort == "" {
			effort = e
		}
	}

	useAPI := st.UseAPI && len(st.APILimits) > 0
	d := Display{Limits: st.StatusLineLimits, UsageAsOf: st.StatusLineLimitsAt}
	if useAPI {
		d.Limits, d.UsageAsOf = st.APILimits, st.APIAt
	}

	stale := IsStale(d.UsageAsOf, now)
	var lines []string
	if len(d.Limits) == 0 {
		lines = append(lines, s.BasicNoUsage)
	}
	for _, l := range d.Limits {
		lines = append(lines, l.Line(s, now))
	}
	if stale {
		lines = append(lines, s.UsageAsOf(ResetText(d.UsageAsOf, now, s)))
	}
	if st.UseAPI && st.APIFailure != "" {
		lines = append(lines, s.UsageAPIFailed(st.APIFailure))
	}
	d.Slot = Slot{
		ID:         StatusID,
		Text:       StatusText(model, effort, d.Limits),
		Background: IdleBackground,
		Foreground: IdleForeground,
		Help:       strings.Join(lines, "\n"),
		Dimmed:     len(d.Limits) == 0 || stale,
	}
	return d
}
