package report

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/damtafaiz/agent-stack-audit/internal/version"
)

type MarkdownWriter struct{}

func NewMarkdownWriter() *MarkdownWriter { return &MarkdownWriter{} }

func (w *MarkdownWriter) Write(r Report, destination string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destination, "report.md"), []byte(Render(r)), 0o644)
}

// Render produces the exact report.md shape shown in Lampiran F.
func Render(r Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# agent-stack-audit — Laporan Scan\n\n")
	fmt.Fprintf(&b, "Dijalankan: %s\n", r.ScannedAt)
	fmt.Fprintf(&b, "Host: %s\n", r.Host)
	fmt.Fprintf(&b, "Skema versi: %s\n\n", r.SchemaVersion)

	b.WriteString("## Ringkasan\n\n")
	b.WriteString("| Metrik | Nilai |\n|---|---|\n")
	fmt.Fprintf(&b, "| Total skill ditemukan | %s |\n", formatThousands(r.Summary.TotalSkillsFound))
	fmt.Fprintf(&b, "| Total hook ditemukan | %s |\n", formatThousands(r.Summary.TotalHooksFound))
	fmt.Fprintf(&b, "| Konflik ditemukan | %s |\n", formatThousands(r.Summary.ConflictsFound))
	fmt.Fprintf(&b, "| Estimasi token overhead | ~%s token |\n", formatThousands(r.Summary.EstimatedTokenOverhead))
	fmt.Fprintf(&b, "| Memory store ditemukan | %s |\n", formatThousands(r.Summary.MemoryStoresFound))
	fmt.Fprintf(&b, "| Peringatan trust | %s |\n\n", formatThousands(r.Summary.TrustWarnings))

	b.WriteString("## Konflik Hook\n\n")
	if len(r.Conflicts) == 0 {
		b.WriteString("Tidak ada konflik hook ditemukan.\n\n")
	}
	for _, c := range r.Conflicts {
		fmt.Fprintf(&b, "### %s — %s\n", c.ID, c.Confidence)
		fmt.Fprintf(&b, "- Event: `%s`\n", c.Event)
		fmt.Fprintf(&b, "- Matcher: `%s`\n", c.Matcher)
		fmt.Fprintf(&b, "- Sumber: %s\n", strings.Join(c.Sources, ", "))
		fmt.Fprintf(&b, "- Detail: %s\n\n", c.Detail)
	}

	b.WriteString("## Estimasi Token Overhead\n\n")
	if len(r.TokenCost) == 0 {
		b.WriteString("Tidak ada skill always-on yang terdeteksi.\n\n")
	} else {
		b.WriteString("| Sumber | Estimasi token | Always-on |\n|---|---|---|\n")
		for _, tc := range sortedByTokensDesc(r.TokenCost) {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", tc.SourceSystem, formatThousands(tc.EstimatedTokens), yaTidak(tc.AlwaysOn))
		}
		b.WriteString("\n")
		method := "char_ratio_approximate"
		if len(r.TokenCost) > 0 {
			method = r.TokenCost[0].EstimationMethod
		}
		fmt.Fprintf(&b, "Metode estimasi: %s (~4 karakter/token). Ini APROKSIMASI,\nbukan angka tokenizer resmi.\n\n", method)
	}

	b.WriteString("## Audit Memori\n\n")
	if len(r.MemoryAudit) == 0 {
		b.WriteString("Tidak ada memory store yang terdeteksi.\n\n")
	} else {
		b.WriteString("| Sistem | Path | Ukuran | Terakhir ditulis | World-readable |\n|---|---|---|---|---|\n")
		for _, m := range r.MemoryAudit {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				m.System, m.Path, formatBytes(m.SizeBytes), formatTimestamp(m.LastModified), yaTidak(m.WorldReadable))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Trust Report\n\n")
	if len(r.TrustReport) == 0 {
		b.WriteString("Tidak ada temuan trust report.\n\n")
	}
	for _, t := range r.TrustReport {
		fmt.Fprintf(&b, "### %s — %s\n", t.ID, t.Confidence)
		fmt.Fprintf(&b, "- Temuan: %s\n", t.Finding)
		fmt.Fprintf(&b, "- Path: %s\n\n", t.Path)
	}

	b.WriteString("---\n")
	fmt.Fprintf(&b, "Laporan ini dibuat oleh agent-stack-audit v%s. Read-only, tidak ada\nperubahan yang dilakukan pada sistem.\n", version.Version)

	return b.String()
}

func sortedByTokensDesc(entries []TokenCostEntry) []TokenCostEntry {
	sorted := make([]TokenCostEntry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].EstimatedTokens > sorted[j].EstimatedTokens
	})
	return sorted
}

func yaTidak(b bool) string {
	if b {
		return "Ya"
	}
	return "Tidak"
}

// formatThousands renders an int with "." as the thousands separator,
// matching Lampiran F's own example ("~8.420 token").
func formatThousands(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var groups []string
	for len(s) > 3 {
		groups = append([]string{s[len(s)-3:]}, groups...)
		s = s[:len(s)-3]
	}
	groups = append([]string{s}, groups...)
	result := strings.Join(groups, ".")
	if neg {
		result = "-" + result
	}
	return result
}

// formatBytes renders a byte count as human-readable, matching Lampiran
// F's example ("4.5 MB", "1.2 MB").
func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	size := float64(n)
	unit := ""
	for _, u := range units {
		size /= 1024
		unit = u
		if size < 1024 {
			break
		}
	}
	return fmt.Sprintf("%.1f %s", size, unit)
}

// formatTimestamp converts an RFC3339 timestamp to Lampiran F's display
// format ("2026-08-29 22:10"). Falls back to the raw string if parsing
// fails, so a malformed timestamp degrades gracefully instead of panicking.
func formatTimestamp(rfc3339 string) string {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	return t.Format("2006-01-02 15:04")
}
