package console

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"AgentTestMemoryMCP/internal/facts"
)

func TestHandleMCPRetrieve(t *testing.T) {
	dir := t.TempDir()
	repo, err := facts.NewRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	_ = repo.Append(facts.Fact{
		ID: "fact-test-1", EpisodeID: "ep-1", Source: "agenttest-plan",
		Text:       "历史需求: 列出 WorkSpace 目录. 结果: completed. 工具: filesystem__list_directory.",
		Tags:       []string{"WorkSpace", "filesystem"},
		Tools:      []string{"filesystem__list_directory"},
		Outcome:    "success",
		Weight:     1.0,
		Confidence: 0.9,
		CreatedAt:  now,
		LastActive: now,
	})

	srv, err := NewServer(dir)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(MCPRetrieveRequest{
		Context:   "用户诉求: 列出 WorkSpace 目录\n",
		QueryHint: "列出 WorkSpace",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/mcp_retrieve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.handleMCPRetrieve(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out MCPRetrieveResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.MCP.Skipped == "true" {
		t.Fatalf("unexpected skip: %+v", out.MCP)
	}
	if out.HintsBody == "" || out.Route == nil {
		t.Fatalf("hints=%q route=%+v", out.HintsBody, out.Route)
	}
	if out.Engine != "factworld" {
		t.Fatalf("engine=%s", out.Engine)
	}
	_ = filepath.Join(dir, "facts", "facts.jsonl")
}
