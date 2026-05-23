package graph

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"AgentTestMemoryMCP/internal/facts"
)

var edgeMu sync.Mutex

// EdgesPath 返回 edges.jsonl 路径。
func EdgesPath(dataDir string) string {
	return filepath.Join(dataDir, "graph", "edges.jsonl")
}

// RebuildEdgesFile 由当前 facts 全量重建边文件。
func RebuildEdgesFile(dataDir string, all []facts.Fact) error {
	edges := DeriveEdges(all)
	return WriteEdges(dataDir, edges)
}

// WriteEdges 覆写 edges.jsonl。
func WriteEdges(dataDir string, edges []Edge) error {
	dir := filepath.Join(dataDir, "graph")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := EdgesPath(dataDir)
	edgeMu.Lock()
	defer edgeMu.Unlock()
	tmp := path + ".tmp"
	fh, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(fh)
	for _, e := range edges {
		if err := enc.Encode(e); err != nil {
			fh.Close()
			return err
		}
	}
	if err := fh.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadEdges 读取边；文件不存在返回 nil。
func LoadEdges(dataDir string) ([]Edge, error) {
	path := EdgesPath(dataDir)
	fh, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer fh.Close()
	var out []Edge
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var e Edge
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		out = append(out, e)
	}
	return out, sc.Err()
}
