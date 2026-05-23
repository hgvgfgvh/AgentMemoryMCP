package graph

import (
	"fmt"
	"testing"
)

func TestWeightedBFS_HubAttenuation(t *testing.T) {
	// hub "tag:common" 连 20 个 fact，特异 fact 经稀有 tag 应仍有能级
	var edges []Edge
	hub := NodeTag("common")
	for i := 0; i < 20; i++ {
		fid := NodeFact(fmt.Sprintf("f-hub-%d", i))
		edges = append(edges, Edge{From: fid, To: hub, Type: EdgeHasTag, Weight: 0.85})
	}
	rare := NodeTag("rare-xy")
	specific := NodeFact("f-specific")
	edges = append(edges,
		Edge{From: specific, To: rare, Type: EdgeHasTag, Weight: 0.85},
		Edge{From: NodeFact("f-hub-0"), To: specific, Type: EdgeSimilar, Weight: 0.7},
	)
	g := NewMemoryGraph(edges)
	seeds := map[string]float64{NodeFact("f-hub-0"): 1.0}
	act := WeightedBFS(g, seeds, DefaultBFSConfig())
	if act[specific] <= 0 {
		t.Fatalf("expected energy on specific fact, got %v", act[specific])
	}
	// hub 不应激活全部 20（visited 防重复路径；深度限制）
	if len(act) > 15 {
		t.Fatalf("hub exploded activation count=%d", len(act))
	}
}

func TestWeightedBFS_NoInfiniteOnSimilarCycle(t *testing.T) {
	a, b := NodeFact("a"), NodeFact("b")
	edges := []Edge{
		{From: a, To: b, Type: EdgeSimilar, Weight: 0.9},
		{From: b, To: a, Type: EdgeSimilar, Weight: 0.9},
	}
	g := NewMemoryGraph(edges)
	seeds := map[string]float64{a: 1.0}
	act := WeightedBFS(g, seeds, DefaultBFSConfig())
	if len(act) > 10 {
		t.Fatalf("cycle caused runaway activations: %d", len(act))
	}
}
