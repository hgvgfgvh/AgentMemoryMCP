package facts

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Repo 追加读写 facts.jsonl。
type Repo struct {
	path string
	mu   sync.RWMutex
}

func NewRepo(dataDir string) (*Repo, error) {
	dir := filepath.Join(dataDir, "facts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Repo{path: filepath.Join(dir, "facts.jsonl")}, nil
}

func (r *Repo) Append(f Fact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	fh, err := os.OpenFile(r.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer fh.Close()
	_, err = fh.Write(append(b, '\n'))
	return err
}

func (r *Repo) List() ([]Fact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fh, err := os.Open(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer fh.Close()
	var out []Fact
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var f Fact
		if err := json.Unmarshal([]byte(line), &f); err != nil {
			continue
		}
		out = append(out, f)
	}
	return out, sc.Err()
}

func (r *Repo) ReplaceByCorrelation(correlationID string, newFacts []Fact) error {
	if correlationID == "" {
		return r.appendAll(newFacts)
	}
	all, err := r.List()
	if err != nil {
		return err
	}
	var kept []Fact
	for _, f := range all {
		if f.CorrelationID != correlationID {
			kept = append(kept, f)
		}
	}
	kept = append(kept, newFacts...)
	return r.rewrite(kept)
}

func (r *Repo) appendAll(fs []Fact) error {
	for _, f := range fs {
		if err := r.Append(f); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) rewrite(fs []Fact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	tmp := r.path + ".tmp"
	fh, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(fh)
	for _, f := range fs {
		if err := enc.Encode(f); err != nil {
			fh.Close()
			return err
		}
	}
	if err := fh.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, r.path); err != nil {
		return err
	}
	return nil
}

// Path 返回存储路径（测试用）。
func (r *Repo) Path() string { return r.path }

func EnsureDataDirs(dataDir string) error {
	for _, sub := range []string{"episodes", "facts", "graph", "jobs/pending", "jobs/done", "jobs/dead", "store_log"} {
		if err := os.MkdirAll(filepath.Join(dataDir, sub), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", sub, err)
		}
	}
	return nil
}
