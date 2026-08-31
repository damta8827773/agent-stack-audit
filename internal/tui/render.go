// Package tui renders the scan summary for the terminal using lipgloss.
// lipgloss's default renderer detects terminal capability on its own and
// downgrades to plain text automatically when stdout isn't a color-capable
// TTY, which is what satisfies FASE 8's "otomatis fallback ke plain text
// kalau tidak [didukung]" requirement without extra detection code here.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/damtafaiz/agent-stack-audit/internal/report"
)

var (
	titleStyle     = lipgloss.NewStyle().Bold(true)
	labelStyle     = lipgloss.NewStyle().Bold(true)
	confirmedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	likelyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	infoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

// Render produces the interactive-summary text shown by `scan --tui`
// (FASE 8). It is a static styled render, not a bubbletea event loop -
// FASE 1 lists bubbletea as optional, and a summary block has no
// interaction to model.
func Render(r report.Report) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("agent-stack-audit - Ringkasan Scan"))
	b.WriteString("\n\n")

	fmt.Fprintf(&b, "%s %d\n", labelStyle.Render("Total skill ditemukan:"), r.Summary.TotalSkillsFound)
	fmt.Fprintf(&b, "%s %d\n", labelStyle.Render("Total hook ditemukan:"), r.Summary.TotalHooksFound)
	fmt.Fprintf(&b, "%s %d\n", labelStyle.Render("Konflik ditemukan:"), r.Summary.ConflictsFound)
	fmt.Fprintf(&b, "%s ~%d token\n", labelStyle.Render("Estimasi token overhead:"), r.Summary.EstimatedTokenOverhead)
	fmt.Fprintf(&b, "%s %d\n", labelStyle.Render("Memory store ditemukan:"), r.Summary.MemoryStoresFound)
	fmt.Fprintf(&b, "%s %d\n", labelStyle.Render("Peringatan trust:"), r.Summary.TrustWarnings)

	if len(r.Conflicts) > 0 {
		b.WriteString("\n" + titleStyle.Render("Konflik Hook") + "\n")
		for _, c := range r.Conflicts {
			b.WriteString(confidenceStyle(c.Confidence).Render(
				fmt.Sprintf("[%s] %s - %s (%s)", c.Confidence, c.ID, c.Event, strings.Join(c.Sources, " vs "))))
			b.WriteString("\n")
		}
	}

	if len(r.TrustReport) > 0 {
		b.WriteString("\n" + titleStyle.Render("Trust Report") + "\n")
		for _, t := range r.TrustReport {
			b.WriteString(confidenceStyle(t.Confidence).Render(fmt.Sprintf("[%s] %s - %s", t.Confidence, t.ID, t.Finding)))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func confidenceStyle(confidence string) lipgloss.Style {
	switch confidence {
	case "CONFIRMED":
		return confirmedStyle
	case "LIKELY":
		return likelyStyle
	default:
		return infoStyle
	}
}
