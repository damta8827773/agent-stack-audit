// SPDX-License-Identifier: MIT

package tui

import (
	"strings"
	"testing"

	"github.com/damta8827773/agent-stack-audit/internal/report"
)

func TestRender_ContainsSummaryNumbers(t *testing.T) {
	r := report.Report{
		Summary: report.Summary{
			TotalSkillsFound: 14, TotalHooksFound: 6, ConflictsFound: 2,
			EstimatedTokenOverhead: 8420, MemoryStoresFound: 2, TrustWarnings: 3,
		},
	}
	out := Render(r)
	for _, want := range []string{"14", "6", "2", "8420", "3"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output missing %q:\n%s", want, out)
		}
	}
}

func TestRender_ShowsConflictsWhenPresent(t *testing.T) {
	r := report.Report{
		Conflicts: []report.ConflictEntry{
			{ID: "conflict-001", Confidence: "CONFIRMED", Event: "SessionStart", Sources: []string{"claude-mem", "superpowers"}},
		},
	}
	out := Render(r)
	if !strings.Contains(out, "conflict-001") || !strings.Contains(out, "SessionStart") {
		t.Errorf("rendered output missing conflict details:\n%s", out)
	}
}

func TestRender_OmitsConflictSectionWhenEmpty(t *testing.T) {
	out := Render(report.Report{})
	if strings.Contains(out, "Konflik Hook") {
		t.Errorf("expected no 'Konflik Hook' section for zero conflicts, got:\n%s", out)
	}
}

func TestRender_ShowsTrustFindings(t *testing.T) {
	r := report.Report{
		TrustReport: []report.TrustReportEntry{
			{ID: "trust-001", Confidence: "LIKELY", Finding: "Skill tanpa file LICENSE"},
		},
	}
	out := Render(r)
	if !strings.Contains(out, "trust-001") || !strings.Contains(out, "LICENSE") {
		t.Errorf("rendered output missing trust finding details:\n%s", out)
	}
}

func TestRender_ShowsVulnFindingsWhenPresent(t *testing.T) {
	r := report.Report{
		Summary: report.Summary{VulnerabilitiesFound: 1},
		VulnAudit: []report.VulnAuditEntry{
			{
				Confidence: "CONFIRMED", Package: "lodash", InstalledVersion: "4.17.15",
				VulnerabilityID: "GHSA-xxxx", FixedVersion: "4.17.21", Severity: "HIGH",
			},
		},
	}
	out := Render(r)
	if !strings.Contains(out, "lodash") || !strings.Contains(out, "GHSA-xxxx") || !strings.Contains(out, "HIGH") {
		t.Errorf("rendered output missing vuln finding details:\n%s", out)
	}
	if !strings.Contains(out, "Kerentanan ditemukan:") {
		t.Errorf("expected the vuln count summary line, got:\n%s", out)
	}
}

func TestRender_OmitsVulnSectionWhenNotRun(t *testing.T) {
	out := Render(report.Report{})
	if strings.Contains(out, "Vuln Audit") || strings.Contains(out, "Kerentanan ditemukan") {
		t.Errorf("expected no vuln-audit mention when the check never ran, got:\n%s", out)
	}
}

func TestRender_VulnFindingMissingSeverityShowsPlaceholder(t *testing.T) {
	r := report.Report{
		VulnAudit: []report.VulnAuditEntry{
			{Confidence: "CONFIRMED", Package: "x", VulnerabilityID: "OSV-1"},
		},
	}
	out := Render(r)
	if !strings.Contains(out, "severity tidak dilaporkan") {
		t.Errorf("expected the no-severity placeholder, got:\n%s", out)
	}
}
