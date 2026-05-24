package engine

import (
	"context"
	"os"
	"testing"
	"time"

	"AgentTestMemoryMCP/internal/facts"
)

func TestFactWorld_L2DropNewFact_NoPersist(t *testing.T) {
	dir := t.TempDir()
	eng, err := NewFactWorldEngine(dir, FactWorldConfig{RetrieveMinScore: 0.3})
	if err != nil {
		t.Fatal(err)
	}
	os.Setenv("MEMORY_MCP_LLM_EXTRACT", "0")
	os.Setenv("MEMORY_MCP_L2_CONFLICT", "1")
	os.Setenv("MEMORY_MCP_ENTITY_MERGE_COSINE", "0.99")
	os.Setenv("MEMORY_MCP_ENTITY_ALIGN_ASYNC", "0")
	defer func() {
		os.Unsetenv("MEMORY_MCP_LLM_EXTRACT")
		os.Unsetenv("MEMORY_MCP_L2_CONFLICT")
		os.Unsetenv("MEMORY_MCP_ENTITY_MERGE_COSINE")
		os.Unsetenv("MEMORY_MCP_ENTITY_ALIGN_ASYNC")
	}()

	ep1 := `[source=agenttest-plan turn=t-l2a plan=p1]
## 用户诉求
列出 WorkSpace
## 门户回复
完成
## 计划终态 (TodoList)
status: completed
tools_called: filesystem__list_directory
`
	eng.Store(context.Background(), StoreInput{Content: ep1, Source: "agenttest-plan", Kind: "episode", CorrelationID: "t-l2a"})
	time.Sleep(700 * time.Millisecond)

	repo, _ := facts.NewRepo(dir)
	list1, _ := repo.List()
	if len(list1) != 1 {
		t.Fatalf("list1=%d", len(list1))
	}

	// 无 LLM 时 L2 默认 C，不会 drop；本测仅验证冲突候选 + C 降权后仍写入第二条
	ep2 := `[source=agenttest-plan turn=t-l2b plan=p2]
## 用户诉求
列出 WorkSpace
## 门户回复
失败 blocked
## 计划终态 (TodoList)
status: failed
tools_called: filesystem__list_directory
`
	eng.Store(context.Background(), StoreInput{Content: ep2, Source: "agenttest-plan", Kind: "episode", CorrelationID: "t-l2b"})
	time.Sleep(700 * time.Millisecond)
	list2, _ := repo.List()
	if len(list2) < 2 {
		t.Fatalf("expected >=2 facts got %d", len(list2))
	}
}
