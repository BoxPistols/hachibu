package core

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

// ParseUsageAPI は使用率APIの応答を読む。呼ぶのは利用者が有効にしたときだけ。
// limits[] を優先して読む。kindはsession・weekly_all・weekly_scopedの3種で、percentは0〜100の整数
func ParseUsageAPI(obj map[string]any) []UsageLimit {
	var out []UsageLimit
	list, _ := obj["limits"].([]any)
	for _, raw := range list {
		lim := dict(raw)
		percent, ok := num(lim["percent"])
		if !ok {
			continue
		}
		resets, _ := ParseISO(stringOr(lim["resets_at"]))
		kind, _ := str(lim["kind"])
		switch kind {
		case "session":
			out = append(out, UsageLimit{Kind: KindFiveHour, Percent: int(percent), ResetsAt: resets})
		case "weekly_all":
			out = append(out, UsageLimit{Kind: KindWeekly, Percent: int(percent), ResetsAt: resets})
		case "weekly_scoped":
			name := trimSpace(stringOr(dict(dict(lim["scope"])["model"])["display_name"]))
			if name != "" {
				out = append(out, UsageLimit{Kind: KindModel, Model: name, Percent: int(percent), ResetsAt: resets})
			}
		}
	}
	if len(out) > 0 {
		return out
	}

	// limits[] が無い応答向け。utilizationは百分率で来る（2026-09-18の実測で11.0のような値）
	for _, k := range []struct {
		key  string
		kind LimitKind
	}{{"five_hour", KindFiveHour}, {"seven_day", KindWeekly}} {
		item := dict(obj[k.key])
		value, ok := num(item["utilization"])
		if !ok {
			continue
		}
		resets, _ := ParseISO(stringOr(item["resets_at"]))
		out = append(out, UsageLimit{Kind: k.kind, Percent: int(math.Round(value)), ResetsAt: resets})
	}
	return out
}

func stringOr(v any) string {
	s, _ := v.(string)
	return s
}

var (
	contextAlias = regexp.MustCompile(`\[(\d+)m\]$`)
	dateSuffix   = regexp.MustCompile(`-\d{8}$`)
	lastModelRe  = regexp.MustCompile(`"model"\s*:\s*"(claude-[A-Za-z0-9.\-\[\]]+)"`)
	lastEffortRe = regexp.MustCompile(`"effort"\s*:\s*"(low|medium|high|xhigh|max)"`)
)

// CompactModelID はモデルのID（"claude-opus-5"、"claude-opus-5-20260101"、"claude-opus-5[1m]"）を帯の形にする: "Opus5"、"Opus5 1M"
func CompactModelID(id string) (string, bool) {
	s := trimSpace(id)
	if !strings.HasPrefix(s, "claude-") {
		return "", false
	}
	s = strings.TrimPrefix(s, "claude-")
	suffix := ""
	if loc := contextAlias.FindStringIndex(s); loc != nil {
		suffix = " " + strings.ToUpper(s[loc[0]+1:loc[1]-1])
		s = s[:loc[0]]
	}
	// 末尾の日付（-20260101）は落とす
	s = dateSuffix.ReplaceAllString(s, "")
	var parts []string
	for _, p := range strings.Split(s, "-") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	family := parts[0]
	// "opus-5" → "Opus5"、"haiku-4-5" → "Haiku4.5"
	version := strings.Join(parts[1:], ".")
	return upperFirst(family) + version + suffix, true
}

// ContextSuffixOfAlias は設定のモデル指定（"opus[1m]"、"sonnet"）から、文脈の長さの指定だけを取り出す: "1M"
func ContextSuffixOfAlias(alias string) (string, bool) {
	loc := contextAlias.FindStringIndex(alias)
	if loc == nil {
		return "", false
	}
	return strings.ToUpper(alias[loc[0]+1 : loc[1]-1]), true
}

// LastModelID は会話記録（JSONL）の末尾から、最後に使われたモデルのIDを探す。
func LastModelID(transcriptTail string) string {
	return lastSubmatch(lastModelRe, transcriptTail)
}

// LastEffort は会話記録（JSONL）の末尾から、最後の応答のeffortを探す。
// セッション中に/effortで変えた値はsettings.jsonに残らないので、こちらを設定より優先する
func LastEffort(transcriptTail string) string {
	return lastSubmatch(lastEffortRe, transcriptTail)
}

func lastSubmatch(re *regexp.Regexp, text string) string {
	all := re.FindAllStringSubmatch(text, -1)
	if len(all) == 0 {
		return ""
	}
	return all[len(all)-1][1]
}

// ModelDisplayName は会話記録のモデルIDと設定のモデル指定を合わせて、帯のモデル名にする。
// 会話記録のIDに文脈の長さが無ければ、同じ系統のモデルを指す設定の指定（"opus[1m]"の"1M"）を添える
func ModelDisplayName(transcriptModelID, settingsModel string) string {
	if transcriptModelID != "" {
		if name, ok := CompactModelID(transcriptModelID); ok {
			// 設定の指定が別の系統のモデル（会話はsonnet、設定は"opus[1m]"など）なら、その文脈の長さは付けない
			if !strings.Contains(name, " ") && settingsModel != "" {
				if ctx, ok := ContextSuffixOfAlias(settingsModel); ok {
					if family := firstLetterRun(strings.ToLower(settingsModel)); family != "" &&
						strings.Contains(strings.ToLower(transcriptModelID), family) {
						name += " " + ctx
					}
				}
			}
			return name
		}
	}
	if settingsModel == "" {
		return ""
	}
	return CompactDisplayName(aliasToDisplay(settingsModel))
}

// aliasToDisplay: "opus[1m]" → "Opus (1M context)"（statusLineのdisplay_nameと同じ形に寄せて、同じ関数で縮める）
func aliasToDisplay(alias string) string {
	s := alias
	ctx := ""
	if c, ok := ContextSuffixOfAlias(alias); ok {
		ctx = " (" + c + " context)"
		s = s[:contextAlias.FindStringIndex(s)[0]]
	}
	return upperFirst(s) + ctx
}

func upperFirst(s string) string {
	for i, r := range s {
		return string(unicode.ToUpper(r)) + s[i+len(string(r)):]
	}
	return s
}

func firstLetterRun(s string) string {
	f := strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) })
	if len(f) == 0 {
		return ""
	}
	return f[0]
}
