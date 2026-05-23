package entity

import (
	"testing"

	"AgentTestMemoryMCP/internal/facts"
)

func TestToolCosine_FilesystemAlias(t *testing.T) {
	s := ToolCosine("filesystem", "file-system")
	if s < 0.92 {
		t.Fatalf("expected >=0.92 got %.2f", s)
	}
}

func TestCompareFacts_HardMerge(t *testing.T) {
	newF := facts.Fact{
		ID: "new", Text: "历史需求: 列出 WorkSpace 目录",
		Tools: []string{"filesystem__list_directory"}, Tags: []string{"WorkSpace"},
		Weight: 1,
	}
	old := facts.Fact{
		ID: "old", Text: "历史需求: 列出 WorkSpace 目录",
		Tools: []string{"filesystem__list_directory"}, Tags: []string{"WorkSpace"},
		Weight: 1,
	}
	sup, fuzzy := CompareFacts(newF, []facts.Fact{old}, "", AlignConfig{HardMerge: 0.92, FuzzyLow: 0.85})
	if len(sup) != 1 || sup[0] != "old" {
		t.Fatalf("expected hard merge sup=%v fuzzy=%v", sup, fuzzy)
	}
}
