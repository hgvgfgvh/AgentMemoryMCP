package console

import (
	"testing"
	"time"

	"AgentTestMemoryMCP/internal/facts"
)

func TestBuildGraphFromFacts_linksTagsAndEpisode(t *testing.T) {
	fs := []facts.Fact{
		{
			ID: "f1", EpisodeID: "job-1", Source: "agenttest-plan", CorrelationID: "t-1",
			Text: "列出 WorkSpace", Tags: []string{"WorkSpace", "filesystem"},
			Tools: []string{"filesystem__list_directory"}, Outcome: "success", Confidence: 0.9,
			CreatedAt: time.Now().UTC(),
		},
		{
			ID: "f2", EpisodeID: "job-2", Source: "agenttest-plan", CorrelationID: "t-2",
			Text: "写入文件", Tags: []string{"WorkSpace", "filesystem"},
			Tools: []string{"filesystem__write_file"}, Outcome: "success", Confidence: 0.85,
			CreatedAt: time.Now().UTC(),
		},
	}
	view := BuildGraphFromFacts(fs, 50)
	if view.Stats.Facts != 2 {
		t.Fatalf("facts stat: %d", view.Stats.Facts)
	}
	if view.Stats.Tags < 1 {
		t.Fatal("expected shared tag node")
	}
	hasEpisodeEdge := false
	for _, e := range view.Edges {
		if e.Group == "from_episode" {
			hasEpisodeEdge = true
		}
	}
	if !hasEpisodeEdge {
		t.Fatal("expected from_episode edge")
	}
}

func TestSearchFacts(t *testing.T) {
	fs := []facts.Fact{
		{ID: "f1", Text: "boundary_memory_complex", Tags: []string{"WorkSpace"}},
	}
	hits := SearchFacts(fs, "boundary", 10)
	if len(hits) != 1 || hits[0].NodeID != nodeIDFact("f1") {
		t.Fatalf("hits: %+v", hits)
	}
}
