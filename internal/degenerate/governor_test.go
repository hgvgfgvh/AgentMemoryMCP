package degenerate

import (
	"testing"
	"time"

	"AgentTestMemoryMCP/internal/facts"
)

func TestApplySupersedes(t *testing.T) {
	all := []facts.Fact{
		{ID: "old", Weight: 1.0},
		{ID: "new", Weight: 1.0},
	}
	out, edges := ApplySupersedes(all, "new", []string{"old"}, DefaultConfig())
	if out[0].Weight != 0.2 || !out[0].Superseded {
		t.Fatalf("old: weight=%v superseded=%v", out[0].Weight, out[0].Superseded)
	}
	if len(edges) != 1 || edges[0].Type != "supersedes" {
		t.Fatalf("edges=%+v", edges)
	}
}

func TestTouchRetrieve(t *testing.T) {
	all := []facts.Fact{{ID: "a", Weight: 1.0}}
	now := time.Now().UTC()
	out := TouchRetrieve(all, []string{"a"}, now, DefaultConfig())
	if out[0].AccessCount != 1 || out[0].Weight <= 1.0 {
		t.Fatalf("access=%d weight=%v", out[0].AccessCount, out[0].Weight)
	}
}
