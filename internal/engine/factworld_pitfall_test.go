package engine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFactWorldPitfallBlocksExecSimple(t *testing.T) {
	dir := t.TempDir()
	eng, err := NewFactWorldEngine(dir, FactWorldConfig{RouteThreshold: 0.75, RetrieveMinScore: 0.3})
	if err != nil {
		t.Fatal(err)
	}
	content := `[source=agenttest-plan turn=t-fail plan=p-fail]

## 用户诉求
列出 WorkSpace 目录并保存清单

## 门户回复
代理未配置导致失败

## 计划终态 (TodoList)
status: failed
tools_called: filesystem__list_directory
`
	eng.Store(context.Background(), StoreInput{
		Content: content, Source: "agenttest-plan", Kind: "episode", CorrelationID: "t-fail",
	})
	time.Sleep(800 * time.Millisecond)

	ctx := "用户诉求: 列出 WorkSpace 目录并保存清单到文件\n"
	retOut := eng.Retrieve(context.Background(), RetrieveInput{Context: ctx})
	var payload struct {
		Hints string `json:"hints"`
	}
	if err := json.Unmarshal([]byte(retOut), &payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload.Hints, "exec_simple_match") {
		t.Fatalf("missing route: %s", payload.Hints)
	}
	if strings.Contains(payload.Hints, `"exec_simple_match":"yes"`) {
		t.Fatalf("pitfall should block exec-simple: %s", payload.Hints)
	}
}
