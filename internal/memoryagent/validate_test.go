package memoryagent

import (
	"testing"

	"AgentTestMemoryMCP/internal/agent"
)

func TestValidateL0_FiltersTools(t *testing.T) {
	ex := &ExtractResult{
		Summary: SummaryExtract{
			Tools:     []string{"filesystem__list_directory", "fake__tool"},
			Artifacts: []string{"WorkSpace/a.txt", "/outside/b.txt"},
			Outcome:   "success",
		},
	}
	pp := agent.Preparse{
		Tools:     []string{"filesystem__list_directory"},
		Artifacts: []string{"WorkSpace/a.txt"},
		Outcome:   "completed",
	}
	ValidateL0(ex, pp)
	if len(ex.Summary.Tools) != 1 || ex.Summary.Tools[0] != "filesystem__list_directory" {
		t.Fatalf("tools: %v", ex.Summary.Tools)
	}
	if len(ex.Summary.Artifacts) != 1 {
		t.Fatalf("artifacts: %v", ex.Summary.Artifacts)
	}
}

func TestValidateL1_DropsBadEvidence(t *testing.T) {
	episode := "用户诉求: 列出 WorkSpace 目录"
	ex := &ExtractResult{
		Atoms: []AtomExtract{
			{Evidence: "列出 WorkSpace 目录", Predicate: "triggers", Subject: "list", Object: "ok"},
			{Evidence: "fabricated sentence not in episode", Predicate: "uses", Subject: "x", Object: "y"},
		},
	}
	kept, dropped := ValidateL1(ex, episode)
	if kept != 1 || dropped != 1 {
		t.Fatalf("kept=%d dropped=%d atoms=%d", kept, dropped, len(ex.Atoms))
	}
}
