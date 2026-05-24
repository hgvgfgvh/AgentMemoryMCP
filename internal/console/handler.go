package console

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"AgentTestMemoryMCP/internal/facts"
)

//go:embed web/*
var webFS embed.FS

// Server 记忆体开发控制台（只读可视化，不走 MCP 工具面）。
type Server struct {
	dataDir string
	repo    *facts.Repo
}

// NewServer 创建控制台；dataDir 与 factworld 相同。
func NewServer(dataDir string) (*Server, error) {
	if dataDir == "" {
		dataDir = "./data"
	}
	if err := facts.EnsureDataDirs(dataDir); err != nil {
		return nil, err
	}
	repo, err := facts.NewRepo(dataDir)
	if err != nil {
		return nil, err
	}
	return &Server{dataDir: dataDir, repo: repo}, nil
}

// Handler 返回控制台 Handler；由调用方挂载到 /console/（StripPrefix 后为本 mux 根路径）。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/graph", s.handleGraph)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/mcp_retrieve", s.handleMCPRetrieve)
	mux.HandleFunc("/api/stats", s.handleStats)

	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(webRoot)))
	return mux
}

// MountPath 挂到 parent mux 的 /console/ 前缀。
func (s *Server) MountPath(parent *http.ServeMux) {
	parent.Handle("/console/", http.StripPrefix("/console", s.Handler()))
	parent.HandleFunc("/console", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/console/", http.StatusFound)
	})
}

func (s *Server) loadFacts() ([]facts.Fact, error) {
	return s.repo.List()
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	maxFacts := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxFacts = n
		}
	}
	all, err := s.loadFacts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	view := BuildGraphFromFacts(all, maxFacts)
	view.Stats.Facts = len(all)
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 30
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	all, err := s.loadFacts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	hits := SearchFacts(all, q, limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"query": q,
		"hits":  hits,
		"total": len(hits),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	all, err := s.loadFacts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	repoPath := ""
	if s.repo != nil {
		repoPath = s.repo.Path()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"facts_count":  len(all),
		"data_dir":     s.dataDir,
		"facts_path":   repoPath,
		"episodes_dir": filepath.Join(s.dataDir, "episodes"),
		"console_ok":   true,
	})
}

// FactsCount 启动时日志用。
func (s *Server) FactsCount() int {
	all, err := s.loadFacts()
	if err != nil {
		return -1
	}
	return len(all)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
