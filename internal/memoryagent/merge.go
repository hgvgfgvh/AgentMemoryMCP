package memoryagent

import (
	"fmt"
	"strings"
	"time"

	"AgentTestMemoryMCP/internal/agent"
	"AgentTestMemoryMCP/internal/facts"
)

var allowedPredicates = map[string]bool{
	"uses": true, "triggers": true, "caused_by": true, "depends_on": true,
	"produces": true, "supersedes": true, "pitfall": true, "similar": true,
}

// MergeExtract 将校验后的抽取结果转为 Fact + StoredAtom。
func MergeExtract(jobID, source, kind, correlationID, content string, ex *ExtractResult, pp agent.Preparse) ProcessOutput {
	now := time.Now().UTC()
	outcome := agent.NormalizeOutcome(ex.Summary.Outcome)
	if outcome == "unknown" {
		outcome = agent.NormalizeOutcome(pp.Outcome)
	}
	isPitfall := ex.Pitfall || outcome == "fail"
	weight := 1.0
	conf := 0.88
	if isPitfall {
		weight = 0.3
		conf = 0.75
	} else if outcome == "success" {
		conf = 0.92
	}
	tools := ex.Summary.Tools
	if len(tools) == 0 {
		tools = pp.Tools
	}
	artifacts := ex.Summary.Artifacts
	if len(artifacts) == 0 {
		artifacts = pp.Artifacts
	}
	tags := ex.Summary.Tags
	if len(tags) == 0 {
		tags = pp.Tags
	}
	text := strings.TrimSpace(ex.Summary.Text)
	if text == "" {
		text = agent.BuildSummaryFromPreparse(pp)
	}
	f := facts.Fact{
		ID:            fmt.Sprintf("fact-%d-%s", now.UnixNano(), jobID[len(jobID)-min(8, len(jobID)):]),
		EpisodeID:     jobID,
		Source:        source,
		CorrelationID: correlationID,
		Text:          text,
		Tags:          tags,
		Outcome:       outcome,
		IsPitfall:     isPitfall,
		Tools:         tools,
		Artifacts:     artifacts,
		TierHint:      2,
		Confidence:    conf,
		Weight:        weight,
		CreatedAt:     now,
	}
	var atoms []StoredAtom
	for i, a := range ex.Atoms {
		if !allowedPredicates[strings.ToLower(strings.TrimSpace(a.Predicate))] {
			continue
		}
		atoms = append(atoms, StoredAtom{
			ID:          fmt.Sprintf("atom-%d-%d", now.UnixNano(), i),
			EpisodeID:   jobID,
			SubjectType: strings.TrimSpace(a.SubjectType),
			Subject:     strings.TrimSpace(a.Subject),
			Predicate:   strings.TrimSpace(a.Predicate),
			ObjectType:  strings.TrimSpace(a.ObjectType),
			Object:      strings.TrimSpace(a.Object),
			Evidence:    truncate(strings.TrimSpace(a.Evidence), 300),
			Confidence:  a.Confidence,
		})
	}
	_ = kind
	_ = content
	return ProcessOutput{Facts: []facts.Fact{f}, Atoms: atoms, UsedLLM: true}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
