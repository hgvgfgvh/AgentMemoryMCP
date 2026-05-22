package engine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTestEngine_MatchFixture(t *testing.T) {
	eng, err := NewTestEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	out := eng.Retrieve(ctx, RetrieveInput{
		Context: "用户诉求: 列出 WorkSpace 目录下的文件和子目录，把清单保存",
	})
	if !strings.Contains(out, `"skipped":"false"`) {
		t.Fatalf("retrieve: %s", out)
	}
	if !strings.Contains(out, "exec_simple_match=yes") {
		t.Fatalf("expected match tag: %s", out)
	}
	if !strings.Contains(out, "boundary_plan_dir_list.txt") {
		t.Fatalf("expected artifact hint: %s", out)
	}
}

func TestTestEngine_NoMatch(t *testing.T) {
	eng, _ := NewTestEngine(t.TempDir())
	out := eng.Retrieve(context.Background(), RetrieveInput{Context: "你好"})
	if strings.Contains(out, "exec_simple_match=yes") {
		t.Fatalf("chitchat should not match: %s", out)
	}
}

func TestTestEngine_StoreWritesMarkdown(t *testing.T) {
	dir := t.TempDir()
	eng, _ := NewTestEngine(dir)
	out := eng.Store(context.Background(), StoreInput{
		Content: "episode log for test store",
		Source:  "agenttest-plan",
		Kind:    "episode",
	})
	var payload map[string]string
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("store json: %v out=%s", err, out)
	}
	if payload["accepted"] != "true" {
		t.Fatalf("store: %s", out)
	}
}
