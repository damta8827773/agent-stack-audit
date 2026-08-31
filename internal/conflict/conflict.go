// SPDX-License-Identifier: MIT

// Package conflict detects hooks from different source systems registered
// on overlapping event+matcher combinations. This is pattern matching on
// matcher strings, not static analysis of what the hook commands actually
// do - see FASE 4's explicit limitation: it cannot guarantee catching every
// race condition, only overlapping registrations.
package conflict

import (
	"fmt"
	"sort"

	"github.com/damta8827773/agent-stack-audit/internal/discover"
)

type Conflict struct {
	ID         string
	Confidence string // CONFIRMED | LIKELY | INFORMATIONAL
	Event      string
	Matcher    string
	Sources    []string
	Detail     string
}

type Checker interface {
	Check(entries []discover.SkillEntry) []Conflict
}

type PatternChecker struct{}

func NewPatternChecker() *PatternChecker { return &PatternChecker{} }

func (c *PatternChecker) Check(entries []discover.SkillEntry) []Conflict {
	byEvent := map[string][]discover.SkillEntry{}
	for _, e := range entries {
		if e.Type != "hook" {
			continue
		}
		byEvent[e.Event] = append(byEvent[e.Event], e)
	}

	var events []string
	for ev := range byEvent {
		events = append(events, ev)
	}
	sort.Strings(events)

	var conflicts []Conflict
	n := 1
	for _, event := range events {
		hooks := byEvent[event]
		for i := 0; i < len(hooks); i++ {
			for j := i + 1; j < len(hooks); j++ {
				a, b := hooks[i], hooks[j]
				if a.SourceSystem == b.SourceSystem {
					// Same source registering more than one matcher on the
					// same event is that tool's own design, not a
					// cross-tool conflict.
					continue
				}
				identical, overlapping := matcherRelation(a.Matcher, b.Matcher)
				confidence := "INFORMATIONAL"
				if identical {
					confidence = "CONFIRMED"
				} else if overlapping {
					confidence = "LIKELY"
				}
				conflicts = append(conflicts, Conflict{
					ID:         fmt.Sprintf("conflict-%03d", n),
					Confidence: confidence,
					Event:      event,
					Matcher:    describeMatchers(a.Matcher, b.Matcher),
					Sources:    []string{a.SourceSystem, b.SourceSystem},
					Detail:     detailFor(confidence, event, a, b),
				})
				n++
			}
		}
	}
	return conflicts
}

func describeMatchers(a, b string) string {
	if a == b {
		return a
	}
	return fmt.Sprintf("%s vs %s", a, b)
}

func detailFor(confidence, event string, a, b discover.SkillEntry) string {
	switch confidence {
	case "CONFIRMED":
		return fmt.Sprintf(
			"Kedua hook (%s, %s) terdaftar pada event %s dengan matcher yang identik persis (%q). Urutan eksekusi tidak terjamin.",
			a.SourceSystem, b.SourceSystem, event, a.Matcher)
	case "LIKELY":
		return fmt.Sprintf(
			"Hook pada event %s overlap sebagian: %q (sumber: %s) vs %q (sumber: %s).",
			event, a.Matcher, a.SourceSystem, b.Matcher, b.SourceSystem)
	default:
		return fmt.Sprintf(
			"Dua hook dari sumber berbeda (%s, %s) aktif pada event %s yang sama, tapi matcher-nya tidak overlap (%q vs %q) - informasi konteks saja.",
			a.SourceSystem, b.SourceSystem, event, a.Matcher, b.Matcher)
	}
}
