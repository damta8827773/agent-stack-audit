// Package memoryaudit reports metadata about local memory/state stores -
// path, size, last-write time, and whether the permission bits allow
// group/other to read it. It never opens a store to read actual record
// content; for SQLite files it may open a connection strictly to list
// table names and row counts (FASE 6 point 3), never SELECT * on content.
// This boundary is non-negotiable per the project's design principle 2.
package memoryaudit

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/damtafaiz/agent-stack-audit/internal/config"
)

// TableInfo is schema-only: a table's name and row count, never its rows.
type TableInfo struct {
	Name     string
	RowCount int64
}

type Entry struct {
	System        string
	Path          string
	SizeBytes     int64
	LastModified  string // RFC3339
	WorldReadable bool
	Tables        []TableInfo // populated only for .db files that opened cleanly
}

type Auditor interface {
	Audit(targets []config.MemoryTarget) []Entry
}

type FSAuditor struct{}

func NewFSAuditor() *FSAuditor { return &FSAuditor{} }

// Audit reports metadata for each target that exists. A target that
// doesn't exist (the store isn't installed) is skipped, not an error -
// mirroring discover's "missing dir is not an error" philosophy, since
// most machines won't have every known memory system installed.
func (a *FSAuditor) Audit(targets []config.MemoryTarget) []Entry {
	var result []Entry
	for _, target := range targets {
		path := expandHome(target.Path)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		entry := Entry{
			System:        target.System,
			Path:          path,
			SizeBytes:     info.Size(),
			LastModified:  info.ModTime().UTC().Format(time.RFC3339),
			WorldReadable: isWorldReadable(info),
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".db") {
			entry.Tables = readSchema(path)
		}

		result = append(result, entry)
	}
	return result
}

// isWorldReadable checks whether group or other read bits are set.
// Meaningful on Unix; Windows has no equivalent permission-bit model (Go
// synthesizes a fixed mode there that would false-flag ordinary files), so
// the check is skipped - not silently claimed as a real cross-platform ACL
// check - on Windows.
func isWorldReadable(info os.FileInfo) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	return info.Mode().Perm()&0o044 != 0
}

// readSchema opens a read-only connection strictly to list table names and
// row counts. Any failure (locked file, not a valid SQLite file, corrupt
// header) is swallowed - this is best-effort enrichment, not something
// memory-audit's core metadata contract depends on.
func readSchema(path string) []TableInfo {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&immutable=1")
	if err != nil {
		return nil
	}
	defer db.Close()

	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			names = append(names, name)
		}
	}

	var tables []TableInfo
	for _, name := range names {
		var count int64
		// #nosec G202 -- name comes from sqlite_master, not user input, and
		// is quoted defensively below; parameterized queries can't bind
		// identifiers in database/sql.
		if err := db.QueryRow(`SELECT COUNT(*) FROM "` + strings.ReplaceAll(name, `"`, `""`) + `"`).Scan(&count); err != nil {
			continue
		}
		tables = append(tables, TableInfo{Name: name, RowCount: count})
	}
	return tables
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}
