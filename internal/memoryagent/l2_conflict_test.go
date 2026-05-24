package memoryagent

import (
	"os"
	"testing"
	"time"

	"AgentTestMemoryMCP/internal/facts"
)

func TestDetectConflictCandidates_OppositeOutcomeSameTool(t *testing.T) {
	old := facts.Fact{
		ID: "fact-old", Weight: 1.0, Outcome: "success",
		Tools: []string{"filesystem__list_directory"},
		Tags:  []string{"WorkSpace"},
	}
	newF := facts.Fact{
		ID: "fact-new", Outcome: "fail", IsPitfall: true,
		Tools: []string{"filesystem__list_directory"},
		Tags:  []string{"WorkSpace"},
	}
	cfg := DefaultL2Config()
	cands := DetectConflictCandidates(newF, []facts.Fact{old}, "turn-new", cfg)
	if len(cands) != 1 || cands[0].Old.ID != "fact-old" {
		t.Fatalf("cands=%+v", cands)
	}
}

func TestDetectConflictCandidates_SkipsSameCorrelation(t *testing.T) {
	old := facts.Fact{
		ID: "fact-old", CorrelationID: "t1", Weight: 1.0, Outcome: "success",
		Tools: []string{"filesystem__list_directory"},
	}
	newF := facts.Fact{
		ID: "fact-new", Outcome: "fail", Tools: []string{"filesystem__list_directory"},
	}
	cands := DetectConflictCandidates(newF, []facts.Fact{old}, "t1", DefaultL2Config())
	if len(cands) != 0 {
		t.Fatalf("expected 0 got %d", len(cands))
	}
}

func TestDetectConflictCandidates_SkipsLowWeight(t *testing.T) {
	old := facts.Fact{
		ID: "fact-old", Weight: 0.2, Outcome: "success",
		Tools: []string{"filesystem__list_directory"},
	}
	newF := facts.Fact{
		ID: "fact-new", Outcome: "fail", Tools: []string{"filesystem__list_directory"},
	}
	cands := DetectConflictCandidates(newF, []facts.Fact{old}, "", DefaultL2Config())
	if len(cands) != 0 {
		t.Fatalf("expected 0 got %d", len(cands))
	}
}

func TestApplyL2Decisions_B_DropsNew(t *testing.T) {
	out := ProcessOutput{
		Facts: []facts.Fact{{ID: "new", Confidence: 0.9, Weight: 1}},
		Atoms: []StoredAtom{{ID: "a1"}},
	}
	applyL2Decisions(&out, []l2DecisionRow{{OldFactID: "old", Choice: "B"}}, DefaultL2Config())
	if !out.SkipNewFact || len(out.Facts) != 0 {
		t.Fatalf("skip=%v facts=%d", out.SkipNewFact, len(out.Facts))
	}
}

func TestApplyL2Decisions_A_Supersedes(t *testing.T) {
	out := ProcessOutput{Facts: []facts.Fact{{ID: "new", Confidence: 0.9, Weight: 1}}}
	applyL2Decisions(&out, []l2DecisionRow{{OldFactID: "fact-old", Choice: "A"}}, DefaultL2Config())
	if len(out.SupersedeIDs) != 1 || out.SupersedeIDs[0] != "fact-old" {
		t.Fatalf("supersede=%v", out.SupersedeIDs)
	}
}

func TestApplyL2Decisions_C_LowersConfidence(t *testing.T) {
	out := ProcessOutput{Facts: []facts.Fact{{ID: "new", Confidence: 0.9, Weight: 1}}}
	cfg := DefaultL2Config()
	applyL2Decisions(&out, []l2DecisionRow{{OldFactID: "fact-old", Choice: "C"}}, cfg)
	if out.Facts[0].Confidence >= 0.9 {
		t.Fatalf("confidence=%v", out.Facts[0].Confidence)
	}
}

func TestApplyL2Conflict_NoClient_DefaultC(t *testing.T) {
	os.Setenv("MEMORY_MCP_L2_CONFLICT", "1")
	defer os.Unsetenv("MEMORY_MCP_L2_CONFLICT")

	old := facts.Fact{
		ID: "fact-old", Weight: 1.0, Outcome: "success", CreatedAt: time.Now().UTC(),
		Tools: []string{"filesystem__list_directory"}, Tags: []string{"WorkSpace"},
		Text: "历史需求: 列出目录. 结果: success.",
	}
	newF := facts.Fact{
		ID: "fact-new", Outcome: "fail", IsPitfall: true, Weight: 1, Confidence: 0.9,
		Tools: []string{"filesystem__list_directory"}, Tags: []string{"WorkSpace"},
		Text: "历史需求: 列出目录. 结果: failed.",
	}
	out := ProcessOutput{Facts: []facts.Fact{newF}}
	ApplyL2Conflict(nil, nil, "status: failed", &out, []facts.Fact{old}, "turn-2")
	if !out.L2Applied {
		t.Fatal("expected L2 applied")
	}
	if out.SkipNewFact {
		t.Fatal("default C should not drop new")
	}
	if out.Facts[0].Confidence >= 0.9 {
		t.Fatalf("expected confidence drop, got %v", out.Facts[0].Confidence)
	}
}

func TestL2ConflictEnabled_DefaultOn(t *testing.T) {
	os.Unsetenv("MEMORY_MCP_L2_CONFLICT")
	if !L2ConflictEnabled() {
		t.Fatal("expected default enabled")
	}
	os.Setenv("MEMORY_MCP_L2_CONFLICT", "0")
	defer os.Unsetenv("MEMORY_MCP_L2_CONFLICT")
	if L2ConflictEnabled() {
		t.Fatal("expected disabled")
	}
}
