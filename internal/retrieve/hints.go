package retrieve

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"AgentTestMemoryMCP/internal/agent"
	"AgentTestMemoryMCP/internal/facts"
)

const memoryRouteMarker = "---memory-route---"

// ScoredFact 检索候选。
type ScoredFact struct {
	Fact  facts.Fact
	Score float64
}

// Search 对全部 facts 打分并取 TopK。
func Search(all []facts.Fact, context, queryHint string, topK int, minScore float64) []ScoredFact {
	var scored []ScoredFact
	for _, f := range all {
		if f.Weight < 0.1 || f.Superseded {
			continue
		}
		s := agent.MatchScore(context, queryHint, f)
		if s >= minScore {
			scored = append(scored, ScoredFact{Fact: f, Score: s})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	if topK > 0 && len(scored) > topK {
		scored = scored[:topK]
	}
	return scored
}

// BuildHints 组装自然语言 hints + memory-route JSON 块。
func BuildHints(scored []ScoredFact, routeThreshold float64, contextStr string) string {
	if len(scored) == 0 {
		return "【跨会话事实参考】\n(无匹配事实；事实库为空或未达阈值。)\n"
	}
	var b strings.Builder
	b.WriteString("【跨会话事实参考】\n")
	b.WriteString(fmt.Sprintf("(factworld: 命中 %d 条)\n", len(scored)))
	top := scored[0]
	b.WriteString("【推荐经验】\n")
	b.WriteString(top.Fact.Text)
	b.WriteString("\n")
	if len(top.Fact.Tools) > 0 {
		b.WriteString("tools: " + strings.Join(top.Fact.Tools, ", ") + "\n")
	}
	if len(top.Fact.Artifacts) > 0 {
		b.WriteString("artifacts: " + strings.Join(top.Fact.Artifacts, ", ") + "\n")
	}
	if len(top.Fact.Tags) > 0 {
		b.WriteString("tags: " + strings.Join(top.Fact.Tags, ", ") + "\n")
	}
	for i := 1; i < len(scored) && i < 3; i++ {
		b.WriteString(fmt.Sprintf("- 其它相关[%d] score=%.2f: %s\n", i+1, scored[i].Score, preview(scored[i].Fact.Text, 120)))
	}

	match := "no"
	conf := top.Score
	if pitfallBlocksRoute(scored, contextStr) {
		match = "no"
	} else {
		out := strings.ToLower(top.Fact.Outcome)
		if top.Score >= routeThreshold && (out == "success" || out == "completed") && !top.Fact.IsPitfall {
			match = "yes"
			if top.Fact.Confidence > conf {
				conf = top.Fact.Confidence
			}
		}
	}
	route := map[string]any{
		"exec_simple_match": match,
		"confidence":        conf,
		"fact_ids":          []string{top.Fact.ID},
	}
	rb, _ := json.Marshal(route)
	b.WriteString("\n")
	b.WriteString(memoryRouteMarker)
	b.WriteString("\n")
	b.Write(rb)
	// 兼容旧 Host 正则
	if match == "yes" {
		b.WriteString(fmt.Sprintf("\n[exec_simple_match=yes confidence=%.2f]", conf))
	}
	return b.String()
}

func pitfallBlocksRoute(scored []ScoredFact, contextStr string) bool {
	for _, s := range scored {
		if !s.Fact.IsPitfall && s.Fact.Outcome != "fail" && s.Fact.Outcome != "failed" {
			continue
		}
		if agent.MatchScore(contextStr, "", s.Fact) >= 0.35 {
			return true
		}
	}
	return false
}

func preview(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "…"
}

// ParseRouteBlock 从 hints 解析 memory-route（供测试）。
func ParseRouteBlock(hints string) (match string, confidence float64, ok bool) {
	idx := strings.Index(hints, memoryRouteMarker)
	if idx < 0 {
		return "", 0, false
	}
	rest := hints[idx+len(memoryRouteMarker):]
	start := strings.Index(rest, "{")
	end := strings.LastIndex(rest, "}")
	if start < 0 || end <= start {
		return "", 0, false
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(rest[start:end+1]), &m); err != nil {
		return "", 0, false
	}
	if v, _ := m["exec_simple_match"].(string); v != "" {
		match = v
	}
	switch c := m["confidence"].(type) {
	case float64:
		confidence = c
	}
	return match, confidence, true
}
