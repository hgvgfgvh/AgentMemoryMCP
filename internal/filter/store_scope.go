package filter

import (
	"regexp"
	"strings"
)

// 执行类 episode 信号（工具、产物、计划步结果等）。
var executionMarkers = []string{
	"tools_called:",
	"tools:",
	"artifacts:",
	"artifact:",
	"filesystem__",
	"sqlite__",
	"resend__",
	"list_agent_capabilities",
	"get_capability_details",
	"setexecutorstep",
	"report_step",
	"boundary_",
	"workSpace/",
	"workspace/",
	".txt",
	".json",
	".yaml",
	"tier=",
	"step1:",
	"step2:",
	"result_summary:",
	"execution_mode:",
	"## 计划终态",
	"## 处理错误",
	"processerror",
	"exec_simple",
	"memory_store",
	"skill",
	"mcp__",
	"__",
}

// Soul MCP 职责：称呼、身份、寒暄、人格/议题（无执行证据时不写入 Memory）。
var soulBoundaryUserPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^\s*(你好|您好|hi|hello)?\s*我是谁\s*$`),
	regexp.MustCompile(`(?i)我是谁`),
	regexp.MustCompile(`(?i)^叫我`),
	regexp.MustCompile(`(?i)请记住`),
	regexp.MustCompile(`(?i)称呼`),
	regexp.MustCompile(`(?i)身份确认`),
	regexp.MustCompile(`(?i)soul\s*mcp`),
	regexp.MustCompile(`(?i)人格记忆`),
	regexp.MustCompile(`(?i)协作人格`),
	regexp.MustCompile(`(?i)我是\s*chalice`),
	regexp.MustCompile(`(?i)老王是测试`),
}

// classifyStoreScope 返回是否应跳过 store 及原因。
func classifyStoreScope(content string) (skip bool, reason string) {
	raw := strings.TrimSpace(content)
	if raw == "" {
		return true, "empty_content"
	}
	wholeKey := trimForMatch(raw)
	if _, ok := chitchatExact[wholeKey]; ok {
		return true, "chitchat"
	}
	if len([]rune(raw)) < 8 {
		return true, "too_short"
	}

	userReq := extractSection(raw, "用户诉求")
	if userReq == "" {
		userReq = extractSection(raw, "用户（WebUI）")
	}
	if userReq == "" {
		userReq = extractSection(raw, "用户")
	}
	userKey := trimForMatch(userReq)
	if userKey != "" {
		if _, ok := chitchatExact[userKey]; ok {
			return true, "chitchat"
		}
		if isSoulBoundaryUserOnly(userKey) && !hasExecutionSignals(raw) {
			return true, "soul_boundary"
		}
	}

	// 计划门户 episode：无执行信号则视为非 Memory 职责（寒暄/纯语义/仅门户复述）。
	if isPlanEpisode(raw) && !hasExecutionSignals(raw) {
		return true, "not_execution_scope"
	}

	// 非 plan 格式的短文本：无执行线索且像寒暄/身份。
	if !isPlanEpisode(raw) && !hasExecutionSignals(raw) {
		whole := trimForMatch(raw)
		if _, ok := chitchatExact[whole]; ok {
			return true, "chitchat"
		}
		if len([]rune(whole)) < 24 && isSoulBoundaryUserOnly(whole) {
			return true, "soul_boundary"
		}
	}

	return false, ""
}

func isPlanEpisode(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(content, "[source=agenttest") ||
		strings.Contains(content, "## 用户诉求") ||
		strings.Contains(lower, "agenttest-plan")
}

func extractSection(content, heading string) string {
	marker := "## " + heading
	idx := strings.Index(content, marker)
	if idx < 0 {
		return ""
	}
	rest := content[idx+len(marker):]
	rest = strings.TrimPrefix(rest, "\n")
	if end := strings.Index(rest, "\n\n## "); end >= 0 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}

func isSoulBoundaryUserOnly(user string) bool {
	user = strings.TrimSpace(user)
	if user == "" {
		return false
	}
	for _, re := range soulBoundaryUserPatterns {
		if re.MatchString(user) {
			return true
		}
	}
	// 极短且无动作动词，多为寒暄/确认身份。
	if len([]rune(user)) <= 12 && !strings.ContainsAny(user, "列出写入执行调用保存打开读取运行") {
		if strings.Contains(user, "谁") || strings.Contains(user, "好") || strings.Contains(user, "谢") {
			return true
		}
	}
	return false
}

func hasExecutionSignals(content string) bool {
	lower := strings.ToLower(content)
	for _, m := range executionMarkers {
		if strings.Contains(lower, strings.ToLower(m)) {
			return true
		}
	}
	// 显式 MCP 工具名 pattern
	if strings.Contains(content, "__") && (strings.Contains(lower, "filesystem") ||
		strings.Contains(lower, "sqlite") || strings.Contains(lower, "resend") ||
		strings.Contains(lower, "figma") || strings.Contains(lower, "memory")) {
		return true
	}
	return false
}
