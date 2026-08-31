package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender_ContainsExpectedSections(t *testing.T) {
	md := Render(sampleReport())

	for _, want := range []string{
		"# agent-stack-audit - Laporan Scan",
		"## Ringkasan",
		"## Konflik Hook",
		"## Estimasi Token Overhead",
		"## Audit Memori",
		"## Trust Report",
		"conflict-001 - CONFIRMED",
		"trust-001 - LIKELY",
		"agent-stack-audit v0.1.0",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("report.md missing expected content: %q", want)
		}
	}
}

func TestRender_ThousandsSeparator(t *testing.T) {
	md := Render(sampleReport())
	if !strings.Contains(md, "~8.420 token") {
		t.Errorf("expected thousands-separated token estimate '~8.420 token', got:\n%s", md)
	}
}

func TestRender_HumanReadableByteSize(t *testing.T) {
	md := Render(sampleReport())
	if !strings.Contains(md, "4.3 MB") {
		t.Errorf("expected human-readable size for 4521984 bytes, got:\n%s", md)
	}
}

func TestRender_EmptySectionsDontCrash(t *testing.T) {
	r := Report{SchemaVersion: "1.0", ScannedAt: "2026-08-30T10:00:00Z", Host: "test"}
	md := Render(r)
	if !strings.Contains(md, "Tidak ada konflik hook ditemukan.") {
		t.Errorf("expected empty-state message for conflicts, got:\n%s", md)
	}
}

func TestRender_TokenCostSortedDescending(t *testing.T) {
	r := sampleReport()
	r.TokenCost = []TokenCostEntry{
		{SourceSystem: "small", EstimatedTokens: 50, EstimationMethod: "char_ratio_per_content_type_approximate"},
		{SourceSystem: "big", EstimatedTokens: 5000, EstimationMethod: "char_ratio_per_content_type_approximate"},
	}
	md := Render(r)
	bigIdx := strings.Index(md, "| big |")
	smallIdx := strings.Index(md, "| small |")
	if bigIdx == -1 || smallIdx == -1 || bigIdx > smallIdx {
		t.Errorf("expected 'big' (5000 tokens) listed before 'small' (50 tokens), got:\n%s", md)
	}
}

func TestMarkdownWriter_WritesFile(t *testing.T) {
	dir := t.TempDir()
	if err := NewMarkdownWriter().Write(sampleReport(), dir); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "report.md"))
	if err != nil {
		t.Fatalf("report.md not written: %v", err)
	}
	if !strings.HasPrefix(string(data), "# agent-stack-audit") {
		t.Errorf("report.md does not start with the expected title")
	}
}

func TestFormatThousands(t *testing.T) {
	cases := map[int]string{
		0: "0", 5: "5", 999: "999", 1000: "1.000",
		8420: "8.420", 1000000: "1.000.000", -1234: "-1.234",
	}
	for in, want := range cases {
		if got := formatThousands(in); got != want {
			t.Errorf("formatThousands(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		500:     "500 B",
		4521984: "4.3 MB",
		1258291: "1.2 MB",
	}
	for in, want := range cases {
		if got := formatBytes(in); got != want {
			t.Errorf("formatBytes(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	if got := formatTimestamp("2026-08-29T22:10:00Z"); got != "2026-08-29 22:10" {
		t.Errorf("formatTimestamp = %q, want 2026-08-29 22:10", got)
	}
	if got := formatTimestamp("not-a-timestamp"); got != "not-a-timestamp" {
		t.Errorf("formatTimestamp should fall back to the raw string, got %q", got)
	}
}
