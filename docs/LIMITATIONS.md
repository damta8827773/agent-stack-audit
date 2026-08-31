# Limitations

This is the honest list of what agent-stack-audit v0.1 does not do, or does
only approximately. README's "What this does NOT do" section is a copy of
this file - keep them in sync.

## conflict-check is pattern matching, not static analysis

Conflicts are detected by splitting each hook's `matcher` string on `|` and
comparing the resulting sets across sources on the same event. This catches
identical and overlapping tool-name matchers (the common real-world case -
e.g. `Write|Edit|MultiEdit` vs `Write|Edit|MultiEdit|NotebookEdit`). It does
**not** understand full regex semantics beyond simple alternation, and it
never inspects what a hook's command actually does. Two hooks registered on
non-overlapping matchers can still race or interfere with each other through
side effects (shared files, shared state) that this tool has no way to see.
Treat CONFIRMED and LIKELY findings as "worth a manual look," not as a
complete list of every possible conflict.

## token-cost is a rough approximation

Token counts use a flat ~4 characters/token ratio
(`estimation_method: "char_ratio_approximate"` in every `token_cost` entry),
not the real tokenizer. Actual token counts vary by content - code,
non-English text, and heavy markdown formatting all shift the real ratio.
gstack's own `gstack-context-bill` calibrates per-content-type divisors
against real `count_tokens` measurements; agent-stack-audit v0.1 doesn't, by
choice - a single documented approximation is more honest than an
uncalibrated attempt at precision. An `--exact` flag that calls the real
`count_tokens` API is on the roadmap, gated behind an explicit egress
warning since it sends file text off the machine.

## World-writable / world-readable checks don't work on Windows

Go synthesizes Unix-style permission bits on Windows in a way that doesn't
reflect real ACL-based access control - every ordinary directory reads back
as if group/other could write to it. Rather than emit a false CONFIRMED
finding, `trust-report`'s world-writable check and `memory-audit`'s
world-readable check are both **skipped entirely on Windows** (they always
report `false`). On Linux and macOS these checks use real POSIX permission
bits and are meaningful.

## Scan depth is capped at 3 levels below each root

`discover` reads directory contents from the scan root down through 3
levels below it (deep enough to reach
`plugins/marketplaces/<vendor>/.claude-plugin/plugin.json`, which is exactly
3 levels below the `plugins/` root). A marketplace layout with an extra
nesting level (e.g. a vendor hosting multiple named plugins in their own
subdirectories) can put a manifest out of reach. This is a deliberate
depth cap, not a bug - unbounded recursion into arbitrary config trees is a
DoS/symlink-loop risk this tool won't take on.

## source_system is inferred, not authoritative

There's no standard field in SKILL.md frontmatter or a hook's matcher entry
that names the parent project. agent-stack-audit infers it, in order of
confidence: (1) a plugin manifest's own `name` field, (2) a small built-in
list of known name prefixes (`ecc-`, `gstack-`, `superpowers`,
`claude-mem`, `watermarks-remover`, `graphify`) matched against the
directory name or hook command string, (3) the literal directory name as a
fallback, or `"unknown"` for a hook whose command references only
`${CLAUDE_PLUGIN_ROOT}` with no other identifying text. A `source_system`
value in the report is a best-effort label, not a verified claim of
ownership.

## v0.1 only scans Claude Code

Codex, Cursor, Gemini CLI, and OpenCode are roadmap items (v0.3+), not
implemented. `CLAUDE_CONFIG_DIR` and `CLAUDE_MEM_DATA_DIR` overrides are
honored, but every other platform-specific location in Lampiran A's
"Roadmap v0.3" rows is not scanned yet - including the mirrored skill
directories that some tools (ECC, gstack) already write to those platforms
today. Scanning those in a future version risks double-counting the same
logical install if both the Claude Code and the mirrored copy get counted
separately; that needs its own dedup design, not a naive path addition.

## memory-audit's SQLite schema read is best-effort

Table names and row counts are read via a read-only connection when a
target file has a `.db` extension. A locked file, a non-SQLite file with a
`.db` extension, or a corrupt database simply produces no `tables` data -
this is never treated as an error, since the metadata (path/size/mtime/
permission) is the part of the contract that's guaranteed, not the schema
enrichment.

## Excludes are exact-path or prefix matches only

`.agent-stack-audit.yml`'s `exclude` list matches an entry's path exactly or
as a path prefix. There's no glob/wildcard support in v0.1 - `skills/foo-*`
won't work, you'd need to list each directory.

## The generic-description check is not AI-content detection

`trust-report` flags a skill/plugin description that leans on generic
marketing vocabulary (the same forbidden-word list this project's own docs
follow, plus a couple of well-known clichés - see
`genericMarketingPhrases` in `internal/trustreport/trustreport.go`). This is
always `INFORMATIONAL`, never higher, and deliberately so: there is no
reliable way to tell whether text was AI-written from surface features
alone, and this tool does not claim otherwise. It flags buzzword-heavy
copy, nothing more - a human can write "seamlessly supercharge your
workflow" and a language model can write plain, specific prose. Don't
read a finding here as an accusation about who or what wrote a skill.

## IP-literal hook calls are a weaker signal than they sound

`trust-report` gives a hook command that calls a raw IP address (IPv4 or
bracketed IPv6) a more specific message than a plain domain call, since
bypassing DNS is a pattern more associated with C2/exfiltration endpoints.
It is still `LIKELY`, not `CONFIRMED` - plenty of legitimate local tooling
calls `http://127.0.0.1:PORT` or a LAN device's IP directly. Review the
actual command before treating this as a real finding.
