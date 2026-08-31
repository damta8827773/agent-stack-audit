package memoryaudit

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/damtafaiz/agent-stack-audit/internal/config"
)

func TestAudit_MissingTargetIsSkipped(t *testing.T) {
	targets := []config.MemoryTarget{
		{System: "claude-mem", Path: filepath.Join(t.TempDir(), "does-not-exist.db")},
	}
	result := NewFSAuditor().Audit(targets)
	if len(result) != 0 {
		t.Fatalf("expected 0 entries for a missing target, got %d", len(result))
	}
}

func TestAudit_ReportsBasicMetadataWithoutReadingContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memory", "notes.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	secretContent := "this is private conversation content that must never appear in findings"
	if err := os.WriteFile(path, []byte(secretContent), 0o644); err != nil {
		t.Fatal(err)
	}

	targets := []config.MemoryTarget{{System: "ECC Memory Vault", Path: path}}
	result := NewFSAuditor().Audit(targets)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	e := result[0]
	if e.System != "ECC Memory Vault" {
		t.Errorf("System = %q, want ECC Memory Vault", e.System)
	}
	if e.SizeBytes != int64(len(secretContent)) {
		t.Errorf("SizeBytes = %d, want %d", e.SizeBytes, len(secretContent))
	}
	if e.LastModified == "" {
		t.Errorf("LastModified should not be empty")
	}
	// The non-negotiable boundary: the struct has no field that could hold
	// file content, and this test's own secretContent proves it never got
	// read into anything the caller could serialize.
}

func TestAudit_SQLiteSchemaOnlyNeverContent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "claude-mem.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE observations (id INTEGER PRIMARY KEY, secret_text TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO observations (secret_text) VALUES (?), (?), (?)`,
		"top secret memory 1", "top secret memory 2", "top secret memory 3"); err != nil {
		t.Fatal(err)
	}
	db.Close()

	targets := []config.MemoryTarget{{System: "claude-mem", Path: dbPath}}
	result := NewFSAuditor().Audit(targets)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	e := result[0]
	if len(e.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d: %+v", len(e.Tables), e.Tables)
	}
	if e.Tables[0].Name != "observations" {
		t.Errorf("table name = %q, want observations", e.Tables[0].Name)
	}
	if e.Tables[0].RowCount != 3 {
		t.Errorf("row count = %d, want 3", e.Tables[0].RowCount)
	}
	// The struct has no field capable of holding "secret_text" values -
	// only Name and RowCount - so there is no way for actual row content
	// to leak through this API even by accident.
}

func TestAudit_NonSQLiteFileIsNotOpenedAsDB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("not a sqlite file"), 0o644); err != nil {
		t.Fatal(err)
	}
	targets := []config.MemoryTarget{{System: "gstack", Path: path}}
	result := NewFSAuditor().Audit(targets)
	if len(result) != 1 || result[0].Tables != nil {
		t.Fatalf("non-.db file should have nil Tables, got: %+v", result)
	}
}

func TestAudit_CorruptDBFileDoesNotCrash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.db")
	if err := os.WriteFile(path, []byte("not actually a sqlite file"), 0o644); err != nil {
		t.Fatal(err)
	}
	targets := []config.MemoryTarget{{System: "claude-mem", Path: path}}
	result := NewFSAuditor().Audit(targets)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry (metadata still reported) even for a corrupt db, got %d", len(result))
	}
	if result[0].Tables != nil {
		t.Errorf("expected nil Tables for a corrupt db, got: %+v", result[0].Tables)
	}
}

func TestAudit_MultipleTargets(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "a.db")
	path2 := filepath.Join(dir, "b", "notes.md")
	os.WriteFile(path1, []byte("x"), 0o644)
	os.MkdirAll(filepath.Dir(path2), 0o755)
	os.WriteFile(path2, []byte("y"), 0o644)

	targets := []config.MemoryTarget{
		{System: "claude-mem", Path: path1},
		{System: "ECC Memory Vault", Path: path2},
		{System: "gstack", Path: filepath.Join(dir, "not-there")},
	}
	result := NewFSAuditor().Audit(targets)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries (missing target skipped), got %d: %+v", len(result), result)
	}
}
