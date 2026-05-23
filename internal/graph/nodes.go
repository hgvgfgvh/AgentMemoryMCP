package graph

import "strings"

// Node ID 约定（与控制台一致）。
func NodeFact(factID string) string { return "fact:" + strings.TrimSpace(factID) }
func NodeEpisode(id string) string  { return "episode:" + strings.TrimSpace(id) }
func NodeTag(tag string) string     { return "tag:" + strings.TrimSpace(tag) }
func NodeTool(tool string) string   { return "tool:" + strings.TrimSpace(tool) }
func NodeSource(src string) string  { return "source:" + strings.TrimSpace(src) }

// FactIDFromNode 从 fact 节点 ID 解析 fact ID。
func FactIDFromNode(node string) (string, bool) {
	const p = "fact:"
	if strings.HasPrefix(node, p) {
		return strings.TrimPrefix(node, p), true
	}
	return "", false
}
