package index

import (
	"math"

	"AgentTestMemoryMCP/internal/textutil"
)

// BM25 轻量 Okapi BM25（语料为单文档时退化为词重合）。
func BM25(query, doc string, avgDL float64, k1, b float64) float64 {
	if k1 <= 0 {
		k1 = 1.2
	}
	if b <= 0 {
		b = 0.75
	}
	qTokens := tokenSlice(query)
	dTokens := tokenSlice(doc)
	if len(qTokens) == 0 || len(dTokens) == 0 {
		return textutil.OverlapScore(query, doc)
	}
	df := map[string]int{}
	for _, t := range dTokens {
		df[t]++
	}
	dl := float64(len(dTokens))
	if avgDL <= 0 {
		avgDL = dl
		if avgDL < 1 {
			avgDL = 1
		}
	}
	score := 0.0
	seen := map[string]bool{}
	for _, term := range qTokens {
		if seen[term] {
			continue
		}
		seen[term] = true
		tf := float64(df[term])
		if tf == 0 {
			continue
		}
		idf := math.Log(1 + 1.0/(1+tf)) // 单文档 IDF 近似
		denom := tf + k1*(1-b+b*dl/avgDL)
		score += idf * (tf * (k1 + 1) / denom)
	}
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func tokenSlice(s string) []string {
	m := textutil.Tokens(s)
	out := make([]string, 0, len(m))
	for t := range m {
		out = append(out, t)
	}
	return out
}
