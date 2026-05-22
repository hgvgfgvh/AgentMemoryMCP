package engine

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed fixture_completed_todolist.json
var embeddedFixtureJSON []byte

// FixturePlan 测试用已完成 TodoList 快照（仅测试引擎使用，非 MCP 协议 DTO）。
type FixturePlan struct {
	ID              string        `json:"id"`
	UserRequirement string        `json:"user_requirement"`
	Summary         string        `json:"summary"`
	Status          string        `json:"status"`
	ExecutionMode   string        `json:"execution_mode,omitempty"`
	Steps           []FixtureStep `json:"steps"`
	MatchKeywords   []string      `json:"-"`
}

type FixtureStep struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Instruction     string   `json:"instruction"`
	CapabilityHints []string `json:"capability_hints,omitempty"`
	Tier            int      `json:"tier,omitempty"`
	Status          string   `json:"status"`
	ResultSummary   string   `json:"result_summary,omitempty"`
	Artifacts       []string `json:"artifacts,omitempty"`
	ToolsCalled     []string `json:"tools_called,omitempty"`
}

func loadFixturePlan() (FixturePlan, error) {
	var raw struct {
		ID              string        `json:"id"`
		UserRequirement string        `json:"user_requirement"`
		Summary         string        `json:"summary"`
		Status          string        `json:"status"`
		ExecutionMode   string        `json:"execution_mode"`
		Steps           []FixtureStep `json:"steps"`
	}
	if err := json.Unmarshal(embeddedFixtureJSON, &raw); err != nil {
		return FixturePlan{}, err
	}
	fp := FixturePlan{
		ID:              raw.ID,
		UserRequirement: raw.UserRequirement,
		Summary:         raw.Summary,
		Status:          raw.Status,
		ExecutionMode:   raw.ExecutionMode,
		Steps:           raw.Steps,
		MatchKeywords:   deriveMatchKeywords(raw.UserRequirement),
	}
	return fp, nil
}

// deriveMatchKeywords 从需求文本提取若干子串用于测试命中（非用户原文正则路由）。
func deriveMatchKeywords(requirement string) []string {
	req := strings.TrimSpace(requirement)
	if req == "" {
		return nil
	}
	keys := []string{"WorkSpace", "列出", "目录", "清单"}
	var out []string
	seen := map[string]bool{}
	for _, k := range keys {
		if strings.Contains(req, k) && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	if len(out) < 2 {
		// 回退：按标点切词
		for _, part := range strings.FieldsFunc(req, func(r rune) bool {
			return r == ' ' || r == '，' || r == '。' || r == ',' || r == '.' || r == '、'
		}) {
			part = trim(part)
			if len([]rune(part)) >= 2 && !seen[part] {
				seen[part] = true
				out = append(out, part)
			}
		}
	}
	return out
}

func contextMatchesFixture(context string, fp FixturePlan) bool {
	ctx := strings.TrimSpace(context)
	if ctx == "" {
		return false
	}
	if strings.Contains(ctx, fp.UserRequirement) {
		return true
	}
	hit := 0
	for _, k := range fp.MatchKeywords {
		if strings.Contains(ctx, k) {
			hit++
		}
	}
	return hit >= 2
}
