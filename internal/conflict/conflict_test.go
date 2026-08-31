// SPDX-License-Identifier: MIT

package conflict

import (
	"testing"

	"github.com/damta8827773/agent-stack-audit/internal/discover"
)

func hook(source, event, matcher string) discover.SkillEntry {
	return discover.SkillEntry{
		SourceSystem: source,
		Type:         "hook",
		Event:        event,
		Matcher:      matcher,
		AlwaysOn:     true,
	}
}

// TestCheck_RealConfirmedConflict is the golden fixture from FASE 0
// research: claude-mem and superpowers both register SessionStart with the
// identical matcher "startup|clear|compact".
func TestCheck_RealConfirmedConflict(t *testing.T) {
	entries := []discover.SkillEntry{
		hook("claude-mem", "SessionStart", "startup|clear|compact"),
		hook("superpowers", "SessionStart", "startup|clear|compact"),
	}

	conflicts := NewPatternChecker().Check(entries)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %+v", len(conflicts), conflicts)
	}
	c := conflicts[0]
	if c.Confidence != "CONFIRMED" {
		t.Errorf("Confidence = %q, want CONFIRMED", c.Confidence)
	}
	if c.Event != "SessionStart" {
		t.Errorf("Event = %q, want SessionStart", c.Event)
	}
	if c.ID == "" {
		t.Errorf("ID should not be empty")
	}
}

// TestCheck_PartialOverlapIsLikely reproduces Scenario A's refined finding:
// watermarks-remover's PostToolUse matcher "Write|Edit|MultiEdit|NotebookEdit"
// overlaps with, but isn't identical to, an "Edit"-only matcher.
func TestCheck_PartialOverlapIsLikely(t *testing.T) {
	entries := []discover.SkillEntry{
		hook("watermarks-remover", "PostToolUse", "Write|Edit|MultiEdit|NotebookEdit"),
		hook("ecc", "PostToolUse", "Edit"),
	}

	conflicts := NewPatternChecker().Check(entries)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Confidence != "LIKELY" {
		t.Errorf("Confidence = %q, want LIKELY", conflicts[0].Confidence)
	}
}

func TestCheck_NoOverlapIsInformational(t *testing.T) {
	entries := []discover.SkillEntry{
		hook("gstack", "PreToolUse", "AskUserQuestion"),
		hook("some-other-tool", "PreToolUse", "Bash"),
	}

	conflicts := NewPatternChecker().Check(entries)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Confidence != "INFORMATIONAL" {
		t.Errorf("Confidence = %q, want INFORMATIONAL", conflicts[0].Confidence)
	}
}

func TestCheck_SameSourceIsNeverAConflict(t *testing.T) {
	entries := []discover.SkillEntry{
		hook("gstack", "Stop", "*"),
		hook("gstack", "Stop", "*"),
	}
	conflicts := NewPatternChecker().Check(entries)
	if len(conflicts) != 0 {
		t.Fatalf("expected 0 conflicts for hooks from the same source, got %d: %+v", len(conflicts), conflicts)
	}
}

func TestCheck_DifferentEventsNeverConflict(t *testing.T) {
	entries := []discover.SkillEntry{
		hook("claude-mem", "SessionStart", "startup|clear|compact"),
		hook("superpowers", "Stop", "startup|clear|compact"),
	}
	conflicts := NewPatternChecker().Check(entries)
	if len(conflicts) != 0 {
		t.Fatalf("expected 0 conflicts across different events, got %d: %+v", len(conflicts), conflicts)
	}
}

func TestCheck_WildcardOverlapsEverything(t *testing.T) {
	entries := []discover.SkillEntry{
		hook("claude-mem", "PostToolUse", "*"),
		hook("watermarks-remover", "PostToolUse", "Write|Edit|MultiEdit|NotebookEdit"),
	}
	conflicts := NewPatternChecker().Check(entries)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Confidence != "LIKELY" {
		t.Errorf("Confidence = %q, want LIKELY (wildcard overlaps but isn't textually identical)", conflicts[0].Confidence)
	}
}

func TestCheck_BothWildcardIsConfirmed(t *testing.T) {
	entries := []discover.SkillEntry{
		hook("claude-mem", "PostToolUse", "*"),
		hook("some-tool", "PostToolUse", ""),
	}
	conflicts := NewPatternChecker().Check(entries)
	if len(conflicts) != 1 || conflicts[0].Confidence != "CONFIRMED" {
		t.Fatalf("expected 1 CONFIRMED conflict for two wildcard matchers, got %+v", conflicts)
	}
}

func TestCheck_IgnoresNonHookEntries(t *testing.T) {
	entries := []discover.SkillEntry{
		{SourceSystem: "ecc", Type: "skill", Path: "skill.md"},
		{SourceSystem: "gstack", Type: "plugin", Path: "plugin.json"},
	}
	conflicts := NewPatternChecker().Check(entries)
	if len(conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, non-hook entries must be ignored, got %d", len(conflicts))
	}
}

func TestCheck_ThreeWayProducesPairwiseConflicts(t *testing.T) {
	entries := []discover.SkillEntry{
		hook("claude-mem", "SessionStart", "startup|clear|compact"),
		hook("superpowers", "SessionStart", "startup|clear|compact"),
		hook("gstack", "SessionStart", "startup|clear|compact"),
	}
	conflicts := NewPatternChecker().Check(entries)
	if len(conflicts) != 3 {
		t.Fatalf("expected 3 pairwise conflicts among 3 mutually-conflicting sources, got %d: %+v", len(conflicts), conflicts)
	}
	ids := map[string]bool{}
	for _, c := range conflicts {
		if ids[c.ID] {
			t.Errorf("duplicate conflict ID: %s", c.ID)
		}
		ids[c.ID] = true
	}
}

func TestMatcherRelation(t *testing.T) {
	cases := []struct {
		name            string
		a, b            string
		wantIdentical   bool
		wantOverlapping bool
	}{
		{"identical simple", "Write|Edit", "Write|Edit", true, true},
		{"identical reordered", "Write|Edit", "Edit|Write", true, true},
		{"partial overlap", "Write|Edit|MultiEdit", "Edit", false, true},
		{"no overlap", "Bash", "Read", false, false},
		{"wildcard vs specific", "*", "Bash", false, true},
		{"both wildcard", "*", "", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			identical, overlapping := matcherRelation(tc.a, tc.b)
			if identical != tc.wantIdentical || overlapping != tc.wantOverlapping {
				t.Errorf("matcherRelation(%q, %q) = (%v, %v), want (%v, %v)",
					tc.a, tc.b, identical, overlapping, tc.wantIdentical, tc.wantOverlapping)
			}
		})
	}
}
