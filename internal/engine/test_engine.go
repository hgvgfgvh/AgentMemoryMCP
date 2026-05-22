package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"AgentTestMemoryMCP/internal/filter"
	"AgentTestMemoryMCP/internal/response"
)

// TestEngine Phase-1 测试实现：内嵌一份已完成 TodoList 样本；命中则返回固定成功 hints；store 仅落盘文档。
type TestEngine struct {
	dataDir string
	fixture FixturePlan
}

// NewTestEngine 加载 testdata/fixture_completed_todolist.json（编译期 embed）。
func NewTestEngine(dataDir string) (*TestEngine, error) {
	if dataDir == "" {
		dataDir = "data"
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "store_log"), 0o755); err != nil {
		return nil, err
	}
	fp, err := loadFixturePlan()
	if err != nil {
		return nil, fmt.Errorf("load fixture: %w", err)
	}
	return &TestEngine{dataDir: dataDir, fixture: fp}, nil
}

func (e *TestEngine) Store(ctx context.Context, in StoreInput) string {
	_ = ctx
	if skip, reason := filter.ShouldSkipStore(in.Content); skip {
		return response.FormatStore(response.StorePayload{
			Accepted:   "false",
			Skipped:    "true",
			SkipReason: reason,
			Message:    "store skipped by filter",
			Phase:      "test-fixture",
		})
	}
	path, err := e.appendStoreDoc(in)
	msg := "accepted; recorded as markdown under data/store_log (test engine, no fact graph)"
	if err != nil {
		msg = "accepted but write failed: " + err.Error()
	}
	if path != "" {
		msg += "; path=" + path
	}
	return response.FormatStore(response.StorePayload{
		Accepted: "true",
		JobID:    fmt.Sprintf("test-store-%d", time.Now().Unix()),
		Skipped:  "false",
		Message:  msg,
		Phase:    "test-fixture",
	})
}

func (e *TestEngine) Retrieve(ctx context.Context, in RetrieveInput) string {
	_ = ctx
	if skip, reason := filter.ShouldSkipRetrieve(in.Context); skip {
		return response.FormatRetrieve(response.RetrievePayload{
			Hints:      "",
			Skipped:    "true",
			SkipReason: reason,
			Phase:      "test-fixture",
		})
	}
	if !contextMatchesFixture(in.Context, e.fixture) {
		return response.FormatRetrieve(response.RetrievePayload{
			Hints:   e.buildNoMatchHints(),
			Skipped: "false",
			Phase:   "test-fixture",
		})
	}
	return response.FormatRetrieve(response.RetrievePayload{
		Hints:   e.buildMatchedHints(),
		Skipped: "false",
		Phase:   "test-fixture",
	})
}

func (e *TestEngine) buildMatchedHints() string {
	fp := e.fixture
	var b strings.Builder
	b.WriteString("[exec_simple_match=yes confidence=0.88]\n")
	b.WriteString("【记忆命中·测试固定样本】\n")
	b.WriteString("说明: TestEngine 内嵌已完成 TodoList；非事实图抽取。\n")
	b.WriteString("历史用户需求: " + fp.UserRequirement + "\n")
	b.WriteString("计划摘要: " + fp.Summary + "\n")
	b.WriteString("计划状态: " + fp.Status + "\n")
	if fp.ExecutionMode != "" {
		b.WriteString("执行模式: " + fp.ExecutionMode + "\n")
	}
	for i, s := range fp.Steps {
		b.WriteString(fmt.Sprintf("步骤%d [%s] %s tier=%d status=%s\n", i+1, s.ID, s.Title, s.Tier, s.Status))
		if s.Instruction != "" {
			b.WriteString("  instruction: " + s.Instruction + "\n")
		}
		if len(s.CapabilityHints) > 0 {
			b.WriteString("  capability_hints: " + strings.Join(s.CapabilityHints, ", ") + "\n")
		}
		if s.ResultSummary != "" {
			b.WriteString("  result_summary: " + s.ResultSummary + "\n")
		}
		if len(s.Artifacts) > 0 {
			b.WriteString("  artifacts: " + strings.Join(s.Artifacts, ", ") + "\n")
		}
		if len(s.ToolsCalled) > 0 {
			b.WriteString("  tools_called: " + strings.Join(s.ToolsCalled, ", ") + "\n")
		}
	}
	b.WriteString("路径提示: 可复用 filesystem 列出 WorkSpace 并写入 boundary_plan_dir_list.txt\n")
	return b.String()
}

func (e *TestEngine) buildNoMatchHints() string {
	return "【跨会话事实参考】\n(test-fixture: 未命中内嵌样本关键词；请检查 context 是否含 WorkSpace/列出/目录 等)\n"
}

func (e *TestEngine) appendStoreDoc(in StoreInput) (string, error) {
	name := fmt.Sprintf("store_%s.md", time.Now().Format("20060102_150405"))
	path := filepath.Join(e.dataDir, "store_log", name)
	body := fmt.Sprintf("# memory_store 测试记录\n\n- stored_at: %s\n- source: %s\n- kind: %s\n- correlation_id: %s\n\n## content\n\n%s\n",
		time.Now().UTC().Format(time.RFC3339),
		trim(in.Source),
		trim(in.Kind),
		trim(in.CorrelationID),
		in.Content,
	)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
