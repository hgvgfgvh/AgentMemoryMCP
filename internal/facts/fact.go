package facts

import "time"

// Fact 抽取后的事实点（第三层记忆最小单元）。
type Fact struct {
	ID            string    `json:"id"`
	EpisodeID     string    `json:"episode_id"`
	Source        string    `json:"source,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	Text          string    `json:"text"`
	Tags          []string  `json:"tags,omitempty"`
	Outcome       string    `json:"outcome,omitempty"` // success | fail | unknown
	Tools         []string  `json:"tools,omitempty"`
	Artifacts     []string  `json:"artifacts,omitempty"`
	TierHint      int       `json:"tier_hint,omitempty"`
	Confidence    float64   `json:"confidence"`
	Weight        float64   `json:"weight"`
	CreatedAt     time.Time `json:"created_at"`
}
