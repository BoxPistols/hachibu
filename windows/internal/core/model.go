// Package core は、帯に何をどう出すかの判断を持つ。画面もOSも持たないので、Linuxでもテストできる。
// macOS版のSources/HachibuCoreを移したもので、判断が変わったときは両方を直す。
package core

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// RGBA は0〜255の色。
type RGBA struct{ R, G, B, A float64 }

var (
	IdleBackground = RGBA{58, 58, 60, 255}
	IdleForeground = RGBA{190, 190, 195, 255}
	White          = RGBA{255, 255, 255, 255}
	Clear          = RGBA{}
)

// StatusID は、基本表示でただ1つ出す状態の枠の番号。
const StatusID = "status"

// Slot は帯に並べる枠1つぶん。
type Slot struct {
	ID         string
	Text       string
	Background RGBA
	Foreground RGBA
	// Help はツールチップに出す説明（リセット時刻など、帯に載っていない情報）
	Help   string
	Dimmed bool
}

// LimitKind は使用率の枠の種類。表示名ではなく種類で見分ける（表示名は言語で変わる）。
type LimitKind int

const (
	KindFiveHour LimitKind = iota
	KindWeekly
	// KindModel はモデル別の週枠。UsageLimit.Modelにサーバが返すモデルの表示名が入る
	KindModel
)

// UsageLimit は使用率の枠1つ（5時間枠・週枠・モデル別枠）。
type UsageLimit struct {
	Kind    LimitKind
	Model   string
	Percent int
	// ResetsAt はリセットの時刻。分からなければゼロ値
	ResetsAt time.Time
}

func (l UsageLimit) Name(s *Strings) string {
	switch l.Kind {
	case KindFiveHour:
		return s.LimitFiveHour
	case KindWeekly:
		return s.LimitWeekly
	default:
		return l.Model
	}
}

// Letter は帯での略号: S=5時間枠、W=週枠、モデル別は名前の頭文字。
func (l UsageLimit) Letter() string {
	switch l.Kind {
	case KindFiveHour:
		return "S"
	case KindWeekly:
		return "W"
	default:
		r, size := utf8.DecodeRuneInString(l.Model)
		if r == utf8.RuneError && size == 0 {
			return ""
		}
		return string(r)
	}
}

// Line はメニューとツールチップに出す1行（"週枠75%（1/5(土) 7:30にリセット）"）。
func (l UsageLimit) Line(s *Strings, now time.Time) string {
	resets := ""
	if !l.ResetsAt.IsZero() {
		resets = ResetText(l.ResetsAt, now, s)
	}
	return s.LimitLine(l.Name(s), l.Percent, resets)
}

// ResetText は、当日なら時刻だけ、それ以外は曜日と日付を添えた形にする。時刻はnowのタイムゾーンで書く。
func ResetText(t, now time.Time, s *Strings) string {
	t = t.In(now.Location())
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return s.ResetSameDay(t)
	}
	return s.ResetOtherDay(t)
}

var fractionalSeconds = regexp.MustCompile(`\.\d+`)

// ParseISO はサーバが返す "2026-09-19T23:59:59.865631+00:00" 形式を読む。小数秒の桁数がまちまちなので落としてから読む。
func ParseISO(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, fractionalSeconds.ReplaceAllString(s, ""))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func trimSpace(s string) string {
	return strings.Trim(s, " \t")
}
