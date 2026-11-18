package core

import (
	"regexp"
	"strings"
)

// ContextSplit は状態の文字列から取り出したコンテキスト長。
// 帯とトレイでは幅を取らない小さな印にし、メニューとツールチップでは文字で書く
type ContextSplit struct {
	// Model はモデル名（"Opus5"）
	Model string
	// Context はコンテキスト長（"1M"）
	Context string
	// Rest は残り（" xhigh · S2 W81"。先頭の空白を含む）
	Rest string
}

// RE2には先読みが無いので、後ろの空白か終端は捕まえてから位置で切る
var contextMarkRe = regexp.MustCompile(`^(\S+) (\d+[MK])(?: |$)`)

// SplitContext: "Opus5 1M xhigh · S2" → ("Opus5", "1M", " xhigh · S2")。コンテキスト長が無ければfalse
func SplitContext(text string) (ContextSplit, bool) {
	m := contextMarkRe.FindStringSubmatchIndex(text)
	if m == nil {
		return ContextSplit{}, false
	}
	return ContextSplit{Model: text[m[2]:m[3]], Context: text[m[4]:m[5]], Rest: text[m[5]:]}, true
}

// SpelledOut はメニューの先頭に出す、省略しない1行。"Opus5 · 1M context · xhigh"
func SpelledOut(statusText string) (string, bool) {
	head, _, _ := strings.Cut(statusText, " · ")
	s, ok := SplitContext(head)
	if !ok {
		return "", false
	}
	parts := []string{s.Model, s.Context + " context"}
	if effort := trimSpace(s.Rest); effort != "" {
		parts = append(parts, effort)
	}
	return strings.Join(parts, " · "), true
}
