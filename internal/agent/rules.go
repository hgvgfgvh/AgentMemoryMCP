package agent

import (
	"regexp"
	"strings"
	"time"

	"fmt"

	"AgentTestMemoryMCP/internal/facts"
	"AgentTestMemoryMCP/internal/textutil"
)

var (
	reSection     = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)
	reToolsCalled = regexp.MustCompile(`tools_called:\s*(.+)`)
	reArtifacts   = regexp.MustCompile(`artifacts:\s*(.+)`)
	reStatusLine  = regexp.MustCompile(`(?m)^status:\s*(\w+)`)
	rePlanID      = regexp.MustCompile(`plan=([^\s\]]+)`)
	reTurnID      = regexp.MustCompile(`turn=([^\s\]]+)`)
)

// ExtractFromEpisode 规则抽取（无 LLM）：从 Host episode 文本生成事实点。
func ExtractFromEpisode(jobID, source, kind, correlationID, content string) []facts.Fact {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	pp := PreparseEpisode(content)
	userReq, portal, outcome := pp.UserReq, pp.Portal, pp.Outcome
	tools, artifacts, tags := pp.Tools, pp.Artifacts, pp.Tags
	summary := buildSummaryText(userReq, portal, outcome, tools, artifacts)
	if summary == "" {
		summary = preview(content, 400)
	}

	now := time.Now().UTC()
	conf := 0.85
	if outcome == "completed" || outcome == "success" {
		conf = 0.9
	}
	normOutcome := normalizeOutcome(outcome)
	weight := 1.0
	isPitfall := false
	if normOutcome == "fail" {
		isPitfall = true
		weight = 0.3
		conf = 0.75
	}
	f := facts.Fact{
		ID:            fmt.Sprintf("fact-%d-%s", now.UnixNano(), preview(jobID, 12)),
		EpisodeID:     jobID,
		Source:        source,
		CorrelationID: correlationID,
		Text:          summary,
		Tags:          tags,
		Outcome:       normOutcome,
		IsPitfall:     isPitfall,
		Tools:         tools,
		Artifacts:     artifacts,
		TierHint:      2,
		Confidence:    conf,
		Weight:        weight,
		CreatedAt:     now,
	}
	return []facts.Fact{f}
}

func splitSections(content string) map[string]string {
	out := map[string]string{}
	matches := reSection.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		out[""] = content
		return out
	}
	for i, loc := range matches {
		if len(loc) < 4 {
			continue
		}
		title := strings.TrimSpace(content[loc[2]:loc[3]])
		bodyStart := loc[1]
		bodyEnd := len(content)
		if i+1 < len(matches) {
			bodyEnd = matches[i+1][0]
		}
		out[title] = strings.TrimSpace(content[bodyStart:bodyEnd])
	}
	return out
}

func parseListLine(m []string) []string {
	if len(m) < 2 {
		return nil
	}
	raw := strings.TrimSpace(m[1])
	raw = strings.Trim(raw, "[]")
	var parts []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func deriveTags(text string) []string {
	keys := []string{"WorkSpace", "列出", "目录", "清单", "filesystem", "boundary_plan_dir_list"}
	var out []string
	seen := map[string]bool{}
	lower := strings.ToLower(text)
	for _, k := range keys {
		if strings.Contains(lower, strings.ToLower(k)) && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

// BuildSummaryFromPreparse 由预解析生成摘要文本。
func BuildSummaryFromPreparse(pp Preparse) string {
	return buildSummaryText(pp.UserReq, pp.Portal, pp.Outcome, pp.Tools, pp.Artifacts)
}

func buildSummaryText(userReq, portal, outcome string, tools, artifacts []string) string {
	var b strings.Builder
	if userReq != "" {
		b.WriteString("历史需求: ")
		b.WriteString(preview(userReq, 200))
		b.WriteString(". ")
	}
	if outcome != "" && outcome != "unknown" {
		b.WriteString("结果: ")
		b.WriteString(outcome)
		b.WriteString(". ")
	}
	if len(tools) > 0 {
		b.WriteString("工具: ")
		b.WriteString(strings.Join(tools, ", "))
		b.WriteString(". ")
	}
	if len(artifacts) > 0 {
		b.WriteString("产物: ")
		b.WriteString(strings.Join(artifacts, ", "))
		b.WriteString(". ")
	}
	if portal != "" {
		b.WriteString("摘要: ")
		b.WriteString(preview(portal, 160))
	}
	return strings.TrimSpace(b.String())
}

// NormalizeOutcome 规范化 outcome 字符串。
func NormalizeOutcome(s string) string {
	return normalizeOutcome(s)
}

func normalizeOutcome(s string) string {
	switch strings.ToLower(s) {
	case "completed", "success", "ok":
		return "success"
	case "failed", "fail", "blocked":
		return "fail"
	default:
		return "unknown"
	}
}

func preview(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "…"
}

// MatchScore 对单条 fact 相对 context 打分。
func MatchScore(context, queryHint string, f facts.Fact) float64 {
	doc := f.Text + " " + strings.Join(f.Tags, " ") + " " + strings.Join(f.Tools, " ")
	score := textutil.OverlapScore(context, doc)
	if queryHint != "" {
		score = score*0.6 + textutil.OverlapScore(queryHint, doc)*0.4
	}
	for _, t := range f.Tags {
		if textutil.ContainsAny(context, []string{t}) {
			score += 0.08
		}
	}
	if f.Outcome == "success" {
		score += 0.05
	}
	if score > 1 {
		score = 1
	}
	return score
}
