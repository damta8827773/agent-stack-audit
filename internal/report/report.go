// Package report aggregates every module's output into the report.json
// schema (FASE 2 / 8.1) and writes it as Markdown and/or JSON.
package report

type Report struct {
	SchemaVersion string             `json:"schema_version"`
	ScannedAt     string             `json:"scanned_at"`
	Host          string             `json:"host"`
	Summary       Summary            `json:"summary"`
	Discover      []DiscoverEntry    `json:"discover"`
	Conflicts     []ConflictEntry    `json:"conflicts"`
	TokenCost     []TokenCostEntry   `json:"token_cost"`
	MemoryAudit   []MemoryAuditEntry `json:"memory_audit"`
	TrustReport   []TrustReportEntry `json:"trust_report"`
}

type Summary struct {
	TotalSkillsFound       int `json:"total_skills_found"`
	TotalHooksFound        int `json:"total_hooks_found"`
	ConflictsFound         int `json:"conflicts_found"`
	EstimatedTokenOverhead int `json:"estimated_token_overhead"`
	MemoryStoresFound      int `json:"memory_stores_found"`
	TrustWarnings          int `json:"trust_warnings"`
}

type DiscoverEntry struct {
	SourceSystem   string `json:"source_system"`
	Path           string `json:"path"`
	Type           string `json:"type"`
	HasFrontmatter bool   `json:"has_frontmatter"`
	AlwaysOn       bool   `json:"always_on"`
}

type ConflictEntry struct {
	ID         string   `json:"id"`
	Confidence string   `json:"confidence"`
	Event      string   `json:"event"`
	Matcher    string   `json:"matcher"`
	Sources    []string `json:"sources"`
	Detail     string   `json:"detail"`
}

type TokenCostEntry struct {
	SourceSystem     string `json:"source_system"`
	EstimatedTokens  int    `json:"estimated_tokens"`
	AlwaysOn         bool   `json:"always_on"`
	EstimationMethod string `json:"estimation_method"`
}

type MemoryAuditEntry struct {
	System        string      `json:"system"`
	Path          string      `json:"path"`
	SizeBytes     int64       `json:"size_bytes"`
	LastModified  string      `json:"last_modified"`
	WorldReadable bool        `json:"world_readable"`
	Tables        []TableInfo `json:"tables,omitempty"`
}

// TableInfo is schema-only, per FASE 6 point 3: a SQLite table's name and
// row count, never its rows.
type TableInfo struct {
	Name     string `json:"name"`
	RowCount int64  `json:"row_count"`
}

type TrustReportEntry struct {
	ID         string `json:"id"`
	Confidence string `json:"confidence"`
	Finding    string `json:"finding"`
	Path       string `json:"path"`
}

type Writer interface {
	Write(r Report, destination string) error
}
