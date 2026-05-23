package memoryagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"AgentTestMemoryMCP/internal/agent"
	"AgentTestMemoryMCP/internal/facts"
	"AgentTestMemoryMCP/internal/llm"
)

// LLMExtract 调用 LLM 做 S5 结构化抽取。
func LLMExtract(ctx context.Context, client *llm.Client, tmpl *PlanTemplate, jobID, source, content string, pp agent.Preparse, existing []facts.Fact) (*ExtractResult, error) {
	if client == nil || tmpl == nil {
		return nil, fmt.Errorf("llm extract: missing client or template")
	}
	user := buildExtractUserPrompt(jobID, source, content, pp, existing)
	raw, err := client.ChatJSON(ctx, tmpl.SystemPrompt(), user)
	if err != nil {
		return nil, err
	}
	raw = extractJSON(raw)
	var ex ExtractResult
	if err := json.Unmarshal([]byte(raw), &ex); err != nil {
		return nil, fmt.Errorf("parse extract json: %w", err)
	}
	if strings.TrimSpace(ex.Summary.Text) == "" {
		return nil, fmt.Errorf("extract: empty summary.text")
	}
	return &ex, nil
}

func buildExtractUserPrompt(jobID, source, content string, pp agent.Preparse, existing []facts.Fact) string {
	var b strings.Builder
	b.WriteString("episode_id: " + jobID + "\nsource: " + source + "\n\n")
	b.WriteString("## 预解析（不得超出）\n")
	b.WriteString("outcome: " + pp.Outcome + "\n")
	b.WriteString("tools: " + strings.Join(pp.Tools, ", ") + "\n")
	b.WriteString("artifacts: " + strings.Join(pp.Artifacts, ", ") + "\n")
	b.WriteString("tags: " + strings.Join(pp.Tags, ", ") + "\n\n")
	if len(existing) > 0 {
		b.WriteString("## 已有事实摘要（Top 3）\n")
		n := 3
		if len(existing) < n {
			n = len(existing)
		}
		for i := len(existing) - n; i < len(existing); i++ {
			f := existing[i]
			b.WriteString(fmt.Sprintf("- id=%s outcome=%s text=%s\n", f.ID, f.Outcome, truncate(f.Text, 200)))
		}
		b.WriteString("\n")
	}
	b.WriteString("## Episode 原文\n")
	b.WriteString(content)
	return b.String()
}

func extractJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		if i := strings.LastIndex(raw, "```"); i >= 0 {
			raw = raw[:i]
		}
		raw = strings.TrimSpace(raw)
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
