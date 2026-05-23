package memoryagent

import (
	"context"
	"os"
	"testing"

	"AgentTestMemoryMCP/internal/agent"
)

func TestProcessEpisode_RulesFallback(t *testing.T) {
	os.Setenv("MEMORY_MCP_LLM_EXTRACT", "0")
	defer os.Unsetenv("MEMORY_MCP_LLM_EXTRACT")

	content := `[source=agenttest-plan turn=t1 plan=p1]
## 用户诉求
列出 WorkSpace 目录
## 门户回复
已完成
## 计划终态 (TodoList)
status: completed
tools_called: filesystem__list_directory
artifacts: WorkSpace/out.txt
`
	out := ProcessEpisode(context.Background(), "job-t1", "agenttest-plan", "episode", "t1", content, nil)
	if out.Fallback != true || len(out.Facts) != 1 {
		t.Fatalf("fallback=%v facts=%d", out.Fallback, len(out.Facts))
	}
	if out.Facts[0].Outcome != "success" {
		t.Fatalf("outcome=%s", out.Facts[0].Outcome)
	}
}

func TestMergeExtract_Pitfall(t *testing.T) {
	ex := &ExtractResult{
		Summary: SummaryExtract{
			Text: "历史需求: x. 结果: failed.", Outcome: "failed",
			Tools: []string{"filesystem__list_directory"},
		},
		Pitfall: true,
		Atoms: []AtomExtract{
			{Predicate: "triggers", Subject: "list", Object: "failed", Evidence: "status: failed"},
		},
	}
	pp := agent.Preparse{Outcome: "failed", Tools: []string{"filesystem__list_directory"}}
	out := MergeExtract("job-f", "agenttest-plan", "episode", "c1", "status: failed", ex, pp)
	if !out.Facts[0].IsPitfall || out.Facts[0].Weight != 0.3 {
		t.Fatalf("pitfall fact: %+v", out.Facts[0])
	}
}
