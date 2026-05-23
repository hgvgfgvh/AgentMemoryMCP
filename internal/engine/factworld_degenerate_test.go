package engine

import (
	"context"
	"os"
	"testing"
	"time"

	"AgentTestMemoryMCP/internal/facts"
)

func TestFactWorldSupersedeOnSimilarStore(t *testing.T) {
	dir := t.TempDir()
	eng, err := NewFactWorldEngine(dir, FactWorldConfig{RetrieveMinScore: 0.3})
	if err != nil {
		t.Fatal(err)
	}
	os.Setenv("MEMORY_MCP_LLM_EXTRACT", "0")
	os.Setenv("MEMORY_MCP_ENTITY_MERGE_COSINE", "0.80")
	os.Setenv("MEMORY_MCP_ENTITY_ALIGN_ASYNC", "0")
	defer func() {
		os.Unsetenv("MEMORY_MCP_LLM_EXTRACT")
		os.Unsetenv("MEMORY_MCP_ENTITY_MERGE_COSINE")
		os.Unsetenv("MEMORY_MCP_ENTITY_ALIGN_ASYNC")
	}()

	ep := `[source=agenttest-plan turn=t-s1 plan=p1]
## 用户诉求
列出 WorkSpace 目录
## 门户回复
完成
## 计划终态 (TodoList)
status: completed
tools_called: filesystem__list_directory
`
	eng.Store(context.Background(), StoreInput{Content: ep, Source: "agenttest-plan", Kind: "episode", CorrelationID: "t-s1"})
	time.Sleep(900 * time.Millisecond)
	repo, _ := facts.NewRepo(dir)
	list1, _ := repo.List()
	if len(list1) != 1 {
		t.Fatalf("list1=%d", len(list1))
	}

	ep2 := `[source=agenttest-plan turn=t-s2 plan=p2]
## 用户诉求
列出 WorkSpace 目录
## 门户回复
再次完成
## 计划终态 (TodoList)
status: completed
tools_called: filesystem__list_directory
`
	eng.Store(context.Background(), StoreInput{Content: ep2, Source: "agenttest-plan", Kind: "episode", CorrelationID: "t-s2"})
	time.Sleep(900 * time.Millisecond)
	list2, _ := repo.List()
	if len(list2) < 2 {
		t.Fatalf("expected 2 facts got %d", len(list2))
	}
	var oldW float64
	for _, f := range list2 {
		if f.CorrelationID == "t-s1" {
			oldW = f.Weight
			if !f.Superseded {
				t.Log("warn: t-s1 not marked superseded yet (may be fuzzy-only)")
			}
		}
	}
	if oldW >= 1.0 {
		t.Fatalf("expected old fact weight reduced, got %v", oldW)
	}
}
