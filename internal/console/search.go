package console

import (
	"strings"

	"AgentTestMemoryMCP/internal/facts"
)

// SearchHit 搜索命中项。
type SearchHit struct {
	NodeID string         `json:"node_id"`
	Group  string         `json:"group"`
	Label  string         `json:"label"`
	Score  float64        `json:"score"`
	Data   map[string]any `json:"data,omitempty"`
}

// SearchFacts 在事实库中搜索（子串 + 标签/工具精确匹配）。
func SearchFacts(all []facts.Fact, query string, limit int) []SearchHit {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" || limit <= 0 {
		return nil
	}
	var hits []SearchHit
	for _, f := range all {
		score, ok := scoreFact(f, query)
		if !ok {
			continue
		}
		hits = append(hits, SearchHit{
			NodeID: nodeIDFact(f.ID),
			Group:  "fact",
			Label:  truncateLabel(f.Text, 48),
			Score:  score,
			Data:   factDataMap(f),
		})
	}
	sortHits(hits)
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func scoreFact(f facts.Fact, q string) (float64, bool) {
	var score float64
	if strings.Contains(strings.ToLower(f.ID), q) {
		score += 3
	}
	if strings.Contains(strings.ToLower(f.Text), q) {
		score += 2
	}
	if strings.Contains(strings.ToLower(f.Source), q) {
		score += 1.5
	}
	if strings.Contains(strings.ToLower(f.CorrelationID), q) {
		score += 1.5
	}
	if strings.Contains(strings.ToLower(f.EpisodeID), q) {
		score += 1
	}
	for _, t := range f.Tags {
		if strings.EqualFold(strings.TrimSpace(t), q) || strings.Contains(strings.ToLower(t), q) {
			score += 1.2
		}
	}
	for _, t := range f.Tools {
		if strings.Contains(strings.ToLower(t), q) {
			score += 1.2
		}
	}
	for _, a := range f.Artifacts {
		if strings.Contains(strings.ToLower(a), q) {
			score += 0.8
		}
	}
	return score, score > 0
}

func sortHits(h []SearchHit) {
	for i := 0; i < len(h); i++ {
		for j := i + 1; j < len(h); j++ {
			if h[j].Score > h[i].Score {
				h[i], h[j] = h[j], h[i]
			}
		}
	}
}
