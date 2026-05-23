package memoryagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestProcessEpisode_LLMPath(t *testing.T) {
	payload := ExtractResult{
		Summary: SummaryExtract{
			Text:    "历史需求: 列出 WorkSpace. 结果: completed. 工具: filesystem__list_directory.",
			Outcome: "success",
			Tools:   []string{"filesystem__list_directory"},
			Tags:    []string{"WorkSpace"},
		},
		Atoms: []AtomExtract{
			{
				SubjectType: "ACTION", Subject: "list_directory", Predicate: "triggers",
				ObjectType: "STATE", Object: "completed", Confidence: 0.9,
				Evidence: "已完成 WorkSpace 目录列举",
			},
		},
	}
	body, _ := json.Marshal(payload)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": string(body)}},
			},
		})
	}))
	defer srv.Close()

	os.Setenv("MEMORY_MCP_LLM_EXTRACT", "1")
	os.Setenv("MEMORY_MCP_LLM_API_BASE", srv.URL)
	os.Setenv("MEMORY_MCP_LLM_API_KEY", "test")
	os.Setenv("MEMORY_MCP_LLM_MODEL", "test-model")
	defer func() {
		os.Unsetenv("MEMORY_MCP_LLM_EXTRACT")
		os.Unsetenv("MEMORY_MCP_LLM_API_BASE")
		os.Unsetenv("MEMORY_MCP_LLM_API_KEY")
		os.Unsetenv("MEMORY_MCP_LLM_MODEL")
	}()

	content := `[source=agenttest-plan]
## 用户诉求
列出 WorkSpace 目录
## 门户回复
已完成 WorkSpace 目录列举
## 计划终态 (TodoList)
status: completed
tools_called: filesystem__list_directory
`
	out := ProcessEpisode(context.Background(), "job-llm", "agenttest-plan", "episode", "llm-1", content, nil)
	if out.Fallback || !out.UsedLLM {
		t.Fatalf("expected llm path fallback=%v used=%v", out.Fallback, out.UsedLLM)
	}
	if len(out.Facts) != 1 || len(out.Atoms) != 1 {
		t.Fatalf("facts=%d atoms=%d", len(out.Facts), len(out.Atoms))
	}
}
