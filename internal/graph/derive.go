package graph

import (
	"sort"
	"strings"

	"AgentTestMemoryMCP/internal/facts"
)

// DeriveEdges 由 facts 推导全量边（与线上一致，供 store 重建与 retrieve 冷启动）。
func DeriveEdges(all []facts.Fact) []Edge {
	if len(all) == 0 {
		return nil
	}
	tagCount := map[string]int{}
	for _, f := range all {
		for _, tag := range f.Tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagCount[tag]++
			}
		}
	}

	var edges []Edge
	for _, f := range all {
		nid := NodeFact(f.ID)
		if eid := strings.TrimSpace(f.EpisodeID); eid != "" {
			edges = append(edges, Edge{
				From: nid, To: NodeEpisode(eid), Type: EdgeFromEpisode,
				Weight: edgeWeight(EdgeFromEpisode), EpisodeID: eid,
			})
		}
		for _, tag := range f.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" || tagCount[tag] < 2 {
				continue
			}
			edges = append(edges, Edge{
				From: nid, To: NodeTag(tag), Type: EdgeHasTag,
				Weight: edgeWeight(EdgeHasTag), EpisodeID: f.EpisodeID,
			})
		}
		for _, tool := range f.Tools {
			tool = strings.TrimSpace(tool)
			if tool == "" {
				continue
			}
			edges = append(edges, Edge{
				From: nid, To: NodeTool(tool), Type: EdgeUsedTool,
				Weight: edgeWeight(EdgeUsedTool), EpisodeID: f.EpisodeID,
			})
		}
		if src := strings.TrimSpace(f.Source); src != "" {
			edges = append(edges, Edge{
				From: nid, To: NodeSource(src), Type: EdgeSource,
				Weight: edgeWeight(EdgeSource), EpisodeID: f.EpisodeID,
			})
		}
		if f.IsPitfall || isFailOutcome(f.Outcome) {
			edges = append(edges, Edge{
				From: nid, To: NodeTag("pitfall"), Type: EdgePitfall,
				Weight: edgeWeight(EdgePitfall), EpisodeID: f.EpisodeID,
			})
		}
	}

	corrIndex := map[string][]string{}
	for _, f := range all {
		c := strings.TrimSpace(f.CorrelationID)
		if c == "" {
			continue
		}
		corrIndex[c] = append(corrIndex[c], NodeFact(f.ID))
	}
	for _, ids := range corrIndex {
		if len(ids) < 2 {
			continue
		}
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				edges = append(edges, Edge{
					From: ids[i], To: ids[j], Type: EdgeSameCorrelation,
					Weight: edgeWeight(EdgeSameCorrelation),
				})
			}
		}
	}

	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if tagJaccard(all[i].Tags, all[j].Tags) >= 0.35 {
				edges = append(edges, Edge{
					From: NodeFact(all[i].ID), To: NodeFact(all[j].ID),
					Type: EdgeSimilar, Weight: edgeWeight(EdgeSimilar),
				})
			}
		}
	}
	return edges
}

func isFailOutcome(o string) bool {
	switch strings.ToLower(strings.TrimSpace(o)) {
	case "fail", "failed", "blocked":
		return true
	default:
		return false
	}
}

func tagJaccard(a, b []string) float64 {
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

// SortEdges 稳定排序（测试与 diff）。
func SortEdges(edges []Edge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Type < edges[j].Type
	})
}
