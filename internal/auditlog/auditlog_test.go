// SPDX-License-Identifier: MIT

package auditlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleEntry(conflicts int) Entry {
	return Entry{
		Timestamp:     "2026-08-31T00:00:00Z",
		Host:          "windows-amd64",
		Version:       "0.1.0",
		SchemaVersion: "1.0",
		ModulesRun:    []string{"discover", "conflict-check"},
		Summary:       Summary{ConflictsFound: conflicts},
	}
}

func TestAppend_FirstEntryHasEmptyPrevHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit-log.jsonl")
	if err := Append(path, sampleEntry(0)); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadAll(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].PrevHash != "" {
		t.Errorf("expected empty PrevHash on genesis entry, got %q", entries[0].PrevHash)
	}
	if entries[0].EntryHash == "" {
		t.Error("expected a non-empty EntryHash")
	}
}

func TestAppend_ChainsToPreviousEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit-log.jsonl")
	if err := Append(path, sampleEntry(0)); err != nil {
		t.Fatal(err)
	}
	if err := Append(path, sampleEntry(1)); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadAll(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[1].PrevHash != entries[0].EntryHash {
		t.Errorf("second entry's PrevHash = %q, want %q (first entry's EntryHash)", entries[1].PrevHash, entries[0].EntryHash)
	}
}

func TestAppend_CreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "audit-log.jsonl")
	if err := Append(path, sampleEntry(0)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestVerify_EmptyLogIsOK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit-log.jsonl") // never created
	result, err := Verify(path)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.TotalEntries != 0 {
		t.Errorf("expected OK with 0 entries, got %+v", result)
	}
}

func TestVerify_IntactChainIsOK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit-log.jsonl")
	for i := 0; i < 5; i++ {
		if err := Append(path, sampleEntry(i)); err != nil {
			t.Fatal(err)
		}
	}
	result, err := Verify(path)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Errorf("expected an intact 5-entry chain to verify OK, got %+v", result)
	}
	if result.TotalEntries != 5 {
		t.Errorf("expected 5 entries, got %d", result.TotalEntries)
	}
}

func TestVerify_DetectsEditedEntryContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit-log.jsonl")
	for i := 0; i < 3; i++ {
		if err := Append(path, sampleEntry(i)); err != nil {
			t.Fatal(err)
		}
	}

	// Tamper with the middle line's content without recomputing its hash -
	// exactly what editing a plain text file by hand would do.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	lines[1] = strings.Replace(lines[1], `"conflicts_found":1`, `"conflicts_found":99`, 1)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Verify(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatal("expected tampering to be detected, got OK")
	}
	if result.BrokenLine != 2 {
		t.Errorf("expected the break reported at line 2, got %d", result.BrokenLine)
	}
}

func TestVerify_DetectsDeletedEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit-log.jsonl")
	for i := 0; i < 3; i++ {
		if err := Append(path, sampleEntry(i)); err != nil {
			t.Fatal(err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	// Remove the middle line entirely - the last line's prev_hash now
	// points at a hash that no longer appears anywhere above it.
	remaining := []string{lines[0], lines[2]}
	if err := os.WriteFile(path, []byte(strings.Join(remaining, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Verify(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatal("expected a deleted middle entry to break the chain, got OK")
	}
	if result.BrokenLine != 2 {
		t.Errorf("expected the break reported at (new) line 2, got %d", result.BrokenLine)
	}
}

func TestReadAll_MissingFileReturnsEmptyNotError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.jsonl")
	entries, err := ReadAll(path)
	if err != nil {
		t.Fatalf("expected no error for a missing log, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadAll_MalformedLineReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit-log.jsonl")
	if err := os.WriteFile(path, []byte("not valid json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadAll(path); err == nil {
		t.Fatal("expected an error for a malformed line, got nil")
	}
}
