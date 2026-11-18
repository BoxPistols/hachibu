package core

// LegendMark は「表示の読み方」の行に付ける印。
type LegendMark int

const (
	MarkNone LegendMark = iota
	MarkContextDot
	MarkLevel
)

// LegendLine は「表示の読み方」に並べる1行。MarkがMarkLevelのときはLevelの色の点を付ける。
type LegendLine struct {
	Mark  LegendMark
	Level UsageLevel
	Text  string
}

// Legend は帯にいま出ている記号と色だけを説明する。
func Legend(statusText string, limits []UsageLimit, t Thresholds, s *Strings) []LegendLine {
	var lines []LegendLine
	if split, ok := SplitContext(statusText); ok {
		lines = append(lines, LegendLine{Mark: MarkContextDot, Text: s.LegendContext(split.Context)})
	}
	for _, l := range limits {
		if l.Kind == KindModel {
			lines = append(lines, LegendLine{Text: s.LegendModelLimit(l.Letter(), l.Model)})
		} else {
			lines = append(lines, LegendLine{Text: s.LegendLimit(l.Letter(), l.Name(s))})
		}
	}
	lines = append(lines,
		LegendLine{Mark: MarkLevel, Level: LevelWarning, Text: s.LegendWarning(t.Warning)},
		LegendLine{Mark: MarkLevel, Level: LevelCritical, Text: s.LegendCritical(t.Critical)})
	return lines
}
