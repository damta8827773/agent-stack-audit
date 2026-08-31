package report

import (
	"testing"

	"github.com/damta8827773/agent-stack-audit/internal/conflict"
	"github.com/damta8827773/agent-stack-audit/internal/discover"
	"github.com/damta8827773/agent-stack-audit/internal/memoryaudit"
	"github.com/damta8827773/agent-stack-audit/internal/tokencost"
	"github.com/damta8827773/agent-stack-audit/internal/trustreport"
)

func TestBuild_SummaryCounts(t *testing.T) {
	in := BuildInput{
		Host: "windows-amd64",
		Discover: []discover.SkillEntry{
			{SourceSystem: "ecc", Type: "skill", Path: "a"},
			{SourceSystem: "gstack", Type: "plugin", Path: "b"},
			{SourceSystem: "claude-mem", Type: "hook", Path: "c", Event: "SessionStart"},
			{SourceSystem: "superpowers", Type: "hook", Path: "d", Event: "SessionStart"},
		},
		Conflicts: []conflict.Conflict{
			{ID: "conflict-001", Confidence: "CONFIRMED", Event: "SessionStart", Sources: []string{"claude-mem", "superpowers"}},
		},
		TokenCost: []tokencost.Entry{
			{SourceSystem: "ecc", EstimatedTokens: 100},
			{SourceSystem: "gstack", EstimatedTokens: 250},
		},
		MemoryAudit: []memoryaudit.Entry{
			{System: "claude-mem", Path: "~/.claude-mem/claude-mem.db", Tables: []memoryaudit.TableInfo{
				{Name: "observations", RowCount: 42},
			}},
		},
		TrustReport: []trustreport.Finding{
			{ID: "trust-001", Confidence: "INFORMATIONAL", Finding: "no LICENSE"},
		},
	}

	r := Build(in)

	if r.SchemaVersion != "1.0" {
		t.Errorf("SchemaVersion = %q, want 1.0", r.SchemaVersion)
	}
	if r.Host != "windows-amd64" {
		t.Errorf("Host = %q, want windows-amd64", r.Host)
	}
	if r.ScannedAt == "" {
		t.Errorf("ScannedAt should not be empty")
	}
	if r.Summary.TotalSkillsFound != 2 {
		t.Errorf("TotalSkillsFound = %d, want 2 (skill + plugin)", r.Summary.TotalSkillsFound)
	}
	if r.Summary.TotalHooksFound != 2 {
		t.Errorf("TotalHooksFound = %d, want 2", r.Summary.TotalHooksFound)
	}
	if r.Summary.ConflictsFound != 1 {
		t.Errorf("ConflictsFound = %d, want 1", r.Summary.ConflictsFound)
	}
	if r.Summary.EstimatedTokenOverhead != 350 {
		t.Errorf("EstimatedTokenOverhead = %d, want 350", r.Summary.EstimatedTokenOverhead)
	}
	if r.Summary.MemoryStoresFound != 1 {
		t.Errorf("MemoryStoresFound = %d, want 1", r.Summary.MemoryStoresFound)
	}
	if r.Summary.TrustWarnings != 1 {
		t.Errorf("TrustWarnings = %d, want 1", r.Summary.TrustWarnings)
	}
	if len(r.MemoryAudit) != 1 || len(r.MemoryAudit[0].Tables) != 1 || r.MemoryAudit[0].Tables[0].Name != "observations" {
		t.Errorf("expected memoryaudit.Entry.Tables to be threaded through to the report, got: %+v", r.MemoryAudit)
	}
}

func TestBuild_EmptyInputProducesZeroedReport(t *testing.T) {
	r := Build(BuildInput{Host: "linux-arm64"})
	if r.Summary.TotalSkillsFound != 0 || r.Summary.ConflictsFound != 0 {
		t.Errorf("expected all-zero summary for empty input, got %+v", r.Summary)
	}
}
