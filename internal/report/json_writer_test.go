package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func sampleReport() Report {
	return Report{
		SchemaVersion: "1.0",
		ScannedAt:     "2026-08-30T10:00:00Z",
		Host:          "windows-amd64",
		Summary: Summary{
			TotalSkillsFound: 14, TotalHooksFound: 6, ConflictsFound: 2,
			EstimatedTokenOverhead: 8420, MemoryStoresFound: 2, TrustWarnings: 3,
		},
		Discover: []DiscoverEntry{
			{SourceSystem: "ECC", Path: "~/.claude/skills/ecc-tdd-workflow", Type: "skill", HasFrontmatter: true, AlwaysOn: true},
		},
		Conflicts: []ConflictEntry{
			{ID: "conflict-001", Confidence: "CONFIRMED", Event: "SessionStart", Matcher: "startup|clear|compact",
				Sources: []string{"claude-mem", "superpowers"}, Detail: "identik persis"},
		},
		TokenCost: []TokenCostEntry{
			{SourceSystem: "gstack", EstimatedTokens: 3200, AlwaysOn: true, EstimationMethod: "char_ratio_per_content_type_approximate"},
		},
		MemoryAudit: []MemoryAuditEntry{
			{System: "claude-mem", Path: "~/.claude-mem/claude-mem.db", SizeBytes: 4521984, LastModified: "2026-08-29T22:10:00Z", WorldReadable: false},
		},
		TrustReport: []TrustReportEntry{
			{ID: "trust-001", Confidence: "LIKELY", Finding: "Skill tanpa file LICENSE", Path: "~/.claude/skills/some-skill"},
		},
	}
}

func TestJSONWriter_WritesValidSchema(t *testing.T) {
	dir := t.TempDir()
	r := sampleReport()

	if err := NewJSONWriter().Write(r, dir); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatalf("report.json not written: %v", err)
	}

	var roundTrip Report
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("report.json is not valid JSON matching Report: %v", err)
	}
	if roundTrip.SchemaVersion != "1.0" {
		t.Errorf("schema_version = %q, want 1.0", roundTrip.SchemaVersion)
	}
	if roundTrip.Summary.EstimatedTokenOverhead != 8420 {
		t.Errorf("estimated_token_overhead = %d, want 8420", roundTrip.Summary.EstimatedTokenOverhead)
	}

	// Field names in the raw JSON must match the snake_case schema from
	// FASE 2 exactly (report.json is a public contract).
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schema_version", "scanned_at", "host", "summary", "discover", "conflicts", "token_cost", "memory_audit", "trust_report"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("report.json missing top-level key %q", key)
		}
	}
}

func TestJSONWriter_CreatesDestinationDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "audit-report")
	if err := NewJSONWriter().Write(sampleReport(), dir); err != nil {
		t.Fatalf("Write should create missing destination dirs, got error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "report.json")); err != nil {
		t.Fatalf("report.json not found in created destination: %v", err)
	}
}
