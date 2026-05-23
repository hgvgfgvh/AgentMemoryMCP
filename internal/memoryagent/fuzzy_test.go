package memoryagent

import "testing"

func TestEvidenceAnchored_FuzzyPunctuation(t *testing.T) {
	episode := "门户回复: connection timeout when listing WorkSpace directory."
	evidence := "Connection Timeout when listing WorkSpace"
	if !EvidenceAnchored(evidence, episode, 0.85) {
		t.Fatal("expected fuzzy anchor pass")
	}
}

func TestEvidenceAnchored_HardSubstringFails(t *testing.T) {
	episode := "connection timeout"
	evidence := "Connection Timeout."
	if EvidenceAnchored(evidence, episode, 0.85) {
		// fuzzy should still pass
		return
	}
	t.Fatal("expected pass via normalization")
}

func TestEvidenceAnchored_Unrelated(t *testing.T) {
	if EvidenceAnchored("totally unrelated", "connection timeout in WorkSpace", 0.85) {
		t.Fatal("expected fail")
	}
}
