package memoryagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"AgentTestMemoryMCP/internal/facts"
	"AgentTestMemoryMCP/internal/llm"
)

// L2Config 语义冲突 L2 参数。
type L2Config struct {
	MinOldWeight     float64
	MaxCandidates    int
	MinTagJaccard    float64
	SkipIfSimilarGTE float64
	Timeout          time.Duration
	ConfidenceDrop   float64
	WeightDrop       float64
}

// DefaultL2Config 从环境变量读取。
func DefaultL2Config() L2Config {
	cfg := L2Config{
		MinOldWeight:     0.5,
		MaxCandidates:    3,
		MinTagJaccard:    0.3,
		SkipIfSimilarGTE: 0.92,
		Timeout:          15 * time.Second,
		ConfidenceDrop:   0.1,
		WeightDrop:       0.05,
	}
	if v := os.Getenv("MEMORY_MCP_L2_MIN_OLD_WEIGHT"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			cfg.MinOldWeight = f
		}
	}
	if v := os.Getenv("MEMORY_MCP_L2_MAX_CANDIDATES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxCandidates = n
		}
	}
	if v := os.Getenv("MEMORY_MCP_L2_TOPIC_JACCARD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			cfg.MinTagJaccard = f
		}
	}
	if v := os.Getenv("MEMORY_MCP_L2_TIMEOUT_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Timeout = time.Duration(n) * time.Millisecond
		}
	}
	return cfg
}

// L2ConflictEnabled 是否启用 Store 语义冲突 L2。
func L2ConflictEnabled() bool {
	v := strings.TrimSpace(os.Getenv("MEMORY_MCP_L2_CONFLICT"))
	if v == "" {
		return true
	}
	return v == "1" || strings.EqualFold(v, "true")
}

// l2DecisionRow LLM 裁决单行。
type l2DecisionRow struct {
	OldFactID string `json:"old_fact_id"`
	Choice    string `json:"choice"`
	Reason    string `json:"reason"`
}

type l2Result struct {
	Decisions []l2DecisionRow `json:"decisions"`
}

const l2SystemPrompt = `你是记忆库语义冲突裁决器。给定「新事实」与若干「旧事实」，判断新事实与每条旧事实是否语义矛盾。
仅输出合法 JSON：
{"decisions":[{"old_fact_id":"...","choice":"A|B|C","reason":"简短中文"}]}

choice 含义：
- A：新事实更新旧事实（旧事实应被取代）
- B：新事实不成立，应忽略本次写入
- C：无法判定或二者可共存，均保留但新事实置信度应降低

规则：
- 同一工具/任务，旧 success 新 failed → 通常 A 或 C（若新 episode 证据更强则 A）
- 旧 failed 新 success → 通常 A（新成功覆盖旧失败经验）
- 不确定时选 C
- 每条 old_fact_id 必须给出且 choice 仅为 A、B、C 之一`

// ApplyL2Conflict 在 EnrichAlignment 之后调用：检测候选 → 可选 LLM → 写回 ProcessOutput。
func ApplyL2Conflict(ctx context.Context, client *llm.Client, episodeContent string, out *ProcessOutput, existing []facts.Fact, correlationID string) {
	if out == nil || len(out.Facts) == 0 || !L2ConflictEnabled() {
		return
	}
	cfg := DefaultL2Config()
	cands := DetectConflictCandidates(out.Facts[0], existing, correlationID, cfg)
	if len(cands) == 0 {
		return
	}

	decisions := defaultL2Decisions(cands, "C")
	if client != nil {
		l2Ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
		if resolved, err := resolveL2WithLLM(l2Ctx, client, episodeContent, out.Facts[0], cands); err == nil && len(resolved) > 0 {
			decisions = resolved
		} else if err != nil {
			log.Printf("[memoryagent] L2 llm fallback coexist: %v", err)
		}
	} else {
		log.Printf("[memoryagent] L2: no llm client, default C for %d candidate(s)", len(cands))
	}

	applyL2Decisions(out, decisions, cfg)
	out.L2Applied = true
	out.L2CandidateCount = len(cands)
}

