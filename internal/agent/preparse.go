package agent

import "strings"

// Preparse 规则预解析结果（S4，无 LLM）。
type Preparse struct {
	UserReq   string
	Portal    string
	PlanBlock string
	Outcome   string
	Tools     []string
	Artifacts []string
	Tags      []string
}

// PreparseEpisode 从 episode Markdown 提取结构化字段。
func PreparseEpisode(content string) Preparse {
	content = strings.TrimSpace(content)
	sections := splitSections(content)
	userReq := strings.TrimSpace(sections["用户诉求"])
	portal := strings.TrimSpace(sections["门户回复"])
	planBlock := sections["计划终态 (TodoList)"]
	if planBlock == "" {
		planBlock = sections["计划终态"]
	}
	outcome := "unknown"
	if m := reStatusLine.FindStringSubmatch(planBlock + "\n" + content); len(m) >= 2 {
		outcome = strings.ToLower(strings.TrimSpace(m[1]))
	}
	tools := parseListLine(reToolsCalled.FindStringSubmatch(planBlock))
	artifacts := parseListLine(reArtifacts.FindStringSubmatch(planBlock))
	tags := deriveTags(userReq + " " + portal + " " + planBlock)
	return Preparse{
		UserReq: userReq, Portal: portal, PlanBlock: planBlock,
		Outcome: outcome, Tools: tools, Artifacts: artifacts, Tags: tags,
	}
}
