package console

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"AgentTestMemoryMCP/internal/engine"
	"AgentTestMemoryMCP/internal/response"
	"AgentTestMemoryMCP/internal/retrieve"
)

const maxRetrieveBody = 512 * 1024

// MCPRetrieveRequest 与 memory_retrieve 入参一致。
type MCPRetrieveRequest struct {
	Context   string `json:"context"`
	QueryHint string `json:"query_hint"`
}

// MCPRetrieveResponse 供开发台展示；mcp 字段与 MCP 工具返回 JSON 一致。
type MCPRetrieveResponse struct {
	MCP        response.RetrievePayload `json:"mcp"`
	Route      *RouteView               `json:"route,omitempty"`
	HintsBody  string                   `json:"hints_body"`
	Note       string                   `json:"note"`
	Engine     string                   `json:"engine"`
	DurationMs int64                    `json:"duration_ms"`
}

// RouteView 从 hints 解析的 memory-route。
type RouteView struct {
	ExecSimpleMatch string   `json:"exec_simple_match"`
	Confidence      float64  `json:"confidence"`
	FactIDs         []string `json:"fact_ids,omitempty"`
}

func (s *Server) handleMCPRetrieve(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRetrieveBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body: " + err.Error()})
		return
	}
	var req MCPRetrieveRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
			return
		}
	}
	req.Context = strings.TrimSpace(req.Context)
	req.QueryHint = strings.TrimSpace(req.QueryHint)
	if req.Context == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "context is required"})
		return
	}

	eng, engName, err := newRetrieveEngine(s.dataDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	start := time.Now()
	raw := eng.Retrieve(context.Background(), engine.RetrieveInput{
		Context:   req.Context,
		QueryHint: req.QueryHint,
	})
	elapsed := time.Since(start).Milliseconds()

	var payload response.RetrievePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "decode mcp response: " + err.Error()})
		return
	}

	out := MCPRetrieveResponse{
		MCP:        payload,
		HintsBody:  hintsBodyOnly(payload.Hints),
		Engine:     engName,
		DurationMs: elapsed,
		Note:       "与 MCP 工具 memory_retrieve 相同引擎路径（图 BFS + BM25）；命中 fact 会写回 last_active，与生产一致。",
	}
	if route := parseRouteView(payload.Hints); route != nil {
		out.Route = route
	}
	writeJSON(w, http.StatusOK, out)
}

func newRetrieveEngine(dataDir string) (engine.Engine, string, error) {
	eng, err := engine.NewFactWorldEngine(dataDir, engine.FactWorldConfig{})
	if err != nil {
		return nil, "", err
	}
	return eng, "factworld", nil
}

func hintsBodyOnly(hints string) string {
	idx := strings.Index(hints, "---memory-route---")
	if idx < 0 {
		return strings.TrimSpace(hints)
	}
	return strings.TrimSpace(hints[:idx])
}

func parseRouteView(hints string) *RouteView {
	match, conf, ok := retrieve.ParseRouteBlock(hints)
	if !ok {
		return nil
	}
	v := &RouteView{ExecSimpleMatch: match, Confidence: conf}
	idx := strings.Index(hints, "---memory-route---")
	if idx < 0 {
		return v
	}
	rest := hints[idx+len("---memory-route---"):]
	start := strings.Index(rest, "{")
	end := strings.LastIndex(rest, "}")
	if start < 0 || end <= start {
		return v
	}
	var m map[string]any
	if json.Unmarshal([]byte(rest[start:end+1]), &m) != nil {
		return v
	}
	if ids, ok := m["fact_ids"].([]any); ok {
		for _, id := range ids {
			if s, ok := id.(string); ok && s != "" {
				v.FactIDs = append(v.FactIDs, s)
			}
		}
	}
	return v
}
