package engine

import (
	"os"
	"strconv"
	"time"
)

func retrieveBudget() time.Duration {
	ms := 300
	if v := os.Getenv("MEMORY_MCP_RETRIEVE_BUDGET_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ms = n
		}
	}
	return time.Duration(ms) * time.Millisecond
}

func useLegacyRetrieve() bool {
	return os.Getenv("MEMORY_MCP_RETRIEVE_LEGACY") == "1"
}
