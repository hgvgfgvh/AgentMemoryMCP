package memoryagent

import (
	"AgentTestMemoryMCP/internal/entity"
	"AgentTestMemoryMCP/internal/facts"
)

// EnrichAlignment 对单条新 fact 做实体硬合并/模糊带检测（2d）。
func EnrichAlignment(newFact facts.Fact, existing []facts.Fact, correlationID string, llmSupersede []string) (facts.Fact, []string, []entity.FuzzyPair) {
	cfg := entity.DefaultAlignConfig()
	hard, fuzzy := entity.CompareFacts(newFact, existing, correlationID, cfg)
	seen := map[string]bool{}
	var all []string
	for _, id := range llmSupersede {
		if id != "" && !seen[id] {
			seen[id] = true
			all = append(all, id)
		}
	}
	for _, id := range hard {
		if !seen[id] {
			seen[id] = true
			all = append(all, id)
		}
	}
	return newFact, all, fuzzy
}
