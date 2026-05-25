// Package filter 提供 Phase-1 零 LLM 轻量过滤（寒暄、空输入等）。
package filter

import (
	"strings"
	"unicode"
)

var chitchatExact = map[string]struct{}{
	"你好": {}, "您好": {}, "hi": {}, "hello": {}, "hey": {},
	"谢谢": {}, "感谢": {}, "多谢": {}, "thanks": {}, "thank you": {},
	"再见": {}, "拜拜": {}, "bye": {}, "goodbye": {},
	"在吗": {}, "在不在": {}, "ok": {}, "好的": {}, "嗯": {}, "哦": {},
}

// ShouldSkipStore 是否应对 memory_store 做 no-op（寒暄 / Soul 边界 / 无执行信号）。
func ShouldSkipStore(content string) (skip bool, reason string) {
	if skip, reason := classifyStoreScope(content); skip {
		return skip, reason
	}
	s := trimForMatch(content)
	if isMostlyPunctuation(s) {
		return true, "punctuation_only"
	}
	return false, ""
}

// ShouldSkipRetrieve 是否应对 memory_retrieve 做 no-op（空 hints）。
func ShouldSkipRetrieve(context string) (skip bool, reason string) {
	// 从 context 中取最后一行或整体做寒暄判断（Host 可能拼多段）。
	lines := strings.Split(strings.TrimSpace(context), "\n")
	candidate := strings.TrimSpace(context)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			candidate = line
			break
		}
	}
	s := trimForMatch(candidate)
	if s == "" {
		return true, "empty_context"
	}
	if _, ok := chitchatExact[s]; ok {
		return true, "chitchat"
	}
	if len([]rune(s)) < 2 {
		return true, "too_short"
	}
	return false, ""
}

func trimForMatch(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 500 {
		s = s[len(s)-500:]
	}
	return strings.ToLower(s)
}

func isMostlyPunctuation(s string) bool {
	var letters int
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			letters++
		}
	}
	return letters == 0
}
