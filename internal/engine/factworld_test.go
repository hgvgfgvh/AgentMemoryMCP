package engine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"AgentTestMemoryMCP/internal/facts"
	"AgentTestMemoryMCP/internal/retrieve"
)

func TestFactWorldStoreRetrieveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	eng, err := NewFactWorldEngine(dir, FactWorldConfig{RouteThreshold: 0.75, RetrieveMinScore: 0.3})
	if err != nil {
		t.Fatal(err)
	}
	content := `[source=agenttest-plan turn=t-1 plan=test-plan]

## 用户诉求
列出 WorkSpace 目录下的文件和子目录，把清单保存到 WorkSpace/boundary_plan_dir_list.txt

## 门户回复
已列出并写入清单。

## 计划终态 (TodoList)
status: completed
tools_called: filesystem__list_directory, filesystem__write_file
artifacts: WorkSpace/boundary_plan_dir_list.txt
`
	in := StoreInput{
		Content: content, Source: "agenttest-plan", Kind: "episode",
		CorrelationID: "t-1",
	}
	storeOut := eng.Store(context.Background(), in)
	if !strings.Contains(storeOut, `"accepted":"true"`) {
		t.Fatalf("store: %s", storeOut)
	}
	time.Sleep(800 * time.Millisecond)
	repo, _ := facts.NewRepo(dir)
	list, err := repo.List()
	if err != nil || len(list) == 0 {
		t.Fatalf("facts not persisted: %v len=%d", err, len(list))
	}
	ctx := "用户诉求: 列出 WorkSpace 目录下的文件和子目录，把清单保存到 WorkSpace/boundary_plan_dir_list.txt\n"
	retOut := eng.Retrieve(context.Background(), RetrieveInput{Context: ctx, QueryHint: ctx})
	if !strings.Contains(retOut, "factworld") && !strings.Contains(retOut, "命中") {
		t.Fatalf("retrieve: %s", retOut)
	}
	var payload struct {
		Hints string `json:"hints"`
	}
	if err := json.Unmarshal([]byte(retOut), &payload); err != nil {
		t.Fatalf("parse retrieve json: %v", err)
	}
	match, conf, ok := retrieve.ParseRouteBlock(payload.Hints)
	if !ok || match != "yes" || conf < 0.75 {
		t.Fatalf("route block: ok=%v match=%s conf=%v", ok, match, conf)
	}
}
