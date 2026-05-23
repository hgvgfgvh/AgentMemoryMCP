package graph

// Edge 持久图边（edges.jsonl 一行）。
type Edge struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Type      string  `json:"type"`
	Weight    float64 `json:"weight"`
	EpisodeID string  `json:"episode_id,omitempty"`
}

const (
	EdgeHasTag          = "has_tag"
	EdgeUsedTool        = "used_tool"
	EdgeFromEpisode     = "from_episode"
	EdgeSameCorrelation = "same_correlation"
	EdgeSimilar         = "similar"
	EdgeSource          = "source"
	EdgePitfall         = "pitfall"
	EdgeSupersedes      = "supersedes"
)

func edgeWeight(edgeType string) float64 {
	switch edgeType {
	case EdgePitfall:
		return 0.6
	case EdgeSimilar:
		return 0.7
	case EdgeHasTag, EdgeUsedTool:
		return 0.85
	case EdgeSameCorrelation:
		return 0.9
	default:
		return 0.8
	}
}
