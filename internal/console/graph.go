package console

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"AgentTestMemoryMCP/internal/facts"
)

// GraphNode 拓扑图节点（供 vis-network 等前端使用）。
type GraphNode struct {
	ID    string         `json:"id"`
	Label string         `json:"label"`
	Group string         `json:"group"` // fact | episode | tag | tool | source
	Title string         `json:"title,omitempty"`
	Size  int            `json:"size,omitempty"`
	Data  map[string]any `json:"data,omitempty"`
}

// GraphEdge 拓扑图边。
type GraphEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
	Group string `json:"group,omitempty"` // from_episode | has_tag | used_tool | same_correlation | similar
}

// GraphView 完整图视图。
type GraphView struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Stats GraphStats  `json:"stats"`
}

// GraphStats 汇总统计。
type GraphStats struct {
	Facts    int `json:"facts"`
	Episodes int `json:"episodes"`
	Tags     int `json:"tags"`
	Tools    int `json:"tools"`
	Sources  int `json:"sources"`
	Edges    int `json:"edges"`
}

// BuildGraphFromFacts 由扁平 facts 推导拓扑（标签/工具/episode 枢纽 + 事实相似边）。
func BuildGraphFromFacts(all []facts.Fact, maxFacts int) GraphView {
	if maxFacts <= 0 {
		maxFacts = 200
	}
	fs := all
	if len(fs) > maxFacts {
		fs = fs[len(fs)-maxFacts:]
	}

	var view GraphView
	episodeSeen := map[string]bool{}
	tagCount := map[string]int{}
	toolSeen := map[string]bool{}
	sourceSeen := map[string]bool{}

	for _, f := range fs {
		view.Stats.Facts++
		nid := nodeIDFact(f.ID)
		label := truncateLabel(f.Text, 36)
		if label == "" {
			label = truncateLabel(f.ID, 24)
		}
		title := buildFactTitle(f)
		size := 12 + int(f.Confidence*14)
		if f.Outcome == "fail" || f.Outcome == "failed" {
			size += 4
		}
		view.Nodes = append(view.Nodes, GraphNode{
			ID: nid, Label: label, Group: "fact", Title: title, Size: size,
			Data: factDataMap(f),
		})

		if eid := strings.TrimSpace(f.EpisodeID); eid != "" && !episodeSeen[eid] {
			episodeSeen[eid] = true
			view.Stats.Episodes++
			view.Nodes = append(view.Nodes, GraphNode{
				ID: nodeIDEpisode(eid), Label: truncateLabel(eid, 28), Group: "episode",
				Title: "Episode: " + eid, Size: 10,
				Data: map[string]any{"episode_id": eid},
			})
		}
		if eid := strings.TrimSpace(f.EpisodeID); eid != "" {
			view.Edges = append(view.Edges, GraphEdge{
				From: nid, To: nodeIDEpisode(eid), Label: "from_episode", Group: "from_episode",
			})
		}

		for _, t := range f.Tags {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			tagCount[t]++
		}
		for _, tool := range f.Tools {
			tool = strings.TrimSpace(tool)
			if tool == "" || toolSeen[tool] {
				continue
			}
			toolSeen[tool] = true
			view.Stats.Tools++
			tid := nodeIDTool(tool)
			view.Nodes = append(view.Nodes, GraphNode{
				ID: tid, Label: truncateLabel(tool, 32), Group: "tool",
				Title: "Tool: " + tool, Size: 8,
			})
		}
		for _, tool := range f.Tools {
			tool = strings.TrimSpace(tool)
			if tool == "" {
				continue
			}
			view.Edges = append(view.Edges, GraphEdge{
				From: nid, To: nodeIDTool(tool), Label: "used_tool", Group: "used_tool",
			})
		}

		if src := strings.TrimSpace(f.Source); src != "" {
			if !sourceSeen[src] {
				sourceSeen[src] = true
				view.Stats.Sources++
				view.Nodes = append(view.Nodes, GraphNode{
					ID: nodeIDSource(src), Label: truncateLabel(src, 24), Group: "source",
					Title: "Source: " + src, Size: 8,
				})
			}
			view.Edges = append(view.Edges, GraphEdge{
				From: nid, To: nodeIDSource(src), Label: "source", Group: "source",
			})
		}
	}

	// 标签枢纽：至少 2 条 fact 共享才建 tag 节点，避免星图爆炸
	var sharedTags []string
	for tag, n := range tagCount {
		if n >= 2 {
			sharedTags = append(sharedTags, tag)
		}
	}
	sort.Strings(sharedTags)
	for _, tag := range sharedTags {
		view.Stats.Tags++
		tid := nodeIDTag(tag)
		view.Nodes = append(view.Nodes, GraphNode{
			ID: tid, Label: "#" + truncateLabel(tag, 20), Group: "tag",
			Title: "Tag: " + tag, Size: 10,
		})
	}
	for _, f := range fs {
		nid := nodeIDFact(f.ID)
		for _, tag := range f.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" || tagCount[tag] < 2 {
				continue
			}
			view.Edges = append(view.Edges, GraphEdge{
				From: nid, To: nodeIDTag(tag), Label: "has_tag", Group: "has_tag",
			})
		}
	}

	// 同 correlation 连线
	corrIndex := map[string][]string{}
	for _, f := range fs {
		c := strings.TrimSpace(f.CorrelationID)
		if c == "" {
			continue
		}
		corrIndex[c] = append(corrIndex[c], nodeIDFact(f.ID))
	}
	for _, ids := range corrIndex {
		if len(ids) < 2 {
			continue
		}
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				view.Edges = append(view.Edges, GraphEdge{
					From: ids[i], To: ids[j], Label: "same_turn", Group: "same_correlation",
				})
			}
		}
	}

	// 标签 Jaccard 相似（fact-fact，阈值 0.35，限制度数）
	for i := 0; i < len(fs); i++ {
		for j := i + 1; j < len(fs); j++ {
			if tagJaccard(fs[i].Tags, fs[j].Tags) >= 0.35 {
				view.Edges = append(view.Edges, GraphEdge{
					From: nodeIDFact(fs[i].ID), To: nodeIDFact(fs[j].ID),
					Label: "similar", Group: "similar",
				})
			}
		}
	}

	view.Stats.Edges = len(view.Edges)
	return view
}

