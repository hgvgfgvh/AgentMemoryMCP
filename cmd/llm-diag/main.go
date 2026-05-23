package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"AgentTestMemoryMCP/internal/agent"
	"AgentTestMemoryMCP/internal/llm"
	"AgentTestMemoryMCP/internal/memoryagent"

	"gopkg.in/yaml.v3"
)

func main() {
	cfgPath := flag.String("config", "", "AgentTest config/app.yaml（加载 mcp_env）")
	flag.Parse()
	if *cfgPath == "" {
		*cfgPath = os.Getenv("AGENTTEST_CONFIG")
	}
	if *cfgPath == "" {
		*cfgPath = filepath.Join("..", "AgentTest", "config", "app.yaml")
	}
	if err := loadMCPEnv(*cfgPath); err != nil {
		fmt.Println("load mcp_env:", err)
		os.Exit(1)
	}
	c, ok := llm.ConfigFromEnv()
	if !ok {
		fmt.Println("FAIL: MEMORY_MCP_LLM_API_BASE empty")
		os.Exit(1)
	}
	fmt.Printf("base=%s model=%s extract=%s\n", c.BaseURL, c.Model, os.Getenv("MEMORY_MCP_LLM_EXTRACT"))

	tmpl, _ := memoryagent.SelectTemplate("agenttest-plan")
	content := `## 用户诉求
列出 WorkSpace 目录
## 门户回复
已完成 WorkSpace 目录列举，清单已写入 llm_smoke_list.txt。
## 计划终态 (TodoList)
status: completed
tools_called: filesystem__list_directory, filesystem__write_file
artifacts: WorkSpace/llm_smoke_list.txt
`
	pp := agent.PreparseEpisode(content)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	ex, err := memoryagent.LLMExtract(ctx, &c, tmpl, "diag", "agenttest-plan", content, pp, nil)
	if err != nil {
		fmt.Printf("LLMExtract FAIL: %v\n", err)
		os.Exit(1)
	}
	kept, dropped := memoryagent.ValidateL1(ex, content)
	fmt.Printf("LLMExtract OK atoms_raw=%d after_L1=%d dropped=%d summary_len=%d\n",
		len(ex.Atoms)+dropped, kept, dropped, len(ex.Summary.Text))

	out := memoryagent.ProcessEpisode(ctx, "job-diag", "agenttest-plan", "episode", "llm-diag-corr", content, nil)
	fmt.Printf("ProcessEpisode: used_llm=%v fallback=%v facts=%d atoms=%d kept=%d drop=%d\n",
		out.UsedLLM, out.Fallback, len(out.Facts), len(out.Atoms), out.AtomsKept, out.AtomsDrop)
	if out.Fallback {
		os.Exit(2)
	}
}

func loadMCPEnv(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root struct {
		Hook struct {
			Env map[string]string `yaml:"mcp_env"`
		} `yaml:"plan_memory_hook"`
	}
	if err := yaml.Unmarshal(b, &root); err != nil {
		return err
	}
	for k, v := range root.Hook.Env {
		os.Setenv(k, v)
	}
	return nil
}
