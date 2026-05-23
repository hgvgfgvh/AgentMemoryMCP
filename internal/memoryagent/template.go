package memoryagent

import (
	"embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed templates/agenttest-plan.yaml
var embeddedTemplates embed.FS

// PlanTemplate 抽取模板。
type PlanTemplate struct {
	ID              string   `yaml:"id"`
	SourceMatch     string   `yaml:"source_match"`
	Description     string   `yaml:"description"`
	Sections        []string `yaml:"sections"`
	Predicates      []string `yaml:"predicates"`
	NodeTypes       []string `yaml:"node_types"`
	PitfallStatuses []string `yaml:"pitfall_statuses"`
	PromptRules     string   `yaml:"prompt_rules"`
}

// SelectTemplate 按 source 选择模板。
func SelectTemplate(source string) (*PlanTemplate, error) {
	source = strings.TrimSpace(source)
	b, err := embeddedTemplates.ReadFile("templates/agenttest-plan.yaml")
	if err != nil {
		return nil, err
	}
	var t PlanTemplate
	if err := yaml.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	if source == "" || source == t.SourceMatch || strings.HasPrefix(source, "agenttest") {
		return &t, nil
	}
	return &t, nil
}

func (t *PlanTemplate) SystemPrompt() string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf(`你是记忆系统结构化抽取器。仅输出合法 JSON，符合下列 schema：
{
  "summary": {"text":"","outcome":"success|failed|unknown","tools":[],"artifacts":[],"tags":[]},
  "atoms": [{"subject_type":"ENTITY|ACTION|STATE|CONCEPT","subject":"","predicate":"","object_type":"","object":"","confidence":0.0,"evidence":""}],
  "edges": [{"from":"node:...","to":"node:...","type":"","weight":0.8}],
  "supersede_fact_ids": [],
  "pitfall": false
}
允许 predicate: %s
允许 node_type: %s
规则:
%s`, strings.Join(t.Predicates, ", "), strings.Join(t.NodeTypes, ", "), strings.TrimSpace(t.PromptRules))
}
