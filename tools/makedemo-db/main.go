// SPDX-License-Identifier: MIT

// Command makedemo-db generates the placeholder claude-mem.db used by the
// docs/vhs/*.tape screenshot recordings (see docs/CAPTURING_SCREENSHOTS.md).
// Not checked into the repo since it's a generated binary file - run this
// instead. Row content is placeholder text only, never anything real:
// memory-audit itself never reads row content either way (see
// docs/SECURITY_MODEL.md §3), but a demo screenshot showing table names
// next to that claim should be backed by an actual table, not an empty
// file with nothing to demonstrate the claim against.
package main

import (
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: makedemo-db <output-path>")
	}
	path := os.Args[1]
	os.Remove(path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE sessions (id INTEGER PRIMARY KEY, started_at TEXT)`,
		`CREATE TABLE observations (id INTEGER PRIMARY KEY, session_id INTEGER, summary TEXT)`,
		`INSERT INTO sessions (started_at) VALUES ('2026-08-29T09:00:00Z'), ('2026-08-30T14:00:00Z')`,
		`INSERT INTO observations (session_id, summary) VALUES (1, 'demo-only placeholder row'), (1, 'demo-only placeholder row'), (2, 'demo-only placeholder row')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			log.Fatal(err)
		}
	}
}
