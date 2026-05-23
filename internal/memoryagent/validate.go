package memoryagent

import (
	"strings"

	"AgentTestMemoryMCP/internal/agent"
)

// ValidateL0 tools/artifacts/outcome 不得超出 preparse（剔除越界项）。
func ValidateL0(ex *ExtractResult, pp agent.Preparse) {
	if ex == nil {
		return
	}
	ex.Summary.Tools = intersectList(ex.Summary.Tools, pp.Tools)
	ex.Summary.Artifacts = intersectList(ex.Summary.Artifacts, pp.Artifacts)
	if !outcomeAllowed(ex.Summary.Outcome, pp.Outcome) {
		ex.Summary.Outcome = agent.NormalizeOutcome(pp.Outcome)
	}
	for i := range ex.Atoms {
		// atoms 不携带 tools 列表；predicate 在 merge 时校验
		_ = i
	}
}

// ValidateL1 过滤 evidence 未通过 Fuzzy 锚定的 atoms。
func ValidateL1(ex *ExtractResult, episode string) (kept, dropped int) {
	if ex == nil {
		return 0, 0
	}
	var ok []AtomExtract
	for _, a := range ex.Atoms {
		if strings.TrimSpace(a.Evidence) == "" {
			dropped++
			continue
		}
		if !EvidenceAnchored(a.Evidence, episode, 0) {
			dropped++
			continue
		}
		ok = append(ok, a)
	}
	ex.Atoms = ok
	return len(ok), dropped
}

func intersectList(got, allowed []string) []string {
	if len(allowed) == 0 {
		return nil
	}
	set := map[string]bool{}
	for _, a := range allowed {
		set[strings.TrimSpace(a)] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, g := range got {
		g = strings.TrimSpace(g)
		if g == "" || !set[g] || seen[g] {
			continue
		}
		seen[g] = true
		out = append(out, g)
	}
	return out
}

func outcomeAllowed(llmOutcome, parsed string) bool {
	llmOutcome = strings.ToLower(strings.TrimSpace(llmOutcome))
	parsed = strings.ToLower(strings.TrimSpace(parsed))
	if llmOutcome == "" || llmOutcome == "unknown" {
		return true
	}
	if parsed == "unknown" {
		return true
	}
	return agent.NormalizeOutcome(llmOutcome) == agent.NormalizeOutcome(parsed)
}
