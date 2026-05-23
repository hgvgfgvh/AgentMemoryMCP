package retrieve

import (
	"context"
	"strings"

	"AgentTestMemoryMCP/internal/agent"
	"AgentTestMemoryMCP/internal/facts"
	"AgentTestMemoryMCP/internal/graph"
	"AgentTestMemoryMCP/internal/index"
)

// PipelineConfig retrieve 图路径参数。
type PipelineConfig struct {
	TopK           int
	MinAnchorScore float64
	RouteThreshold float64
	PruneAlpha     float64
	PruneBeta      float64
	PruneGamma     float64
	PruneDelta     float64
}

func DefaultPipelineConfig(routeThreshold float64, topK int, minScore float64) PipelineConfig {
	if topK <= 0 {
		topK = 5
	}
	if minScore <= 0 {
		minScore = 0.35
	}
	if routeThreshold <= 0 {
		routeThreshold = 0.75
	}
	return PipelineConfig{
		TopK:           topK,
		MinAnchorScore: minScore,
		RouteThreshold: routeThreshold,
		PruneAlpha:     0.4,
		PruneBeta:      0.4,
		PruneGamma:     0.2,
		PruneDelta:     0.5,
	}
}

// SearchWithGraph 种子锚定 → BFS → BM25 复合剪枝。
func SearchWithGraph(ctx context.Context, all []facts.Fact, mg *graph.MemoryGraph, contextStr, queryHint string, cfg PipelineConfig) []ScoredFact {
	if ctx.Err() != nil {
		return nil
	}
	if len(all) == 0 {
		return nil
	}

	seeds := seedAnchors(all, contextStr, queryHint, 3, cfg.MinAnchorScore)
	if len(seeds) == 0 {
		return nil
	}
	if ctx.Err() != nil {
		return nil
	}

	activated := graph.WeightedBFS(mg, seeds, graph.DefaultBFSConfig())
	if ctx.Err() != nil {
		return nil
	}

	scored := pruneComposite(all, activated, contextStr, cfg)
	if cfg.TopK > 0 && len(scored) > cfg.TopK {
		scored = scored[:cfg.TopK]
	}
	return scored
}

func seedAnchors(all []facts.Fact, contextStr, queryHint string, topN int, minScore float64) map[string]float64 {
	type pair struct {
		id    string
		score float64
	}
	var ranked []pair
	for _, f := range all {
		if f.Weight < 0.1 || f.Superseded {
			continue
		}
		doc := factDoc(f)
		ms := agent.MatchScore(contextStr, queryHint, f)
		bm := index.BM25(contextStr+" "+queryHint, doc, 0, 0, 0)
		score := 0.6*ms + 0.4*bm
		if score >= minScore {
			ranked = append(ranked, pair{id: graph.NodeFact(f.ID), score: score})
		}
	}
	if len(ranked) == 0 {
		return nil
	}
	// partial sort topN
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].score > ranked[i].score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	if topN > len(ranked) {
		topN = len(ranked)
	}
	out := make(map[string]float64, topN)
	for i := 0; i < topN; i++ {
		out[ranked[i].id] = ranked[i].score
	}
	return out
}

func pruneComposite(all []facts.Fact, activated map[string]float64, contextStr string, cfg PipelineConfig) []ScoredFact {
	var scored []ScoredFact
	for _, f := range all {
		if f.Weight < 0.1 || f.Superseded {
			continue
		}
		doc := factDoc(f)
		bm := index.BM25(contextStr, doc, 0, 0, 0)
		energy := activated[graph.NodeFact(f.ID)]
		final := cfg.PruneAlpha*bm + cfg.PruneBeta*energy + cfg.PruneGamma*f.Weight
		if f.IsPitfall || isFailOutcome(f.Outcome) {
			final -= cfg.PruneDelta
		}
		if final < cfg.MinAnchorScore*0.5 && energy < cfg.MinAnchorScore*0.5 {
			continue
		}
		scored = append(scored, ScoredFact{Fact: f, Score: final})
	}
	sortScored(scored)
	return scored
}

func factDoc(f facts.Fact) string {
	return f.Text + " " + strings.Join(f.Tags, " ") + " " + strings.Join(f.Tools, " ")
}

func isFailOutcome(o string) bool {
	switch strings.ToLower(strings.TrimSpace(o)) {
	case "fail", "failed", "blocked":
		return true
	default:
		return false
	}
}

func sortScored(scored []ScoredFact) {
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
}
