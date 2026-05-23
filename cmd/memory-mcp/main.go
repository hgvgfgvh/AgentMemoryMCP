// memory-mcp：长效事实记忆 MCP Server（Phase-1 架构骨架）。
//
// 对外工具：memory_store、memory_retrieve（字符串协议）。
// 内部：StubEngine，不实现记忆 Agent / 图结构 / 向量索引。
//
// 用法：
//
//	stdio（默认）: memory-mcp.exe
//	HTTP:         memory-mcp.exe -http 127.0.0.1:8090
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"AgentTestMemoryMCP/internal/console"
	"AgentTestMemoryMCP/internal/engine"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	httpAddr    = flag.String("http", "", "若设置则使用 Streamable HTTP（MCP），并可同时挂载 /console/ 开发 UI")
	consoleAddr = flag.String("console", "", "仅启动记忆拓扑开发控制台（只读），如 127.0.0.1:8091")
	dataDir     = flag.String("data", "", "数据目录（默认 ./data 或环境变量 MEMORY_MCP_DATA_DIR）")
	engineKind  = flag.String("engine", "", "引擎：factworld（默认）| stub | test（内嵌样本）")
)

func main() {
	flag.Parse()
	log.SetOutput(os.Stderr)

	dir := *dataDir
	if dir == "" {
		dir = os.Getenv("MEMORY_MCP_DATA_DIR")
	}
	kind := trim(*engineKind)
	if kind == "" {
		kind = os.Getenv("MEMORY_MCP_ENGINE")
	}
	if kind == "" {
		kind = "factworld"
	}
	var eng engine.Engine
	var err error
	switch strings.ToLower(kind) {
	case "stub":
		eng, err = engine.NewStubEngine(dir)
		log.Printf("[memory-mcp] engine=stub")
	case "test", "fixture":
		eng, err = engine.NewTestEngine(dir)
		log.Printf("[memory-mcp] engine=test-fixture")
	default:
		eng, err = engine.NewFactWorldEngine(dir, engine.FactWorldConfig{})
		log.Printf("[memory-mcp] engine=factworld (rules extract + JSONL index)")
	}
	if err != nil {
		log.Fatalf("engine: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "agent-test-memory",
		Title:   "AgentTest Memory MCP",
		Version: "0.1.0-phase1",
	}, nil)

	registerTools(server, eng)

	consoleSrv, consoleErr := console.NewServer(dir)
	if consoleErr != nil {
		log.Printf("[memory-mcp] console disabled: %v", consoleErr)
		consoleSrv = nil
	}

	if addr := trim(*consoleAddr); addr != "" && trim(*httpAddr) == "" {
		if consoleSrv == nil {
			log.Fatalf("console: %v", consoleErr)
		}
		log.Printf("[memory-mcp] dev console http://%s/console/ (read-only topology)", addr)
		log.Printf("[memory-mcp] console data_dir=%s facts=%d", dir, consoleSrv.FactsCount())
		if err := http.ListenAndServe(addr, consoleWithRoot(consoleSrv)); err != nil {
			log.Fatalf("console: %v", err)
		}
		return
	}

	if addr := trim(*httpAddr); addr != "" {
		mux := http.NewServeMux()
		mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		mux.Handle("/", mcpHandler)
		if consoleSrv != nil {
			consoleSrv.MountPath(mux)
			log.Printf("[memory-mcp] dev console http://%s/console/", addr)
		}
		log.Printf("[memory-mcp] streamable HTTP listening on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("http: %v", err)
		}
		return
	}

	if consoleSrv != nil {
		startConsoleBackground(consoleSrv, dir)
	}

	t := &mcp.LoggingTransport{Transport: &mcp.StdioTransport{}, Writer: os.Stderr}
	log.Printf("[memory-mcp] stdio transport")
	if err := server.Run(context.Background(), t); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// startConsoleBackground stdio 模式下由 MCP 进程内伴生启动开发控制台（Host 无感）。
// 环境变量：MEMORY_MCP_CONSOLE_LISTEN（默认 127.0.0.1:8091）；MEMORY_MCP_CONSOLE_DISABLE=1 关闭。
func startConsoleBackground(consoleSrv *console.Server, dataDir string) {
	if envTruthy(os.Getenv("MEMORY_MCP_CONSOLE_DISABLE")) {
		log.Printf("[memory-mcp] dev console disabled (MEMORY_MCP_CONSOLE_DISABLE)")
		return
	}
	addr := trim(os.Getenv("MEMORY_MCP_CONSOLE_LISTEN"))
	if addr == "" {
		addr = "127.0.0.1:8091"
	}
	go func() {
		log.Printf("[memory-mcp] dev console (stdio 伴生) http://%s/console/ data_dir=%s facts=%d",
			addr, dataDir, consoleSrv.FactsCount())
		if err := http.ListenAndServe(addr, consoleWithRoot(consoleSrv)); err != nil {
			log.Printf("[memory-mcp] dev console exit: %v", err)
		}
	}()
}

func envTruthy(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "1" || v == "true" || v == "yes"
}

func registerTools(server *mcp.Server, eng engine.Engine) {
	type storeArgs struct {
		Content       string `json:"content" jsonschema:"required,待沉淀的原始文本（单条 episode）"`
		Source        string `json:"source,omitempty" jsonschema:"Host 标识，如 agenttest-plan"`
		Kind          string `json:"kind,omitempty" jsonschema:"粗分类，如 episode、note"`
		CorrelationID string `json:"correlation_id,omitempty" jsonschema:"Host 关联 ID，如 turn_id"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name: "memory_store",
		Description: `存入事实材料（字符串协议）。外部仅投递 content；关系与图由内部 Memory Agent 处理（Phase-1 未实现）。
同步返回 JSON 字符串：accepted、job_id、skipped 等字段均为 string。存入可异步，本工具先 ACK。`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args storeArgs) (*mcp.CallToolResult, any, error) {
		out := eng.Store(ctx, engine.StoreInput{
			Content:       args.Content,
			Source:        args.Source,
			Kind:          args.Kind,
			CorrelationID: args.CorrelationID,
		})
		return textResult(out), nil, nil
	})

	type retrieveArgs struct {
		Context   string `json:"context" jsonschema:"required,Host 本轮完整上下文字符串（含用户输入与会话级摘要）"`
		QueryHint string `json:"query_hint,omitempty" jsonschema:"可选补充检索意图，不能替代 context"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name: "memory_retrieve",
		Description: `取出记忆参考提示（字符串协议）。须传 context；由内部 Memory Agent 裁切 hints（Phase-1 返回 stub 占位）。
同步、快速。返回 JSON 字符串：hints、skipped 等字段均为 string。`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args retrieveArgs) (*mcp.CallToolResult, any, error) {
		out := eng.Retrieve(ctx, engine.RetrieveInput{
			Context:   args.Context,
			QueryHint: args.QueryHint,
		})
		return textResult(out), nil, nil
	})
}

func textResult(jsonText string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: jsonText},
		},
	}
}

// consoleWithRoot 独立控制台模式：/ 重定向到 /console/。
func consoleWithRoot(srv *console.Server) http.Handler {
	mux := http.NewServeMux()
	srv.MountPath(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/console/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
	return mux
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 {
		c := s[len(s)-1]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}
