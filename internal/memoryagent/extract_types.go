package memoryagent

import (
	"AgentTestMemoryMCP/internal/entity"
	"AgentTestMemoryMCP/internal/facts"
)

// ExtractResult LLM 结构化抽取输出（S5）。
type ExtractResult struct {
	Summary          SummaryExtract `json:"summary"`
	Atoms            []AtomExtract  `json:"atoms"`
	Edges            []EdgeExtract  `json:"edges"`
	SupersedeFactIDs []string       `json:"supersede_fact_ids"`
	Pitfall          bool           `json:"pitfall"`
}

// SummaryExtract Host hints 主摘要。
type SummaryExtract struct {
	Text      string   `json:"text"`
	Outcome   string   `json:"outcome"`
	Tools     []string `json:"tools"`
	Artifacts []string `json:"artifacts"`
	Tags      []string `json:"tags"`
}

// AtomExtract 原子三元组。
type AtomExtract struct {
	SubjectType string  `json:"subject_type"`
	Subject     string  `json:"subject"`
	Predicate   string  `json:"predicate"`
	ObjectType  string  `json:"object_type"`
	Object      string  `json:"object"`
	Confidence  float64 `json:"confidence"`
	Evidence    string  `json:"evidence"`
}

// EdgeExtract 提议边（可选，与 derive 合并）。
type EdgeExtract struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Type   string  `json:"type"`
	Weight float64 `json:"weight"`
}

// ProcessOutput Store 流水线最终产物。
type ProcessOutput struct {
	Facts        []facts.Fact
	Atoms        []StoredAtom
	UsedLLM      bool
	Fallback     bool
	AtomsKept    int
	AtomsDrop    int
	SupersedeIDs []string
	FuzzyPairs   []entity.FuzzyPair
}

// StoredAtom 持久化原子记录。
type StoredAtom struct {
	ID          string  `json:"id"`
	EpisodeID   string  `json:"episode_id"`
	SubjectType string  `json:"subject_type"`
	Subject     string  `json:"subject"`
	Predicate   string  `json:"predicate"`
	ObjectType  string  `json:"object_type"`
	Object      string  `json:"object"`
	Evidence    string  `json:"evidence"`
	Confidence  float64 `json:"confidence"`
}