func defaultL2Decisions(cands []ConflictCandidate, choice string) []l2DecisionRow {
	rows := make([]l2DecisionRow, 0, len(cands))
	for _, c := range cands {
		rows = append(rows, l2DecisionRow{OldFactID: c.Old.ID, Choice: choice, Reason: "default"})
	}
	return rows
}

func resolveL2WithLLM(ctx context.Context, client *llm.Client, episode string, newF facts.Fact, cands []ConflictCandidate) ([]l2DecisionRow, error) {
	user := buildL2UserPrompt(episode, newF, cands)
	raw, err := client.ChatJSON(ctx, l2SystemPrompt, user)
	if err != nil {
		return nil, err
	}
	raw = extractJSON(raw)
	var res l2Result
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return nil, fmt.Errorf("l2 json: %w", err)
	}
	valid := map[string]bool{}
	for _, c := range cands {
		valid[c.Old.ID] = true
	}
	var out []l2DecisionRow
	for _, d := range res.Decisions {
		id := strings.TrimSpace(d.OldFactID)
		if !valid[id] {
			continue
		}
		ch := strings.ToUpper(strings.TrimSpace(d.Choice))
		if ch != "A" && ch != "B" && ch != "C" {
			ch = "C"
		}
		out = append(out, l2DecisionRow{OldFactID: id, Choice: ch, Reason: d.Reason})
	}
	// 补全未裁决候选 → C
	seen := map[string]bool{}
	for _, d := range out {
		seen[d.OldFactID] = true
	}
	for _, c := range cands {
		if !seen[c.Old.ID] {
			out = append(out, l2DecisionRow{OldFactID: c.Old.ID, Choice: "C", Reason: "missing"})
		}
	}
	return out, nil
}

func buildL2UserPrompt(episode string, newF facts.Fact, cands []ConflictCandidate) string {
	var b strings.Builder
	b.WriteString("## 新事实\n")
	b.WriteString(fmt.Sprintf("id=%s outcome=%s pitfall=%v tools=%s tags=%s\n",
		newF.ID, newF.Outcome, newF.IsPitfall, strings.Join(newF.Tools, ","), strings.Join(newF.Tags, ",")))
	b.WriteString(truncate(newF.Text, 400))
	b.WriteString("\n\n## 冲突旧事实\n")
	for i, c := range cands {
		o := c.Old
		b.WriteString(fmt.Sprintf("%d) id=%s weight=%.2f outcome=%s pitfall=%v tools=%s\n",
			i+1, o.ID, o.Weight, o.Outcome, o.IsPitfall, strings.Join(o.Tools, ",")))
		b.WriteString(truncate(o.Text, 300))
		b.WriteString("\n")
	}
	b.WriteString("\n## Episode 摘录\n")
	b.WriteString(truncate(episode, 1200))
	return b.String()
}

func applyL2Decisions(out *ProcessOutput, decisions []l2DecisionRow, cfg L2Config) {
	if out == nil || len(out.Facts) == 0 {
		return
	}
	seenSup := map[string]bool{}
	for _, id := range out.SupersedeIDs {
		seenSup[id] = true
	}

	for _, d := range decisions {
		ch := strings.ToUpper(strings.TrimSpace(d.Choice))
		switch ch {
		case "B":
			out.SkipNewFact = true
			out.Facts = nil
			out.Atoms = nil
			out.L2DropNew = true
			return
		case "A":
			if d.OldFactID != "" && !seenSup[d.OldFactID] {
				seenSup[d.OldFactID] = true
				out.SupersedeIDs = append(out.SupersedeIDs, d.OldFactID)
			}
		case "C":
			f := out.Facts[0]
			f.Confidence -= cfg.ConfidenceDrop
			if f.Confidence < 0.5 {
				f.Confidence = 0.5
			}
			f.Weight -= cfg.WeightDrop
			if f.Weight < 0.2 {
				f.Weight = 0.2
			}
			out.Facts[0] = f
		}
	}
}
