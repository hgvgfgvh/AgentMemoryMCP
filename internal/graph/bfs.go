package graph

import (
	"math"
)

// BFSConfig Weighted BFS 参数。
type BFSConfig struct {
	MaxDepth        int
	StepAttenuation float64
	MinEnergy       float64
}

// DefaultBFSConfig 专家推荐默认值。
func DefaultBFSConfig() BFSConfig {
	return BFSConfig{
		MaxDepth:        2,
		StepAttenuation: 0.75,
		MinEnergy:       0.25,
	}
}

type bfsItem struct {
	node   string
	depth  int
	energy float64
}

// WeightedBFS 从种子节点扩散激活能级；含出度惩罚与 visited 防环。
func WeightedBFS(g *MemoryGraph, seeds map[string]float64, cfg BFSConfig) map[string]float64 {
	if g == nil || len(seeds) == 0 {
		return map[string]float64{}
	}
	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = 2
	}
	if cfg.StepAttenuation <= 0 {
		cfg.StepAttenuation = 0.75
	}
	if cfg.MinEnergy <= 0 {
		cfg.MinEnergy = 0.25
	}

	activated := make(map[string]float64, len(seeds))
	for id, e := range seeds {
		if e > activated[id] {
			activated[id] = e
		}
	}

	var queue []bfsItem
	visited := map[string]bool{}
	for id, e := range seeds {
		queue = append(queue, bfsItem{node: id, depth: 0, energy: e})
		visited[id] = true
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth >= cfg.MaxDepth {
			continue
		}
		for _, edge := range g.Adj[cur.node] {
			if edge.Type == EdgeSupersedes {
				continue
			}
			v := edge.To
			if visited[v] {
				continue
			}
			factor := cfg.StepAttenuation
			if edge.Type == EdgePitfall {
				factor *= 0.5
			}
			od := g.OutDegree[cur.node]
			if od < 0 {
				od = 0
			}
			factor /= math.Log(3 + float64(od))

			nextEnergy := cur.energy * edge.Weight * factor
			if nextEnergy < cfg.MinEnergy {
				continue
			}
			if nextEnergy > activated[v] {
				activated[v] = nextEnergy
			}
			visited[v] = true
			queue = append(queue, bfsItem{node: v, depth: cur.depth + 1, energy: nextEnergy})
		}
	}
	return activated
}
