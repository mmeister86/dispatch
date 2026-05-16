package knowledge

import (
	"strings"
	"testing"
)

func TestXAlgorithmInsightsContainsCoreWritingSignals(t *testing.T) {
	insights := XAlgorithmInsights()
	if strings.TrimSpace(insights) == "" {
		t.Fatal("XAlgorithmInsights should not be empty")
	}

	for _, want := range []string{
		"in-network",
		"out-of-network",
		"hydration",
		"filtering",
		"scoring",
		"dwell",
		"reply",
		"repost",
		"not interested",
		"block",
		"mute",
		"report",
		"heuristic",
	} {
		if !strings.Contains(strings.ToLower(insights), want) {
			t.Fatalf("XAlgorithmInsights missing %q:\n%s", want, insights)
		}
	}
}
