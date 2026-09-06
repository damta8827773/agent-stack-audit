// SPDX-License-Identifier: MIT

// Package auditlog implements the append-only, hash-chained scan history
// described in CLAUDE.md section 3 ("Audit log & integritas"):
// ~/.agent-stack-audit/audit-log.jsonl, one line per scan, each line
// carrying prev_hash and entry_hash (SHA-256) so the chain lets Verify
// detect a tampered or removed line. This is detection, not prevention -
// nothing that writes to a local, user-writable file can prevent someone
// with filesystem access from editing it; the hash chain only makes that
// edit noticeable afterward, same as any other append-only ledger design
// (this package's shape is a reference to gstack-egress's pattern, not
// shared code).
//
// Entries are summary-only (module counts, never findings content) - the
// log's job is answering "did anything change between scans", not
// keeping a second copy of report.json.
package auditlog

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Summary mirrors report.Summary's fields. Duplicated rather than
// imported: auditlog has no other reason to depend on the report
// package, and the log's schema should be free to diverge from
// report.json's schema_version independently over time.
type Summary struct {
	TotalSkillsFound       int `json:"total_skills_found"`
	TotalHooksFound        int `json:"total_hooks_found"`
	ConflictsFound         int `json:"conflicts_found"`
	EstimatedTokenOverhead int `json:"estimated_token_overhead"`
	MemoryStoresFound      int `json:"memory_stores_found"`
	TrustWarnings          int `json:"trust_warnings"`
	VulnerabilitiesFound   int `json:"vulnerabilities_found"`
}

// Entry is one scan's record in the log.
type Entry struct {
	Timestamp     string   `json:"timestamp"` // RFC3339
	Host          string   `json:"host"`
	Version       string   `json:"version"`
	SchemaVersion string   `json:"schema_version"`
	ModulesRun    []string `json:"modules_run"`
	Summary       Summary  `json:"summary"`
	PrevHash      string   `json:"prev_hash"` // "" for the first entry in the log
	EntryHash     string   `json:"entry_hash"`
}

// Append reads the log's current last entry (if any) to chain from it,
// fills in PrevHash/EntryHash on e, and appends one JSON line. Creates
// the parent directory if it doesn't exist yet.
func Append(path string, e Entry) error {
	last, err := lastEntry(path)
	if err != nil {
		return err
	}
	if last != nil {
		e.PrevHash = last.EntryHash
	} else {
		e.PrevHash = ""
	}
	e.EntryHash = hashEntry(e)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

// hashEntry hashes the entry's canonical JSON encoding with entry_hash
// itself cleared first, so the hash never depends on its own value.
// encoding/json marshals struct fields in declaration order (not map
// iteration order), so this is reproducible across runs and platforms.
func hashEntry(e Entry) string {
	e.EntryHash = ""
	canonical, _ := json.Marshal(e)
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

func lastEntry(path string) (*Entry, error) {
	entries, err := ReadAll(path)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	return &entries[len(entries)-1], nil
}

// ReadAll reads every entry in file order. A log that doesn't exist yet
// (no scan has ever run) returns an empty slice, not an error.
func ReadAll(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(raw, &e); err != nil {
			return entries, fmt.Errorf("malformed line %d: %w", line, err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return entries, err
	}
	return entries, nil
}

// VerifyResult reports whether the chain is intact end to end.
type VerifyResult struct {
	TotalEntries int
	OK           bool
	BrokenLine   int // 1-indexed; 0 when OK
	Reason       string
}

// Verify recomputes every entry's hash and checks both that it matches
// the entry's own stored entry_hash (catches an edited entry) and that
// it matches the next entry's prev_hash (catches a deleted, reordered,
// or inserted entry).
func Verify(path string) (VerifyResult, error) {
	entries, err := ReadAll(path)
	if err != nil {
		return VerifyResult{}, err
	}

	prevHash := ""
	for i, e := range entries {
		if e.PrevHash != prevHash {
			return VerifyResult{
				TotalEntries: len(entries),
				BrokenLine:   i + 1,
				Reason:       "prev_hash tidak cocok dengan entry_hash entri sebelumnya",
			}, nil
		}
		if want := hashEntry(e); e.EntryHash != want {
			return VerifyResult{
				TotalEntries: len(entries),
				BrokenLine:   i + 1,
				Reason:       "entry_hash tidak cocok dengan isi entri ini sendiri",
			}, nil
		}
		prevHash = e.EntryHash
	}

	return VerifyResult{TotalEntries: len(entries), OK: true}, nil
}
