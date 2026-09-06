# Security Model

## 1. Minimal permissions

agent-stack-audit only needs READ access to the config directories it scans
(`~/.claude/skills`, `~/.claude/plugins`, `~/.claude/settings.json`, the
project-level `.claude/` equivalents, plus the known memory-store
locations). It never writes to, moves, or deletes anything it finds, except
its own output files (`report.md`/`report.json`/`suggested-fixes.md`, all
under `<destination>`, never a plugin/skill's own files) and its own audit
log (`~/.agent-stack-audit/audit-log.jsonl`, override with
`AGENT_STACK_AUDIT_HOME`). `--fix` is opt-in and requires an explicit `y`
confirmation before writing anything - never a silent bulk operation.

## 2. No data leaves the machine, except one explicit opt-in flag

Every module runs entirely on the local filesystem by default -
`discover`, `conflict-check`, `token-cost`, `memory-audit`, `trust-report`,
and `--fix`'s suggestions never make a network call. The one exception:
`scan --vuln-check` sends package name+version (never file content, never
source code) to OSV.dev to check against known vulnerabilities, and it
always prints exactly what will be sent and waits for an explicit `y`
before doing so - every scan, with no flag to suppress that prompt. See
`internal/vulnaudit` and [docs/LIMITATIONS.md](LIMITATIONS.md).

Still-planned future features that would need network access
(`--check-updates` to check for a newer release, `--exact` to call the
real `count_tokens` API for precise token counts) will each require their
own explicit flag and the same egress-warning treatment, following the
egress-receipt pattern gstack uses for the same problem - the design
pattern is a reference point, not shared code; agent-stack-audit's own
implementation (if and when built) is independent.

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

## 5. Repo integrity (what's real vs. what's just policy)

Anyone can read or fork a public repo - that's how open source works, not
a weakness of this project specifically. What's actually enforceable:

- **CODEOWNERS** (`.github/CODEOWNERS`) requires review on
  `internal/memoryaudit/` and `internal/vulnaudit/` changes - the two
  modules where a bug would matter most (memory content leaking, or an
  unexpected network call).
- `.github/AI_AGENT_NOTICE.md` is explicitly **not** a technical control -
  it's a written policy statement for any AI agent reading this repo,
  documented as such rather than oversold as enforcement.
- `NOTICE` documents attribution expectations for redistribution -
  legal/social convention, not a technical mechanism either.
- **Branch protection on `main`** is enabled: every non-admin push must go
  through a PR with 1 approving review, and all 9 CI checks (`test` x3
  OS, `cross-build` x3 target, `validate-skill`, `fuzz`, `no-em-dash`)
  must pass first. `enforce_admins` is deliberately `false` - with a
  single maintainer, requiring a second reviewer that doesn't exist would
  just block every merge, so the repo owner can still merge their own
  work directly. This is a real, narrower guarantee than "nobody can
  merge without review" - it's "external contributors can't merge without
  review and passing CI"; tighten it (`enforce_admins: true`) once there's
  a second regular contributor who can actually review the owner's PRs.

## 6. The audit log detects tampering, it does not prevent it

`~/.agent-stack-audit/audit-log.jsonl` is a plain, user-writable local
file. Nothing about SHA-256 hashing a chain of entries stops someone with
filesystem access from editing or deleting it - that would require an
external anchor (a remote append-only service, a TPM, a blockchain
timestamp) that this project deliberately doesn't have, since adding one
would mean a network call on every scan, contradicting the "zero network
by default" principle above.

What the hash chain actually buys: `agent-stack-audit verify-log`
recomputes every entry's hash and will notice if any entry was edited
after the fact (its stored `entry_hash` stops matching its own content)
or removed (the next entry's `prev_hash` stops matching anything above
it). See `internal/auditlog/auditlog_test.go`'s
`TestVerify_DetectsEditedEntryContent` and
`TestVerify_DetectsDeletedEntry` for both cases actually exercised, not
just asserted in prose. What it cannot detect: someone deleting the
*entire* file and letting a fresh chain start from nothing - there is no
external record to compare a from-scratch file against. That's a real,
disclosed limit, not an oversight; see
[docs/LIMITATIONS.md](LIMITATIONS.md).

