package core

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// StaleAfter より古い使用率は、帯で薄く表示し、いつ時点の値かを添える。
const StaleAfter = 30 * time.Minute

// IsStale は、時刻が分からなければfalse。
func IsStale(asOf, now time.Time) bool {
	if asOf.IsZero() {
		return false
	}
	return now.Sub(asOf) > StaleAfter
}

// StatusText はTouch Barのステータスと同じ形: "Opus5 1M xhigh · S9 W75 F55"。
// modelは表示用に縮めたモデル名（CompactDisplayNameの結果）。空の値は無いものとして扱う。
func StatusText(model, effort string, limits []UsageLimit) string {
	var head []string
	for _, v := range []string{model, effort} {
		if v != "" {
			head = append(head, v)
		}
	}
	var tail []string
	for _, l := range limits {
		tail = append(tail, l.Letter()+strconv.Itoa(l.Percent))
	}
	var parts []string
	for _, p := range []string{strings.Join(head, " "), strings.Join(tail, " ")} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return StatusPlaceholder
	}
	return strings.Join(parts, " · ")
}

// UsageOnlyText は使用率だけの文字列（"S11 W75 F55"）。使用率が無ければ空。
func UsageOnlyText(limits []UsageLimit) string {
	if len(limits) == 0 {
		return ""
	}
	return StatusText("", "", limits)
}

var (
	contextInName = regexp.MustCompile(`\s*\(\s*\d+(\.\d+)?\s*[KkMm]\s+context\s*\)`)
	nameVersion   = regexp.MustCompile(`([A-Za-z]) (\d)`)
)

// CompactDisplayName はstatusLineのmodel.display_nameを帯の幅に合わせて縮める。"Opus 5 (1M context)" → "Opus5 1M"
func CompactDisplayName(name string) string {
	s := trimSpace(name)
	suffix := ""
	if loc := contextInName.FindStringIndex(s); loc != nil {
		var size strings.Builder
		for _, r := range s[loc[0]:loc[1]] {
			if (r >= '0' && r <= '9') || r == '.' || strings.ContainsRune("KkMm", r) {
				size.WriteRune(r)
			}
		}
		suffix = " " + strings.ToUpper(size.String())
		s = s[:loc[0]] + s[loc[1]:]
	}
	// 名前と版番号の間の空白を詰める（"Opus 5" → "Opus5"）
	s = nameVersion.ReplaceAllString(s, "$1$2")
	return s + suffix
}

// StatusLineSnapshot はClaude CodeのstatusLineが標準入力に渡すJSONから読んだ値。
type StatusLineSnapshot struct {
	Model  string
	Effort string
	Limits []UsageLimit
}

// ParseStatusLine はstatusLineのJSONを読む。rate_limitsはPro/Maxで、そのセッションの最初の応答の後にだけ入る。
func ParseStatusLine(obj map[string]any) StatusLineSnapshot {
	var snap StatusLineSnapshot
	if name, ok := str(dict(obj["model"])["display_name"]); ok {
		snap.Model = CompactDisplayName(name)
	}
	// 表示名に文脈の長さが入らないモデルがある（"Fable 5.1"）。窓の大きさが100万以上なら1Mを添える
	if snap.Model != "" {
		if _, has := SplitContext(snap.Model); !has {
			if size, ok := num(dict(obj["context_window"])["context_window_size"]); ok && int(size) >= 1_000_000 {
				snap.Model += " " + strconv.Itoa(int(size)/1_000_000) + "M"
			}
		}
	}
	snap.Effort, _ = str(dict(obj["effort"])["level"])
	rates := dict(obj["rate_limits"])
	for _, k := range []struct {
		key  string
		kind LimitKind
	}{{"five_hour", KindFiveHour}, {"seven_day", KindWeekly}} {
		item := dict(rates[k.key])
		pct, ok := num(item["used_percentage"])
		if !ok {
			continue
		}
		limit := UsageLimit{Kind: k.kind, Percent: int(math.Round(pct))}
		if secs, ok := num(item["resets_at"]); ok {
			whole, frac := math.Modf(secs)
			limit.ResetsAt = time.Unix(int64(whole), int64(frac*1e9))
		}
		snap.Limits = append(snap.Limits, limit)
	}
	return snap
}

// JSONの値を型ごとに取り出す小さな道具。型が違えば無いものとして扱う

func dict(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func str(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func num(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
}
