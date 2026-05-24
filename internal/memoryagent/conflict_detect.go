package memoryagent

import (
	"sort"
	"strings"

	"AgentTestMemoryMCP/internal/agent"
	"AgentTestMemoryMCP/internal/entity"
	"AgentTestMemoryMCP/internal/facts"
)

// ConflictCandidate 语义冲突候选（与高 weight 旧 fact 结论对立）。
type ConflictCandidate struct {
	Old facts.Fact
}

// DetectConflictCandidates 规则层：筛出需 L2 裁决的旧 fact（0 次 LLM）。
func DetectConflictCandidates(newFact facts.Fact, existing []facts.Fact, sameCorrelation string, cfg L2Config) []ConflictCandidate {
	if len(existing) == 0 {
		return nil
	}
	if cfg.MinOldWeight <= 0 {
		cfg = DefaultL2Config()
	}
	newOutcome := agent.NormalizeOutcome(newFact.Outcome)
	if newOutcome == "unknown" && !newFact.IsPitfall {
		return nil
	}

	var cands []ConflictCandidate
	for _, old := range existing {
		if old.ID == newFact.ID || old.Superseded {
			continue
		}
		if old.Weight < cfg.MinOldWeight {
			continue
		}
		if sameCorrelation != "" && old.CorrelationID == sameCorrelation {
			continue
		}
		if !topicOverlap(newFact, old, cfg.MinTagJaccard) {
			continue
		}
		if !outcomesConflict(newFact, old) {
			continue
		}
		// 同结论且高度相似 → 由 supersede/对齐处理，不走 L2
		if sameOutcomeClass(newFact, old) && entitySimilarityScore(newFact, old) >= cfg.SkipIfSimilarGTE {
			continue
		}
		cands = append(cands, ConflictCandidate{Old: old})
	}

	sort.Slice(cands, func(i, j int) bool {
		return cands[i].Old.Weight > cands[j].Old.Weight
	})
	max := cfg.MaxCandidates
	if max <= 0 {
		max = 3
	}
	if len(cands) > max {
		cands = cands[:max]
	}
	return cands
}

func topicOverlap(a, b facts.Fact, minJaccard float64) bool {
	if len(intersectStrings(a.Tools, b.Tools)) > 0 {
		return true
	}
	if tagsJaccard(a.Tags, b.Tags) >= minJaccard {
		return true
	}
	if artifactsOverlap(a.Artifacts, b.Artifacts) {
		return true
	}
	return false
}

func outcomesConflict(newF, old facts.Fact) bool {
	newS := outcomeClass(newF)
	oldS := outcomeClass(old)
	if newS == "unknown" || oldS == "unknown" {
		return false
	}
	return newS != oldS
}

func sameOutcomeClass(a, b facts.Fact) bool {
	return outcomeClass(a) == outcomeClass(b)
}

func outcomeClass(f facts.Fact) string {
	if f.IsPitfall {
		return "fail"
	}
	switch agent.NormalizeOutcome(f.Outcome) {
	case "success":
		return "success"
	case "fail":
		return "fail"
	default:
		return "unknown"
	}
}

// entitySimilarityScore 用 embedding 比较摘要相似度（与 EnrichAlignment 同源）。
func entitySimilarityScore(a, b facts.Fact) float64 {
	cfg := entity.DefaultAlignConfig()
	sup, fuzzy := entity.CompareFacts(a, []facts.Fact{b}, "", cfg)
	if len(sup) > 0 {
		return cfg.HardMerge
	}
	if len(fuzzy) > 0 {
		return fuzzy[0].Score
	}
	return 0
}

func intersectStrings(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	set := map[string]bool{}
	for _, x := range a {
		x = strings.TrimSpace(x)
		if x != "" {
			set[x] = true
		}
	}
	var out []string
	seen := map[string]bool{}
	for _, x := range b {
		x = strings.TrimSpace(x)
		if x == "" || !set[x] || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	return out
}

func tagsJaccard(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	setA := map[string]bool{}
	for _, x := range a {
		x = strings.TrimSpace(x)
		if x != "" {
			setA[x] = true
		}
	}
	inter := 0
	union := len(setA)
	for _, x := range b {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if setA[x] {
			inter++
		} else {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func artifactsOverlap(a, b []string) bool {
	for _, pa := range a {
		pa = strings.TrimSpace(strings.ToLower(pa))
		if pa == "" {
			continue
		}
		for _, pb := range b {
			pb = strings.TrimSpace(strings.ToLower(pb))
			if pb == "" {
				continue
			}
			if pa == pb || strings.HasPrefix(pa, pb) || strings.HasPrefix(pb, pa) {
				return true
			}
			if strings.Contains(pa, "/") && strings.Contains(pb, "/") {
				if pathBase(pa) == pathBase(pb) {
					return true
				}
			}
		}
	}
	return false
}

func pathBase(p string) string {
	p = strings.Trim(p, "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
