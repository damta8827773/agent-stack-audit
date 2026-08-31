package report

import (
	"time"

	"github.com/damtafaiz/agent-stack-audit/internal/conflict"
	"github.com/damtafaiz/agent-stack-audit/internal/discover"
	"github.com/damtafaiz/agent-stack-audit/internal/memoryaudit"
	"github.com/damtafaiz/agent-stack-audit/internal/tokencost"
	"github.com/damtafaiz/agent-stack-audit/internal/trustreport"
)

const SchemaVersion = "1.0"

// BuildInput takes each module's own output type directly - the aggregator
// is the only place that needs to know how to translate them into the
// shared report.json shape, so individual modules stay decoupled from it
// (Lampiran E: "aggregator tidak perlu tahu detail implementasi internal
// masing-masing modul").
type BuildInput struct {
	Host        string
	Discover    []discover.SkillEntry
	Conflicts   []conflict.Conflict
	TokenCost   []tokencost.Entry
	MemoryAudit []memoryaudit.Entry
	TrustReport []trustreport.Finding
}

func Build(in BuildInput) Report {
	discoverEntries := make([]DiscoverEntry, 0, len(in.Discover))
	hookCount := 0
	skillOrPluginCount := 0
	for _, e := range in.Discover {
		discoverEntries = append(discoverEntries, DiscoverEntry{
			SourceSystem:   e.SourceSystem,
			Path:           e.Path,
			Type:           e.Type,
			HasFrontmatter: e.HasFrontmatter,
			AlwaysOn:       e.AlwaysOn,
		})
		if e.Type == "hook" {
			hookCount++
		} else {
			skillOrPluginCount++
		}
	}

	conflictEntries := make([]ConflictEntry, 0, len(in.Conflicts))
	for _, c := range in.Conflicts {
		conflictEntries = append(conflictEntries, ConflictEntry{
			ID:         c.ID,
			Confidence: c.Confidence,
			Event:      c.Event,
			Matcher:    c.Matcher,
			Sources:    c.Sources,
			Detail:     c.Detail,
		})
	}

	tokenEntries := make([]TokenCostEntry, 0, len(in.TokenCost))
	totalTokens := 0
	for _, tc := range in.TokenCost {
		tokenEntries = append(tokenEntries, TokenCostEntry{
			SourceSystem:     tc.SourceSystem,
			EstimatedTokens:  tc.EstimatedTokens,
			AlwaysOn:         tc.AlwaysOn,
			EstimationMethod: tc.EstimationMethod,
		})
		totalTokens += tc.EstimatedTokens
	}

	memEntries := make([]MemoryAuditEntry, 0, len(in.MemoryAudit))
	for _, m := range in.MemoryAudit {
		var tables []TableInfo
		for _, t := range m.Tables {
			tables = append(tables, TableInfo{Name: t.Name, RowCount: t.RowCount})
		}
		memEntries = append(memEntries, MemoryAuditEntry{
			System:        m.System,
			Path:          m.Path,
			SizeBytes:     m.SizeBytes,
			LastModified:  m.LastModified,
			WorldReadable: m.WorldReadable,
			Tables:        tables,
		})
	}

	trustEntries := make([]TrustReportEntry, 0, len(in.TrustReport))
	for _, t := range in.TrustReport {
		trustEntries = append(trustEntries, TrustReportEntry{
			ID:         t.ID,
			Confidence: t.Confidence,
			Finding:    t.Finding,
			Path:       t.Path,
		})
	}

	return Report{
		SchemaVersion: SchemaVersion,
		ScannedAt:     time.Now().UTC().Format(time.RFC3339),
		Host:          in.Host,
		Summary: Summary{
			TotalSkillsFound:       skillOrPluginCount,
			TotalHooksFound:        hookCount,
			ConflictsFound:         len(conflictEntries),
			EstimatedTokenOverhead: totalTokens,
			MemoryStoresFound:      len(memEntries),
			TrustWarnings:          len(trustEntries),
		},
		Discover:    discoverEntries,
		Conflicts:   conflictEntries,
		TokenCost:   tokenEntries,
		MemoryAudit: memEntries,
		TrustReport: trustEntries,
	}
}
