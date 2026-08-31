// Package tokencost estimates the token overhead of always-on skill
// instructions using per-content-type character-to-token ratios. This is
// still an approximation, not a real tokenizer - FASE 5 requires the
// estimation method to be reported explicitly so nobody mistakes it for an
// exact count from the official API.
package tokencost

import (
	"os"
	"sort"
	"strings"

	"github.com/damta8827773/agent-stack-audit/internal/discover"
)

// Two ratios instead of one flat CharsPerToken: YAML frontmatter (short
// key: value lines, dense punctuation) tokenizes measurably denser than
// prose body text does. BodyCharsPerToken keeps the original FASE 5 value
// (~4 chars/token, the well-known rough English-text approximation) so a
// skill with no frontmatter estimates exactly as it did before; only the
// frontmatter portion of a file gets the adjusted ratio. Both are still
// hand-picked approximations, not values calibrated against a real
// tokenizer - the method name reported alongside them says so explicitly.
const (
	FrontmatterCharsPerToken = 3.7
	BodyCharsPerToken        = 4.0
)

const EstimationMethod = "char_ratio_per_content_type_approximate"

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

// Estimate sums the token estimate of every always-on skill file's
// content, grouped by source system. Only Type=="skill" && AlwaysOn
// entries count - on-demand plugins and hooks aren't loaded into context
// every session, so they contribute no overhead to measure here.
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
		totals[se.SourceSystem] += estimateTokens(data)
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

// estimateTokens splits content into its YAML frontmatter (if any) and
// body, applying each content type's own ratio and summing the result.
func estimateTokens(content []byte) int {
	frontmatter, body := splitFrontmatterBody(content)
	tokens := float64(len(frontmatter)) / FrontmatterCharsPerToken
	tokens += float64(len(body)) / BodyCharsPerToken
	return int(tokens)
}

// splitFrontmatterBody mirrors discover's own "---"-delimited frontmatter
// detection. Duplicated rather than imported: it's a few lines, and
// tokencost has no other reason to depend on discover's internals.
func splitFrontmatterBody(content []byte) (frontmatter, body []byte) {
	text := string(content)
	if !strings.HasPrefix(text, "---") {
		return nil, content
	}
	rest := text[3:]
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return nil, content
	}
	fm := rest[:end]
	bodyStart := end + len("\n---")
	return []byte(fm), []byte(rest[bodyStart:])
}

// Total sums estimated tokens across every entry - the
// summary.estimated_token_overhead figure in report.json.
func Total(entries []Entry) int {
	total := 0
	for _, e := range entries {
		total += e.EstimatedTokens
	}
	return total
}
