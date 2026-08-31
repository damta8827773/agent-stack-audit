// Package fix generates human-readable suggestions for CONFIRMED conflicts.
// It never writes to, or even knows the location of, another tool's
// config/hook files - suggestions describe what to change and where in
// prose, and applying them is always a manual step the user takes outside
// this tool. This is the v0.2 "--fix" preview described in
// docs/ROADMAP.md: opt-in, narrowly scoped to CONFIRMED conflicts, never
// silent.
package fix

import (
	"fmt"
	"strings"

	"github.com/damta8827773/agent-stack-audit/internal/conflict"
)

type Suggestion struct {
	ConflictID string
	Text       string
}

// Suggest returns one suggestion per CONFIRMED conflict. LIKELY and
// INFORMATIONAL findings are skipped - --fix stays scoped to the cases
// conflict-check's own algorithm is most confident about (Lampiran B:
// CONFIRMED means an exact matcher match, not a heuristic guess).
func Suggest(conflicts []conflict.Conflict) []Suggestion {
	var suggestions []Suggestion
	for _, c := range conflicts {
		if c.Confidence != "CONFIRMED" {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			ConflictID: c.ID,
			Text:       suggestText(c),
		})
	}
	return suggestions
}

func suggestText(c conflict.Conflict) string {
	a, b := sourceOrUnknown(c.Sources, 0), sourceOrUnknown(c.Sources, 1)

	var s strings.Builder
	fmt.Fprintf(&s, "Event %s: %s dan %s mendaftarkan matcher yang identik persis (%q).\n",
		c.Event, a, b, c.Matcher)
	s.WriteString("Saran (manual - tool ini tidak pernah mengubah file milik plugin/skill lain secara otomatis):\n")
	fmt.Fprintf(&s, "  1. Buka konfigurasi hook %s atau %s di luar tool ini.\n", a, b)
	fmt.Fprintf(&s, "     Persempit matcher-nya supaya tidak identik dengan %q - misal kalau hook itu\n", c.Matcher)
	s.WriteString("     sebenarnya cuma perlu trigger di sebagian tool, ganti ke pattern yang lebih spesifik.\n")
	s.WriteString("  2. Kalau kedua hook memang harus jalan di kondisi yang sama, pastikan keduanya\n")
	s.WriteString("     idempotent (aman dijalankan berkali-kali, tidak saling menimpa hasil satu sama lain).\n")
	return s.String()
}

func sourceOrUnknown(sources []string, i int) string {
	if i < len(sources) {
		return sources[i]
	}
	return "sumber lain"
}

// RenderMarkdown formats every suggestion as one Markdown document.
func RenderMarkdown(suggestions []Suggestion) string {
	var b strings.Builder
	b.WriteString("# Saran Perbaikan (agent-stack-audit --fix)\n\n")
	b.WriteString("Ini SARAN, bukan perubahan otomatis - tool ini tidak pernah menulis ke file\n")
	b.WriteString("milik plugin/skill lain. Terapkan manual sesuai penilaian Anda sendiri.\n\n")
	for _, s := range suggestions {
		fmt.Fprintf(&b, "## %s\n\n%s\n", s.ConflictID, s.Text)
	}
	return b.String()
}
