package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"AgentTestMemoryMCP/internal/align"
	"AgentTestMemoryMCP/internal/atoms"
	"AgentTestMemoryMCP/internal/degenerate"
	"AgentTestMemoryMCP/internal/facts"
	"AgentTestMemoryMCP/internal/filter"
	"AgentTestMemoryMCP/internal/graph"
	"AgentTestMemoryMCP/internal/memoryagent"
	"AgentTestMemoryMCP/internal/response"
	"AgentTestMemoryMCP/internal/retrieve"
)

// FactWorldEngine Phase-2b：规则抽取 + 持久图 + BFS + BM25 检索。
type FactWorldEngine struct {
	dataDir          string
	repo             *facts.Repo
	jobSeq           atomic.Uint64
	wg               sync.WaitGroup
	routeThreshold   float64
	retrieveTopK     int
	retrieveMinScore float64
}

// FactWorldConfig 可选调参。
type FactWorldConfig struct {
	RouteThreshold   float64
	RetrieveTopK     int
	RetrieveMinScore float64
}

func NewFactWorldEngine(dataDir string, cfg FactWorldConfig) (*FactWorldEngine, error) {
	if dataDir == "" {
		dataDir = "data"
	}
	if err := facts.EnsureDataDirs(dataDir); err != nil {
		return nil, err
	}
	repo, err := facts.NewRepo(dataDir)
	if err != nil {
		return nil, err
	}
	if cfg.RouteThreshold <= 0 {
		cfg.RouteThreshold = 0.75
	}
	if cfg.RetrieveTopK <= 0 {
		cfg.RetrieveTopK = 5
	}
	if cfg.RetrieveMinScore <= 0 {
		cfg.RetrieveMinScore = 0.35
	}
	return &FactWorldEngine{
		dataDir:          dataDir,
		repo:             repo,
		routeThreshold:   cfg.RouteThreshold,
		retrieveTopK:     cfg.RetrieveTopK,
		retrieveMinScore: cfg.RetrieveMinScore,
	}, nil
}

func (e *FactWorldEngine) Store(ctx context.Context, in StoreInput) string {
	_ = ctx
	if skip, reason := filter.ShouldSkipStore(in.Content); skip {
		return response.FormatStore(response.StorePayload{
			Accepted:   "false",
			Skipped:    "true",
			SkipReason: reason,
			Message:    "store skipped by filter",
			Phase:      response.PhaseFactWorld(),
		})
	}
	jobID := fmt.Sprintf("job-%d-%d", time.Now().Unix(), e.jobSeq.Add(1))
	if err := e.writeEpisode(jobID, in); err != nil {
		return response.FormatStore(response.StorePayload{
			Accepted: "false",
			Skipped:  "false",
			Message:  "store failed: " + err.Error(),
			Phase:    response.PhaseFactWorld(),
		})
	}
	if err := e.enqueueJob(jobID, in); err != nil {
		return response.FormatStore(response.StorePayload{
			Accepted: "false",
			Skipped:  "false",
			Message:  "enqueue failed: " + err.Error(),
			Phase:    response.PhaseFactWorld(),
		})
	}
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		if err := e.processJob(jobID, in); err != nil {
			log.Printf("[factworld] job %s failed: %v", jobID, err)
			_ = e.moveJob(jobID, "dead")
		} else {
			_ = e.moveJob(jobID, "done")
		}
	}()
	return response.FormatStore(response.StorePayload{
		Accepted: "true",
		JobID:    jobID,
		Skipped:  "false",
		Message:  "accepted; async fact extraction queued",
		Phase:    response.PhaseFactWorld(),
	})
}

func (e *FactWorldEngine) Retrieve(ctx context.Context, in RetrieveInput) string {
	if skip, reason := filter.ShouldSkipRetrieve(in.Context); skip {
		return response.FormatRetrieve(response.RetrievePayload{
			Hints:      "",
			Skipped:    "true",
			SkipReason: reason,
			Phase:      response.PhaseFactWorld(),
		})
	}
	all, err := e.repo.List()
	if err != nil {
		return response.FormatRetrieve(response.RetrievePayload{
			Hints:   "【跨会话事实参考】\n(retrieve error: " + err.Error() + ")\n",
			Skipped: "false",
			Phase:   response.PhaseFactWorld(),
		})
	}
	dcfg := degenerate.DefaultConfig()
	all = degenerate.ApplyStaleDecay(all, time.Now().UTC(), dcfg)
	ctx2, cancel := context.WithTimeout(ctx, retrieveBudget())
	defer cancel()

	var scored []retrieve.ScoredFact
	if useLegacyRetrieve() {
		scored = retrieve.Search(all, in.Context, in.QueryHint, e.retrieveTopK, e.retrieveMinScore)
	} else {
		mg, gerr := graph.LoadOrDerive(e.dataDir, all)
		if gerr != nil {
			scored = retrieve.Search(all, in.Context, in.QueryHint, e.retrieveTopK, e.retrieveMinScore)
		} else {
			pcfg := retrieve.DefaultPipelineConfig(e.routeThreshold, e.retrieveTopK, e.retrieveMinScore)
			scored = retrieve.SearchWithGraph(ctx2, all, mg, in.Context, in.QueryHint, pcfg)
			if len(scored) == 0 {
				scored = retrieve.Search(all, in.Context, in.QueryHint, e.retrieveTopK, e.retrieveMinScore)
			}
		}
	}
	hitIDs := make([]string, 0, len(scored))
	for _, s := range scored {
		hitIDs = append(hitIDs, s.Fact.ID)
	}
	if len(hitIDs) > 0 {
		all = degenerate.TouchRetrieve(all, hitIDs, time.Now().UTC(), dcfg)
		if err := e.repo.Rewrite(all); err != nil {
			log.Printf("[factworld] touch retrieve: %v", err)
		}
	}
	hints := retrieve.BuildHints(scored, e.routeThreshold, in.Context)
	return response.FormatRetrieve(response.RetrievePayload{
		Hints:   hints,
		Skipped: "false",
		Phase:   response.PhaseFactWorld(),
	})
}

