package memoryagent

import (
	"strings"
	"unicode"

	"AgentTestMemoryMCP/internal/textutil"
)

const defaultFuzzyThreshold = 0.85

// EvidenceAnchored 判断 evidence 是否在 episode 上通过 Fuzzy 锚定（L1）。
func EvidenceAnchored(evidence, episode string, threshold float64) bool {
	evidence = strings.TrimSpace(evidence)
	episode = strings.TrimSpace(episode)
	if evidence == "" || episode == "" {
		return false
	}
	if threshold <= 0 {
		threshold = defaultFuzzyThreshold
	}
	if strings.Contains(normalizeForContains(episode), normalizeForContains(evidence)) {
		return true
	}
	if tokenJaccard(evidence, episode) >= threshold {
		return true
	}
	return bigramOverlap(evidence, episode) >= threshold
}

func normalizeForContains(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Han, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func tokenJaccard(a, b string) float64 {
	ta := textutil.Tokens(a)
	tb := textutil.Tokens(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	inter := 0
	for t := range ta {
		if _, ok := tb[t]; ok {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func bigramOverlap(a, b string) float64 {
	na := normalizeForContains(a)
	nb := normalizeForContains(b)
	if len(na) < 2 || len(nb) < 2 {
		return textutil.OverlapScore(a, b)
	}
	setA := map[string]bool{}
	for i := 0; i < len(na)-1; i++ {
		setA[na[i:i+2]] = true
	}
	hit := 0
	for i := 0; i < len(nb)-1; i++ {
		if setA[nb[i:i+2]] {
			hit++
		}
	}
	denom := len(na) - 1
	if len(nb)-1 > denom {
		denom = len(nb) - 1
	}
	if denom == 0 {
		return 0
	}
	return float64(hit) / float64(denom)
}
