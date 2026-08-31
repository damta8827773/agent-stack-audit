# Security Model

## 1. Minimal permissions

agent-stack-audit only needs READ access to the config directories it scans
(`~/.claude/skills`, `~/.claude/plugins`, `~/.claude/settings.json`, the
project-level `.claude/` equivalents, plus the known memory-store
locations). It never writes to, moves, or deletes anything it finds. If a
future `--fix` flag is added (not before v0.2, per the design principles),
it will be opt-in and require interactive confirmation for every individual
change - never a silent bulk operation.

## 2. No data leaves the machine

The default build makes zero network calls. Every module in v0.1 -
`discover`, `conflict-check`, `token-cost`, `memory-audit`, `trust-report` -
operates entirely on the local filesystem. Planned future features that do
need network access (`--check-updates` to check for a newer release,
`--exact` to call the real `count_tokens` API for precise token counts)
will each require their own explicit flag and print an egress warning
before sending anything, following the egress-receipt pattern gstack uses
for the same problem - the design pattern is a reference point, not shared
code; agent-stack-audit's own implementation (if and when built) will be
independent.

## 3. Memory content is never read

This is the single most important guarantee in the codebase.
`internal/memoryaudit` reports only filesystem metadata (path, size,
last-modified time, permission bits) for every known memory store, plus -
for SQLite files specifically - table names and row counts read via a
schema-only query (`SELECT name FROM sqlite_master ...` and
`SELECT COUNT(*) FROM <table>`, never `SELECT *`). The `memoryaudit.Entry`
and `TableInfo` types have no field capable of holding row content; there is
no code path from a memory store's actual data to anything this tool prints
or writes. A change that adds such a field, or that reads rows instead of
counting them, is a critical bug regardless of how it's justified - see
`internal/memoryaudit/memoryaudit_test.go` for tests that plant a known
"secret" value in a fixture DB and assert it never surfaces in the result.

## 4. Reporting a security bug

See [SECURITY.md](../SECURITY.md) for the reporting process (GitHub Security
Advisory or email, target 48-hour acknowledgment).
