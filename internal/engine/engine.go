// Package engine 定义事实世界引擎接口。Phase-1 仅 StubEngine：不实现图、记忆 Agent、向量索引。
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"AgentTestMemoryMCP/internal/filter"
	"AgentTestMemoryMCP/internal/response"
)

// StoreInput memory_store 入参（协议层均为 string）。
type StoreInput struct {
	Content       string
	Source        string
	Kind          string
	CorrelationID string
}

// RetrieveInput memory_retrieve 入参。
type RetrieveInput struct {
	Context   string
	QueryHint string
}

// Engine 事实世界：对外 store/retrieve；内部实现可替换。
type Engine interface {
	Store(ctx context.Context, in StoreInput) string
	Retrieve(ctx context.Context, in RetrieveInput) string
}

// StubEngine Phase-1：异步将 episode 追加到 JSONL；retrieve 返回占位 hints（无图、无 Agent）。
type StubEngine struct {
	dataDir string
	jobSeq  atomic.Uint64
	wg      sync.WaitGroup
}

// NewStubEngine dataDir 为空时使用 ./data。
func NewStubEngine(dataDir string) (*StubEngine, error) {
	if dataDir == "" {
		dataDir = "data"
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	return &StubEngine{dataDir: dataDir}, nil
}

// Store 同步 ACK + 异步写入 stub 队列文件（非事实图）。
func (e *StubEngine) Store(ctx context.Context, in StoreInput) string {
	if skip, reason := filter.ShouldSkipStore(in.Content); skip {
		return response.FormatStore(response.StorePayload{
			Accepted:   "false",
			JobID:      "",
			Skipped:    "true",
			SkipReason: reason,
			Message:    "store skipped by filter",
			Phase:      response.PhaseStub(),
		})
	}

	jobID := fmt.Sprintf("job-%d-%d", time.Now().Unix(), e.jobSeq.Add(1))
	payload := response.FormatStore(response.StorePayload{
		Accepted: "true",
		JobID:    jobID,
		Skipped:  "false",
		Message:  "accepted; processing asynchronously (phase-1 stub, no fact graph)",
		Phase:    response.PhaseStub(),
	})

	record := stubEpisode{
		JobID:          jobID,
		StoredAt:       time.Now().UTC().Format(time.RFC3339),
		Source:         in.Source,
		Kind:           in.Kind,
		CorrelationID:  in.CorrelationID,
		ContentLen:     len([]rune(in.Content)),
		ContentPreview: preview(in.Content, 400),
		Note:           "phase-1 stub: not processed by memory agent",
	}

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		_ = e.appendEpisode(context.Background(), record)
	}()

	return payload
}

// Retrieve 同步、快速；不调用 Memory Agent。
func (e *StubEngine) Retrieve(ctx context.Context, in RetrieveInput) string {
	_ = ctx
	if skip, reason := filter.ShouldSkipRetrieve(in.Context); skip {
		return response.FormatRetrieve(response.RetrievePayload{
			Hints:      "",
			Skipped:    "true",
			SkipReason: reason,
			Phase:      response.PhaseStub(),
		})
	}

	hints := e.buildStubHints(in)
	return response.FormatRetrieve(response.RetrievePayload{
		Hints:   hints,
		Skipped: "false",
		Phase:   response.PhaseStub(),
	})
}

func (e *StubEngine) buildStubHints(in RetrieveInput) string {
	// Phase-1：若有已接受的 episode，返回极简摘要；否则明确占位文案。
	n, last := e.countEpisodes()
	if n == 0 {
		return "【跨会话事实参考】\n(phase-1: 事实世界尚未建立；memory_store 已接入但 Memory Agent / 图索引未实现。)"
	}
	var b string
	b = "【跨会话事实参考】\n"
	b += fmt.Sprintf("(phase-1 stub: 已接收 %d 条 episode 材料，尚未做事实抽取与图关联。)\n", n)
	if last != "" {
		b += "最近一条材料摘要: " + last + "\n"
	}
	if hint := trim(in.QueryHint); hint != "" {
		b += "query_hint 已记录（未参与检索）: " + preview(hint, 120) + "\n"
	}
	return b
}

type stubEpisode struct {
	JobID          string `json:"job_id"`
	StoredAt       string `json:"stored_at"`
	Source         string `json:"source,omitempty"`
	Kind           string `json:"kind,omitempty"`
	CorrelationID  string `json:"correlation_id,omitempty"`
	ContentLen     int    `json:"content_len"`
	ContentPreview string `json:"content_preview"`
	Note           string `json:"note"`
}

func (e *StubEngine) episodesPath() string {
	return filepath.Join(e.dataDir, "episodes_stub.jsonl")
}

func (e *StubEngine) appendEpisode(ctx context.Context, ep stubEpisode) error {
	_ = ctx
	b, err := json.Marshal(ep)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(e.episodesPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func (e *StubEngine) countEpisodes() (int, string) {
	b, err := os.ReadFile(e.episodesPath())
	if err != nil || len(b) == 0 {
		return 0, ""
	}
	lines := splitLines(string(b))
	n := 0
	var lastPreview string
	for _, line := range lines {
		line = trim(line)
		if line == "" {
			continue
		}
		var ep stubEpisode
		if json.Unmarshal([]byte(line), &ep) == nil {
			n++
			if ep.ContentPreview != "" {
				lastPreview = ep.ContentPreview
			}
		}
	}
	return n, lastPreview
}

func preview(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return string(r)
	}
	return string(r[:maxRunes]) + "…"
}

func trim(s string) string {
	const cut = " \t\r\n"
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\r' || s[0] == '\n') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last != ' ' && last != '\t' && last != '\r' && last != '\n' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
