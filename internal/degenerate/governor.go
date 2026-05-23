package degenerate

import (
	"os"
	"strconv"
	"time"

	"AgentTestMemoryMCP/internal/facts"
	"AgentTestMemoryMCP/internal/graph"
)

const defaultSupersedeFactor = 0.2

// Config 退化参数。
type Config struct {
	SupersedeFactor  float64
	StaleAfter       time.Duration
	StaleDecayFactor float64
	RetrieveBoost    float64
}

func DefaultConfig() Config {
	cfg := Config{
		SupersedeFactor:  0.2,
		StaleAfter:       30 * 24 * time.Hour,
		StaleDecayFactor: 0.85,
		RetrieveBoost:    0.02,
	}
	if v := os.Getenv("MEMORY_MCP_SUPERSEDE_FACTOR"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			cfg.SupersedeFactor = f
		}
	}
	if v := os.Getenv("MEMORY_MCP_STALE_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.StaleAfter = time.Duration(n) * 24 * time.Hour
		}
	}
	return cfg
}

// ApplySupersedes 将旧 fact 降权并标记 superseded；返回需写入的 supersedes 边。
func ApplySupersedes(all []facts.Fact, newFactID string, oldIDs []string, cfg Config) ([]facts.Fact, []graph.Edge) {
	if cfg.SupersedeFactor <= 0 {
		cfg.SupersedeFactor = defaultSupersedeFactor
	}
	idSet := map[string]bool{}
	for _, id := range oldIDs {
		idSet[id] = true
	}
	var edges []graph.Edge
	for i := range all {
		if !idSet[all[i].ID] {
			continue
		}
		all[i].Weight *= cfg.SupersedeFactor
		if all[i].Weight < 0.05 {
			all[i].Weight = 0.05
		}
		all[i].Superseded = true
		edges = append(edges, graph.Edge{
			From:   graph.NodeFact(newFactID),
			To:     graph.NodeFact(all[i].ID),
			Type:   graph.EdgeSupersedes,
			Weight: 1.0,
		})
	}
	return all, edges
}

// ApplyStaleDecay 久未访问 fact 权重衰减。
func ApplyStaleDecay(all []facts.Fact, now time.Time, cfg Config) []facts.Fact {
	if cfg.StaleAfter <= 0 {
		cfg.StaleAfter = 30 * 24 * time.Hour
	}
	if cfg.StaleDecayFactor <= 0 {
		cfg.StaleDecayFactor = 0.85
	}
	for i := range all {
		if all[i].Superseded || all[i].Weight < 0.1 {
			continue
		}
		last := all[i].LastActive
		if last.IsZero() {
			last = all[i].CreatedAt
		}
		if now.Sub(last) > cfg.StaleAfter {
			all[i].Weight *= cfg.StaleDecayFactor
			if all[i].Weight < 0.05 {
				all[i].Weight = 0.05
			}
		}
	}
	return all
}

// TouchRetrieve 命中 retrieve 的 fact：更新 last_active、access_count、略增 weight。
func TouchRetrieve(all []facts.Fact, hitIDs []string, now time.Time, cfg Config) []facts.Fact {
	if len(hitIDs) == 0 {
		return all
	}
	if cfg.RetrieveBoost <= 0 {
		cfg.RetrieveBoost = 0.02
	}
	hit := map[string]bool{}
	for _, id := range hitIDs {
		hit[id] = true
	}
	for i := range all {
		if !hit[all[i].ID] {
			continue
		}
		all[i].LastActive = now
		all[i].AccessCount++
		all[i].Weight += cfg.RetrieveBoost
		if all[i].Weight > 1.5 {
			all[i].Weight = 1.5
		}
	}
	return all
}

// MergeExtraEdges 将 supersedes 边并入边列表（去重由 rebuild 全量负责时可只追加）。
func MergeExtraEdges(base, extra []graph.Edge) []graph.Edge {
	if len(extra) == 0 {
		return base
	}
	return append(base, extra...)
}
