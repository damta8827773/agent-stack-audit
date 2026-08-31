// SPDX-License-Identifier: MIT

package fix

import (
	"strings"
	"testing"

	"github.com/damta8827773/agent-stack-audit/internal/conflict"
)

func TestSuggest_OnlyConfirmedConflicts(t *testing.T) {
	conflicts := []conflict.Conflict{
		{ID: "conflict-001", Confidence: "CONFIRMED", Event: "SessionStart", Matcher: "startup|clear|compact", Sources: []string{"claude-mem", "superpowers"}},
		{ID: "conflict-002", Confidence: "LIKELY", Event: "PostToolUse", Matcher: "Edit", Sources: []string{"ecc", "watermarks-remover"}},
		{ID: "conflict-003", Confidence: "INFORMATIONAL", Event: "PreToolUse", Matcher: "Bash vs Read", Sources: []string{"a", "b"}},
	}

	suggestions := Suggest(conflicts)
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion (CONFIRMED only), got %d: %+v", len(suggestions), suggestions)
	}
	if suggestions[0].ConflictID != "conflict-001" {
		t.Errorf("ConflictID = %q, want conflict-001", suggestions[0].ConflictID)
	}
	if !strings.Contains(suggestions[0].Text, "claude-mem") || !strings.Contains(suggestions[0].Text, "superpowers") {
		t.Errorf("suggestion text missing source names: %q", suggestions[0].Text)
	}
	if !strings.Contains(suggestions[0].Text, "startup|clear|compact") {
		t.Errorf("suggestion text missing the matcher: %q", suggestions[0].Text)
	}
	if !strings.Contains(suggestions[0].Text, "manual") {
		t.Errorf("suggestion must make clear this is a manual step, got: %q", suggestions[0].Text)
	}
}

func TestSuggest_NoConfirmedConflictsReturnsEmpty(t *testing.T) {
	conflicts := []conflict.Conflict{
		{ID: "conflict-001", Confidence: "LIKELY", Event: "PostToolUse"},
	}
	suggestions := Suggest(conflicts)
	if len(suggestions) != 0 {
		t.Fatalf("expected 0 suggestions, got %+v", suggestions)
	}
}

func TestSuggest_EmptyInput(t *testing.T) {
	if s := Suggest(nil); len(s) != 0 {
		t.Fatalf("expected 0 suggestions for nil input, got %+v", s)
	}
}

func TestSuggest_MissingSourcesDoesNotPanic(t *testing.T) {
	conflicts := []conflict.Conflict{
		{ID: "conflict-001", Confidence: "CONFIRMED", Event: "SessionStart", Matcher: "*", Sources: nil},
	}
	suggestions := Suggest(conflicts)
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion even with no sources, got %+v", suggestions)
	}
	if !strings.Contains(suggestions[0].Text, "sumber lain") {
		t.Errorf("expected fallback source label, got: %q", suggestions[0].Text)
	}
}

func TestRenderMarkdown(t *testing.T) {
	suggestions := []Suggestion{
		{ConflictID: "conflict-001", Text: "some suggestion text\n"},
	}
	md := RenderMarkdown(suggestions)
	if !strings.Contains(md, "# Saran Perbaikan") {
		t.Errorf("missing title, got: %q", md)
	}
	if !strings.Contains(md, "conflict-001") {
		t.Errorf("missing conflict ID heading, got: %q", md)
	}
	if !strings.Contains(md, "some suggestion text") {
		t.Errorf("missing suggestion body, got: %q", md)
	}
	if !strings.Contains(md, "bukan perubahan otomatis") {
		t.Errorf("missing the manual-only disclaimer, got: %q", md)
	}
}

func TestRenderMarkdown_Empty(t *testing.T) {
	md := RenderMarkdown(nil)
	if !strings.Contains(md, "# Saran Perbaikan") {
		t.Errorf("expected title even with no suggestions, got: %q", md)
	}
}
