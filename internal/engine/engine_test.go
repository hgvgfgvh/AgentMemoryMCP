package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStubEngine_StoreRetrieve(t *testing.T) {
	dir := t.TempDir()
	eng, err := NewStubEngine(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	storeOut := eng.Store(ctx, StoreInput{
		Content:       "用户完成 Plan：列出 WorkSpace 目录并写入 boundary_plan_dir_list.txt",
		Source:        "agenttest-plan",
		CorrelationID: "t-1",
	})
	if !strings.Contains(storeOut, `"accepted":"true"`) {
		t.Fatalf("store: %s", storeOut)
	}

	retOut := eng.Retrieve(ctx, RetrieveInput{
		Context: "用户诉求: 读取上次写的清单文件",
	})
	if !strings.Contains(retOut, `"skipped":"false"`) {
		t.Fatalf("retrieve: %s", retOut)
	}
	if !strings.Contains(retOut, "phase-1") {
		t.Fatalf("expected stub hints: %s", retOut)
	}
}

func TestStubEngine_SkipChitchat(t *testing.T) {
	eng, _ := NewStubEngine(t.TempDir())
	out := eng.Store(context.Background(), StoreInput{Content: "你好"})
	if !strings.Contains(out, `"skipped":"true"`) {
		t.Fatalf("store chitchat: %s", out)
	}
	out = eng.Retrieve(context.Background(), RetrieveInput{Context: "谢谢"})
	if !strings.Contains(out, `"skipped":"true"`) {
		t.Fatalf("retrieve chitchat: %s", out)
	}
}

func TestStubEngine_AppendsEpisodeFile(t *testing.T) {
	dir := t.TempDir()
	eng, _ := NewStubEngine(dir)
	eng.Store(context.Background(), StoreInput{Content: "some real task content here"})
	time.Sleep(100 * time.Millisecond)

	path := filepath.Join(dir, "episodes_stub.jsonl")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var ep stubEpisode
	if err := json.Unmarshal(b[:len(b)-1], &ep); err != nil {
		t.Fatalf("jsonl: %v body=%q", err, string(b))
	}
	if ep.JobID == "" {
		t.Fatal("expected job_id")
	}
}
