package align

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"AgentTestMemoryMCP/internal/degenerate"
	"AgentTestMemoryMCP/internal/entity"
	"AgentTestMemoryMCP/internal/facts"
	"AgentTestMemoryMCP/internal/graph"
	"AgentTestMemoryMCP/internal/llm"
)

// Task 模糊带异步对齐任务。
type Task struct {
	DataDir   string
	NewFactID string
	OldFactID string
	Score     float64
}

var (
	globalQ    chan Task
	globalOnce sync.Once
	minAccess  = 2
)

func startWorker() {
	globalQ = make(chan Task, 64)
	go runWorker()
}

func runWorker() {
	for t := range globalQ {
		processTask(t)
	}
}

// Enqueue 非阻塞入队（主 Store 链不等待）。
func Enqueue(t Task) {
	if !asyncEnabled() {
		return
	}
	globalOnce.Do(startWorker)
	select {
	case globalQ <- t:
	default:
		log.Printf("[align] queue full, drop fuzzy %s -> %s", t.NewFactID, t.OldFactID)
	}
}

func asyncEnabled() bool {
	v := strings.TrimSpace(os.Getenv("MEMORY_MCP_ENTITY_ALIGN_ASYNC"))
	return v != "0" && !strings.EqualFold(v, "false")
}

func processTask(t Task) {
	repo, err := facts.NewRepo(t.DataDir)
	if err != nil {
		return
	}
	all, err := repo.List()
	if err != nil {
		return
	}
	var newF, oldF *facts.Fact
	for i := range all {
		if all[i].ID == t.NewFactID {
			newF = &all[i]
		}
		if all[i].ID == t.OldFactID {
			oldF = &all[i]
		}
	}
	if newF == nil || oldF == nil {
		return
	}
	if oldF.AccessCount < minAccessThreshold() && newF.AccessCount < minAccessThreshold() {
		return
	}
	client, ok := llm.ConfigFromEnv()
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	yes, err := askSameEntity(ctx, &client, *newF, *oldF)
	if err != nil {
		log.Printf("[align] llm %s vs %s: %v", t.NewFactID, t.OldFactID, err)
		return
	}
	if !yes {
		return
	}
	cfg := degenerate.DefaultConfig()
	all, edges := degenerate.ApplySupersedes(all, t.NewFactID, []string{t.OldFactID}, cfg)
	if err := repo.Rewrite(all); err != nil {
		log.Printf("[align] rewrite: %v", err)
		return
	}
	base := graph.DeriveEdges(all)
	_ = graph.WriteEdges(t.DataDir, degenerate.MergeExtraEdges(base, edges))
	log.Printf("[align] merged %s supersedes %s (score=%.2f)", t.NewFactID, t.OldFactID, t.Score)
}

func minAccessThreshold() int {
	if minAccess > 0 {
		return minAccess
	}
	if v := os.Getenv("MEMORY_MCP_ENTITY_ALIGN_MIN_ACCESS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 2
}

func askSameEntity(ctx context.Context, c *llm.Client, a, b facts.Fact) (bool, error) {
	sys := `你是实体对齐判定器。仅回答 JSON：{"same":true} 或 {"same":false}。判断两条事实是否描述同一可合并实体/经验（工具别名、路径写法差异算同一）。`
	user := "A: " + entity.FactSignature(a) + "\nB: " + entity.FactSignature(b)
	raw, err := c.ChatJSON(ctx, sys, user)
	if err != nil {
		return false, err
	}
	raw = strings.ToLower(raw)
	return strings.Contains(raw, `"same":true`) || strings.Contains(raw, `"same": true`), nil
}

// EnqueueFuzzyPairs 批量入队。
func EnqueueFuzzyPairs(dataDir string, pairs []entity.FuzzyPair) {
	for _, p := range pairs {
		Enqueue(Task{DataDir: dataDir, NewFactID: p.NewFactID, OldFactID: p.OldFactID, Score: p.Score})
	}
}
