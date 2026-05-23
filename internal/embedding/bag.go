package embedding

import (
	"math"
	"strings"

	"AgentTestMemoryMCP/internal/textutil"
)

// Bag 简易词袋向量（本地 cosine，无外部 embedding API）。
type Bag map[string]float64

// BagFromText 由文本构造归一化词袋。
func BagFromText(s string) Bag {
	toks := textutil.Tokens(s)
	if len(toks) == 0 {
		return nil
	}
	b := make(Bag, len(toks))
	for t := range toks {
		b[t] += 1
	}
	var sum float64
	for _, v := range b {
		sum += v * v
	}
	if sum == 0 {
		return b
	}
	norm := math.Sqrt(sum)
	for t := range b {
		b[t] /= norm
	}
	return b
}

// Cosine 余弦相似度 [0,1]。
func Cosine(a, b Bag) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	var dot float64
	for t, va := range a {
		if vb, ok := b[t]; ok {
			dot += va * vb
		}
	}
	if dot < 0 {
		return 0
	}
	if dot > 1 {
		return 1
	}
	return dot
}

// NormalizeEntityKey 硬规则规范化实体文本。
func NormalizeEntityKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	// 去常见版本片段
	for _, cut := range []string{"v1", "v2", "mcp"} {
		s = strings.ReplaceAll(s, cut, "")
	}
	return s
}
