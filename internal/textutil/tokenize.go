package textutil

import (
	"strings"
	"unicode"
)

// Tokens 简易分词：字母数字块 + 中文连续段。
func Tokens(s string) map[string]struct{} {
	s = strings.ToLower(s)
	out := map[string]struct{}{}
	var cur strings.Builder
	flush := func() {
		if cur.Len() >= 2 {
			out[cur.String()] = struct{}{}
		}
		cur.Reset()
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
			continue
		}
		if unicode.Is(unicode.Han, r) {
			flush()
			out[string(r)] = struct{}{}
			continue
		}
		flush()
	}
	flush()
	return out
}

// OverlapScore 查询与文档 token 交集占比（0~1）。
func OverlapScore(query, doc string) float64 {
	q := Tokens(query)
	if len(q) == 0 {
		return 0
	}
	d := Tokens(doc)
	if len(d) == 0 {
		return 0
	}
	hit := 0
	for t := range q {
		if _, ok := d[t]; ok {
			hit++
		}
	}
	return float64(hit) / float64(len(q))
}

func ContainsAny(s string, subs []string) bool {
	s = strings.ToLower(s)
	for _, sub := range subs {
		sub = strings.TrimSpace(sub)
		if sub != "" && strings.Contains(s, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}
