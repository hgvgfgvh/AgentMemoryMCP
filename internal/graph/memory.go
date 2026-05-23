package graph

import "AgentTestMemoryMCP/internal/facts"

// MemoryGraph 内存邻接表（retrieve 热路径）。
type MemoryGraph struct {
	Adj       map[string][]Edge
	OutDegree map[string]int
}

// NewMemoryGraph 由边列表构建图。
func NewMemoryGraph(edges []Edge) *MemoryGraph {
	g := &MemoryGraph{
		Adj:       make(map[string][]Edge),
		OutDegree: make(map[string]int),
	}
	for _, e := range edges {
		g.Adj[e.From] = append(g.Adj[e.From], e)
		g.OutDegree[e.From]++
		if e.Type == EdgeSupersedes {
			continue
		}
		rev := e
		rev.From, rev.To = e.To, e.From
		g.Adj[rev.From] = append(g.Adj[rev.From], rev)
		if rev.From != e.From {
			g.OutDegree[rev.From]++
		}
	}
	return g
}

// LoadOrDerive 读持久边；若无文件则从 facts 推导。
func LoadOrDerive(dataDir string, all []facts.Fact) (*MemoryGraph, error) {
	edges, err := LoadEdges(dataDir)
	if err != nil {
		return nil, err
	}
	if len(edges) == 0 && len(all) > 0 {
		edges = DeriveEdges(all)
	}
	return NewMemoryGraph(edges), nil
}
