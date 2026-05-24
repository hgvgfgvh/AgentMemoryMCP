package memoryagent

import (
	"context"
	"log"
	"strings"
	"time"

	"AgentTestMemoryMCP/internal/agent"
	"AgentTestMemoryMCP/internal/facts"
	"AgentTestMemoryMCP/internal/llm"
)

// ProcessEpisode Store 异步链：S3–S10（2c：LLM 抽取 + L0/L1，失败回退 rules）。
func ProcessEpisode(ctx context.Context, jobID, source, kind, correlationID, content string, existing []facts.Fact) ProcessOutput {
	pp := agent.PreparseEpisode(content)
	if strings.TrimSpace(content) == "" {
		return ProcessOutput{}
	}

	tmpl, err := SelectTemplate(source)
	if err != nil {
		log.Printf("[memoryagent] template: %v", err)
	}

	var client *llm.Client
	if c, ok := llm.ConfigFromEnv(); ok {
		client = &c
	}
	tryLLM := LLMExtractEnabled() && client != nil

	if tryLLM && client != nil && tmpl != nil {
		extractCtx, cancel := context.WithTimeout(ctx, 50*time.Second)
		defer cancel()
		ex, err := LLMExtract(extractCtx, client, tmpl, jobID, source, content, pp, existing)
		if err == nil && ex != nil {
			ValidateL0(ex, pp)
			kept, dropped := ValidateL1(ex, content)
			out := MergeExtract(jobID, source, kind, correlationID, content, ex, pp)
			out.AtomsKept = kept
			out.AtomsDrop = dropped
			if len(out.Facts) > 0 {
				out = finalizeStoreOutput(ctx, client, content, out, existing, correlationID)
				return out
			}
		} else if err != nil {
			log.Printf("[memoryagent] llm extract fallback: %v", err)
		}
	}

	fs := agent.ExtractFromEpisode(jobID, source, kind, correlationID, content)
	out := ProcessOutput{Facts: fs, UsedLLM: false, Fallback: true}
	if len(out.Facts) > 0 {
		f := out.Facts[0]
		f.LastActive = f.CreatedAt
		out.Facts[0] = f
		out = finalizeStoreOutput(ctx, client, content, out, existing, correlationID)
	}
	return out
}

func finalizeStoreOutput(ctx context.Context, client *llm.Client, episodeContent string, out ProcessOutput, existing []facts.Fact, correlationID string) ProcessOutput {
	if len(out.Facts) == 0 {
		return out
	}
	f := out.Facts[0]
	f, out.SupersedeIDs, out.FuzzyPairs = EnrichAlignment(f, existing, correlationID, out.SupersedeIDs)
	out.Facts[0] = f
	ApplyL2Conflict(ctx, client, episodeContent, &out, existing, correlationID)
	return out
}
