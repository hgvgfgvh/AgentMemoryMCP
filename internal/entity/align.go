package entity

import (
	"os"
	"strconv"
	"strings"

	"AgentTestMemoryMCP/internal/embedding"
	"AgentTestMemoryMCP/internal/facts"
)

// FuzzyPair 模糊带候选（0.85~0.92），供异步 LLM 对齐。
type FuzzyPair struct {
	NewFactID string
	OldFactID string
	Score     float64
}

// AlignConfig 阈值。
type AlignConfig struct {
	HardMerge float64
	FuzzyLow  float64
}

// DefaultAlignConfig 从环境变量读取，默认 0.92 / 0.85。
func DefaultAlignConfig() AlignConfig {
	cfg := AlignConfig{HardMerge: 0.92, FuzzyLow: 0.85}
	if v := os.Getenv("MEMORY_MCP_ENTITY_MERGE_COSINE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			cfg.HardMerge = f
		}
	}
	if v := os.Getenv("MEMORY_MCP_ENTITY_FUZZY_LOW"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			cfg.FuzzyLow = f
		}
	}
	return cfg
}

// FactSignature 用于相似度比较的签名文本。
func FactSignature(f facts.Fact) string {
	var b strings.Builder
	b.WriteString(f.Text)
	b.WriteString(" ")
	b.WriteString(strings.Join(f.Tags, " "))
	b.WriteString(" ")
	b.WriteString(strings.Join(f.Tools, " "))
	b.WriteString(" ")
	b.WriteString(strings.Join(f.Artifacts, " "))
	return b.String()
}

// CompareFacts 返回 hard supersede 与 fuzzy 候选（不含同 correlation 已由 Replace 处理）。
func CompareFacts(newFact facts.Fact, existing []facts.Fact, sameCorrelation string, cfg AlignConfig) (supersede []string, fuzzy []FuzzyPair) {
	if cfg.HardMerge <= 0 {
		cfg = DefaultAlignConfig()
	}
	if cfg.FuzzyLow <= 0 {
		cfg.FuzzyLow = 0.85
	}
	newBag := embedding.BagFromText(FactSignature(newFact))
	if len(newBag) == 0 {
		return nil, nil
	}
	for _, old := range existing {
		if old.ID == newFact.ID {
			continue
		}
		if sameCorrelation != "" && old.CorrelationID == sameCorrelation {
			continue
		}
		if old.Weight < 0.05 || old.Superseded {
			continue
		}
		score := embedding.Cosine(newBag, embedding.BagFromText(FactSignature(old)))
		if score >= cfg.HardMerge {
			supersede = append(supersede, old.ID)
			continue
		}
		if score >= cfg.FuzzyLow && score < cfg.HardMerge {
			fuzzy = append(fuzzy, FuzzyPair{NewFactID: newFact.ID, OldFactID: old.ID, Score: score})
		}
	}
	return supersede, fuzzy
}

// ToolCosine 比较两个工具名（硬规范化后词袋）。
func ToolCosine(a, b string) float64 {
	ka := embedding.NormalizeEntityKey(a)
	kb := embedding.NormalizeEntityKey(b)
	if ka == kb && ka != "" {
		return 1
	}
	return embedding.Cosine(embedding.BagFromText(a), embedding.BagFromText(b))
}