func (e *FactWorldEngine) writeEpisode(jobID string, in StoreInput) error {
	day := time.Now().Format("2006-01-02")
	dir := filepath.Join(e.dataDir, "episodes", day)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, jobID+".md")
	body := fmt.Sprintf("# episode %s\n\nsource: %s\nkind: %s\ncorrelation_id: %s\n\n%s\n",
		jobID, in.Source, in.Kind, in.CorrelationID, in.Content)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	// 审计镜像
	audit := filepath.Join(e.dataDir, "store_log", fmt.Sprintf("store_%s.md", time.Now().Format("20060102_150405")))
	_ = os.WriteFile(audit, []byte(body), 0o644)
	return nil
}

type pendingJob struct {
	JobID         string `json:"job_id"`
	Source        string `json:"source"`
	Kind          string `json:"kind"`
	CorrelationID string `json:"correlation_id"`
	Content       string `json:"content"`
}

func (e *FactWorldEngine) enqueueJob(jobID string, in StoreInput) error {
	p := pendingJob{
		JobID: jobID, Source: in.Source, Kind: in.Kind,
		CorrelationID: in.CorrelationID, Content: in.Content,
	}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	path := filepath.Join(e.dataDir, "jobs", "pending", jobID+".json")
	return os.WriteFile(path, b, 0o644)
}

func (e *FactWorldEngine) processJob(jobID string, in StoreInput) error {
	ctx := context.Background()
	existing, _ := e.repo.List()
	out := memoryagent.ProcessEpisode(ctx, jobID, in.Source, in.Kind, in.CorrelationID, in.Content, existing)
	if len(out.Facts) == 0 {
		if out.SkipNewFact {
			log.Printf("[factworld] store job %s: L2 dropped new fact (B)", jobID)
			return nil
		}
		return fmt.Errorf("no facts extracted")
	}
	atomRepo, err := atoms.NewRepo(e.dataDir)
	if err != nil {
		return err
	}
	if in.CorrelationID != "" {
		for _, f := range existing {
			if f.CorrelationID == in.CorrelationID {
				_ = atomRepo.RemoveByEpisode(f.EpisodeID)
			}
		}
		if err := e.repo.ReplaceByCorrelation(in.CorrelationID, out.Facts); err != nil {
			return err
		}
	} else {
		for _, f := range out.Facts {
			if err := e.repo.Append(f); err != nil {
				return err
			}
		}
	}
	if len(out.Atoms) > 0 {
		if err := atomRepo.AppendEpisode(jobID, out.Atoms); err != nil {
			log.Printf("[factworld] atoms append: %v", err)
		}
	}
	if out.Fallback {
		log.Printf("[factworld] store job %s: rules fallback (llm off or failed)", jobID)
	} else {
		log.Printf("[factworld] store job %s: llm extract atoms_kept=%d dropped=%d", jobID, out.AtomsKept, out.AtomsDrop)
	}
	if out.L2Applied {
		log.Printf("[factworld] store job %s: L2 conflict candidates=%d drop_new=%v", jobID, out.L2CandidateCount, out.L2DropNew)
	}
	all, err := e.repo.List()
	if err != nil {
		return err
	}
	dcfg := degenerate.DefaultConfig()
	all = degenerate.ApplyStaleDecay(all, time.Now().UTC(), dcfg)
	var extra []graph.Edge
	if len(out.Facts) > 0 && len(out.SupersedeIDs) > 0 {
		all, extra = degenerate.ApplySupersedes(all, out.Facts[0].ID, out.SupersedeIDs, dcfg)
		log.Printf("[factworld] supersede %d fact(s) for job %s", len(out.SupersedeIDs), jobID)
	}
	if err := e.repo.Rewrite(all); err != nil {
		return err
	}
	align.EnqueueFuzzyPairs(e.dataDir, out.FuzzyPairs)
	edges := graph.DeriveEdges(all)
	if err := graph.WriteEdges(e.dataDir, degenerate.MergeExtraEdges(edges, extra)); err != nil {
		log.Printf("[factworld] write edges: %v", err)
	}
	return nil
}

func (e *FactWorldEngine) moveJob(jobID, state string) error {
	src := filepath.Join(e.dataDir, "jobs", "pending", jobID+".json")
	dst := filepath.Join(e.dataDir, "jobs", state, jobID+".json")
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	return os.Rename(src, dst)
}