func factDataMap(f facts.Fact) map[string]any {
	return map[string]any{
		"id": f.ID, "episode_id": f.EpisodeID, "source": f.Source,
		"correlation_id": f.CorrelationID, "text": f.Text, "tags": f.Tags,
		"outcome": f.Outcome, "tools": f.Tools, "artifacts": f.Artifacts,
		"confidence": f.Confidence, "weight": f.Weight,
		"created_at": f.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func buildFactTitle(f facts.Fact) string {
	var b strings.Builder
	b.WriteString(f.ID)
	if f.Outcome != "" {
		b.WriteString("\noutcome: " + f.Outcome)
	}
	if f.Confidence > 0 {
		b.WriteString("\nconfidence: ")
		b.WriteString(formatFloat(f.Confidence))
	}
	if len(f.Tags) > 0 {
		b.WriteString("\ntags: " + strings.Join(f.Tags, ", "))
	}
	if len(f.Tools) > 0 {
		b.WriteString("\ntools: " + strings.Join(f.Tools, ", "))
	}
	if t := strings.TrimSpace(f.Text); t != "" {
		b.WriteString("\n\n")
		if len([]rune(t)) > 280 {
			b.WriteString(string([]rune(t)[:280]))
			b.WriteString("…")
		} else {
			b.WriteString(t)
		}
	}
	return b.String()
}

func tagJaccard(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	setA := map[string]bool{}
	for _, x := range a {
		x = strings.TrimSpace(x)
		if x != "" {
			setA[x] = true
		}
	}
	inter := 0
	union := len(setA)
	for _, x := range b {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if setA[x] {
			inter++
		} else {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func truncateLabel(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	r := []rune(s)
	return string(r[:maxRunes]) + "…"
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func nodeIDFact(id string) string    { return "fact:" + id }
func nodeIDEpisode(id string) string { return "episode:" + id }
func nodeIDTag(tag string) string    { return "tag:" + tag }
func nodeIDTool(tool string) string  { return "tool:" + tool }
func nodeIDSource(src string) string { return "source:" + src }
