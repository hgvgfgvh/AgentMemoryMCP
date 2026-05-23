package memoryagent

import (
	"os"
	"strings"
)

// LLMExtractEnabled 是否尝试 S5 LLM 抽取（须同时配置 API）。
func LLMExtractEnabled() bool {
	v := strings.TrimSpace(os.Getenv("MEMORY_MCP_LLM_EXTRACT"))
	if v == "0" || strings.EqualFold(v, "false") {
		return false
	}
	return true
}
