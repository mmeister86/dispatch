package knowledge

import (
	_ "embed"
	"strings"
)

//go:embed x_algorithm_insights.md
var xAlgorithmInsights string

func XAlgorithmInsights() string {
	return strings.TrimSpace(xAlgorithmInsights)
}
