package atoms

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"AgentTestMemoryMCP/internal/memoryagent"
)

// Repo atoms.jsonl 追加读写。
type Repo struct {
	path string
	mu   sync.RWMutex
}

func NewRepo(dataDir string) (*Repo, error) {
	dir := filepath.Join(dataDir, "atoms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Repo{path: filepath.Join(dir, "atoms.jsonl")}, nil
}

func (r *Repo) AppendEpisode(episodeID string, atoms []memoryagent.StoredAtom) error {
	if len(atoms) == 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	fh, err := os.OpenFile(r.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer fh.Close()
	for _, a := range atoms {
		b, err := json.Marshal(a)
		if err != nil {
			return err
		}
		if _, err := fh.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}

// RemoveByEpisode 重写文件，去掉指定 episode 的原子（correlation 替换时用）。
func (r *Repo) RemoveByEpisode(episodeID string) error {
	all, err := r.List()
	if err != nil {
		return err
	}
	var kept []memoryagent.StoredAtom
	for _, a := range all {
		if a.EpisodeID != episodeID {
			kept = append(kept, a)
		}
	}
	return r.rewrite(kept)
}

func (r *Repo) List() ([]memoryagent.StoredAtom, error) {
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
	var out []memoryagent.StoredAtom
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for sc.Scan() {
		var a memoryagent.StoredAtom
		if err := json.Unmarshal([]byte(sc.Text()), &a); err == nil {
			out = append(out, a)
		}
	}
	return out, sc.Err()
}

func (r *Repo) rewrite(atoms []memoryagent.StoredAtom) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	tmp := r.path + ".tmp"
	fh, err := os.Create(tmp)
	if err != nil {
		return err
	}
	for _, a := range atoms {
		b, err := json.Marshal(a)
		if err != nil {
			fh.Close()
			return err
		}
		if _, err := fh.Write(append(b, '\n')); err != nil {
			fh.Close()
			return err
		}
	}
	if err := fh.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

func EnsureDir(dataDir string) error {
	return os.MkdirAll(filepath.Join(dataDir, "atoms"), 0o755)
}

func Path(dataDir string) string {
	return filepath.Join(dataDir, "atoms", "atoms.jsonl")
}

func CountLines(dataDir string) (int, error) {
	p := Path(dataDir)
	fh, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer fh.Close()
	n := 0
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		n++
	}
	return n, sc.Err()
}
