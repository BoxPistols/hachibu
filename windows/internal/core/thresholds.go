package core

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// UsageLevel は使用率の段階。帯では数字を札で囲み、黄＝注意、赤＝危険として示す。
type UsageLevel int

const (
	LevelNormal UsageLevel = iota
	LevelWarning
	LevelCritical
)

// Thresholds は黄と赤に切り替える使用率（%）。利用者が選べる。赤は黄より大きい値にしか置けない。
type Thresholds struct {
	Warning  int
	Critical int
}

var (
	DefaultThresholds = Thresholds{Warning: 70, Critical: 90}
	WarningChoices    = []int{50, 60, 70, 80}
	CriticalChoices   = []int{80, 85, 90, 95}
)

// NewThresholds は、赤が黄以下になる組み合わせを作らない。
func NewThresholds(warning, critical int) (Thresholds, bool) {
	if warning < 1 || warning > 100 || critical < 1 || critical > 100 || warning >= critical {
		return Thresholds{}, false
	}
	return Thresholds{Warning: warning, Critical: critical}, true
}

func (t Thresholds) Level(percent int) UsageLevel {
	if percent >= t.Critical {
		return LevelCritical
	}
	if percent >= t.Warning {
		return LevelWarning
	}
	return LevelNormal
}

func (t Thresholds) Worst(limits []UsageLimit) UsageLevel {
	worst := LevelNormal
	for _, l := range limits {
		worst = max(worst, t.Level(l.Percent))
	}
	return worst
}

// TextSegment は帯の文字の一部。段階が付いていれば札で囲む。
type TextSegment struct {
	Text  string
	Level UsageLevel
}

// Segments は "S12" "W76" "F55" のような使用率の語だけに段階を付ける。それ以外（モデル名、ctx91%など）は普通の文字のまま。
func Segments(text string, t Thresholds) []TextSegment {
	var out []TextSegment
	plain := ""
	for i, token := range strings.Split(text, " ") {
		sep := " "
		if i == 0 {
			sep = ""
		}
		if percent, ok := usagePercent(token); ok && t.Level(percent) != LevelNormal {
			plain += sep
			if plain != "" {
				out = append(out, TextSegment{Text: plain, Level: LevelNormal})
			}
			plain = ""
			out = append(out, TextSegment{Text: token, Level: t.Level(percent)})
		} else {
			plain += sep + token
		}
	}
	if plain != "" {
		out = append(out, TextSegment{Text: plain, Level: LevelNormal})
	}
	return out
}

// usagePercent は大文字1字＋数字1〜3桁（S12、W100）を読む。
func usagePercent(token string) (int, bool) {
	n := utf8.RuneCountInString(token)
	if n < 2 || n > 4 || token[0] < 'A' || token[0] > 'Z' {
		return 0, false
	}
	digits := token[1:]
	for _, r := range digits {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	v, err := strconv.Atoi(digits)
	return v, err == nil
}
