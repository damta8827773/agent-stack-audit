// Package tokencost estimates the token overhead of always-on skill
// instructions using a flat character-to-token ratio. This is an
// approximation, not a real tokenizer — FASE 5 requires the estimation
// method to be reported explicitly so nobody mistakes it for an exact
// count from the official API.
package tokencost

import (
	"os"
	"sort"

	"github.com/damtafaiz/agent-stack-audit/internal/discover"
)

// CharsPerToken is the rough English-text ratio FASE 5 specifies
// (~4 characters per token). It is deliberately crude and documented as
// such rather than calibrated per content type.
const CharsPerToken = 4.0

const EstimationMethod = "char_ratio_approximate"

type Entry struct {
	SourceSystem     string
	EstimatedTokens  int
	AlwaysOn         bool
	EstimationMethod string
}

type Estimator interface {
	Estimate(entries []discover.SkillEntry) []Entry
}

type CharRatioEstimator struct{}

func NewCharRatioEstimator() *CharRatioEstimator { return &CharRatioEstimator{} }

// Estimate sums the character length of every always-on skill file's
// content, grouped by source system, and converts the total to an
// estimated token count. Only Type=="skill" && AlwaysOn entries count —
// on-demand plugins and hooks aren't loaded into context every session, so
// they contribute no overhead to measure here.
func (e *CharRatioEstimator) Estimate(entries []discover.SkillEntry) []Entry {
	totals := map[string]int{}

	for _, se := range entries {
		if se.Type != "skill" || !se.AlwaysOn {
			continue
		}
		data, err := os.ReadFile(se.Path)
		if err != nil {
			continue // unreadable file: skip silently, not fatal
		}
		totals[se.SourceSystem] += int(float64(len(data)) / CharsPerToken)
	}

	sources := make([]string, 0, len(totals))
	for s := range totals {
		sources = append(sources, s)
	}
	sort.Strings(sources)

	result := make([]Entry, 0, len(sources))
	for _, source := range sources {
		result = append(result, Entry{
			SourceSystem:     source,
			EstimatedTokens:  totals[source],
			AlwaysOn:         true,
			EstimationMethod: EstimationMethod,
		})
	}
	return result
}

// Total sums estimated tokens across every entry — the
// summary.estimated_token_overhead figure in report.json.
func Total(entries []Entry) int {
	total := 0
	for _, e := range entries {
		total += e.EstimatedTokens
	}
	return total
}
