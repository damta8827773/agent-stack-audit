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

Token counts use hand-picked character-to-token ratios
(`estimation_method: "char_ratio_per_content_type_approximate"` in every
`token_cost` entry) - a YAML frontmatter block is counted at
~3.7 chars/token, the body at ~4.0 chars/token (the well-known rough
English-text approximation), not the real tokenizer. Actual token counts
vary by content - code, non-English text, and heavy markdown formatting all
shift the real ratio well beyond what either number captures. gstack's own
`gstack-context-bill` calibrates its per-content-type divisors against real
`count_tokens` measurements; agent-stack-audit's two ratios are not
calibrated against anything - they're a reasonable guess at the direction
and rough size of the frontmatter/body difference, not a precision claim.
An `--exact` flag that calls the real `count_tokens` API is on the roadmap,
gated behind an explicit egress warning since it sends file text off the
machine.

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

## base64-blob and eval-obfuscation checks are pattern matching, not malware analysis

`trust-report` flags a hook command or script whose content contains a long
(80+ character) base64-looking string, or a "decode-then-execute" idiom
(`base64 -d`, `eval(`, `exec('...')`, `atob(`, `FromBase64String`). This is
string pattern matching on the source text - there is no disassembly,
sandboxing, or behavioral analysis anywhere in this tool, and there never
will be without a fundamentally different (and much larger) engineering
effort. Both checks are LIKELY, never CONFIRMED: a long base64 string can
be a legitimate embedded asset or token, and `eval()` has real, benign
uses. Treat a finding here as "worth opening the file and reading it,"
never as a verdict.

## `--fix` only ever writes its own output file

`scan --fix` prints a manual suggestion for every CONFIRMED conflict and,
only after an explicit `y` confirmation, writes them to
`<destination>/suggested-fixes.md`. It never opens, edits, or even knows
the on-disk location of the plugin/skill files a suggestion refers to -
applying a suggestion is always something you do yourself, in whatever
tool actually owns that config. This isn't a scope agent-stack-audit plans
to grow into: modifying another tool's installed files is exactly the kind
of blast radius design principle 1 (read-only by default) exists to avoid.

## `--vuln-check` only knows what OSV/GHSA have published

`vuln-audit` matches dependency versions found in a skill/plugin's own
manifest (`go.mod`, `package.json`/`package-lock.json`,
`requirements.txt`) against OSV.dev's public advisory database. Three
boundaries that matter:

- **Zero-days aren't in scope.** A vulnerability nobody has published yet
  cannot be detected by version matching against a public database, by
  definition. This tells you about *known, disclosed* issues only.
- **Not SAST/DAST.** There is no analysis of the dependency's actual code
  - no disassembly, no execution, no data-flow analysis. This is purely
  "does this exact version appear in a published advisory's affected
  range," the same technique `npm audit`/`pip-audit`/`govulncheck` use.
- **Not a pentest substitute.** A clean `vuln-audit` result means "no
  known CVE matches the exact versions this tool could parse out of your
  manifests" - it says nothing about custom code, misconfigurations, or
  anything a professional security review would catch.

Five ecosystems are parsed (Go via `go.mod`; npm via
`package-lock.json`/`package.json`; PyPI via `requirements.txt`/
`Pipfile.lock`; Packagist via `composer.lock`/`composer.json`; RubyGems
via `Gemfile.lock`), and even then only exact pinned versions count - a
`package.json` range like `^1.2.3` or a `requirements.txt` line like
`flask>=2.0.0` is skipped rather than guessed at (see
`exactVersion`/`parseRequirementsTxt` in
`internal/vulnaudit/manifest.go`), because querying OSV with a guessed
version would be worse than not checking that package at all.
`Gemfile.lock` and the two lockfiles (`package-lock.json`,
`composer.lock`) don't have this problem - a lockfile's whole purpose is
recording the exact resolved version, so every entry found there is
used as-is. This is
also the only part of agent-stack-audit that ever makes a network call,
and it never does so without printing exactly what will be sent and
waiting for an explicit `y` - every scan, with no flag to silence the
prompt.

## The audit log detects tampering after the fact, it doesn't prevent it

`~/.agent-stack-audit/audit-log.jsonl`'s hash chain (see
`internal/auditlog`, `verify-log`, `diff`) makes an *edited* or *deleted*
entry detectable, because each entry's hash depends on the one before
it. It cannot detect someone deleting the whole file and starting a new
chain from nothing, since there's no copy anywhere else to compare
against - this project has no remote/anchored log, deliberately, since
that would require a network call on every scan. `verify-log` answers
"is this specific file's history internally consistent," not "has this
history ever been reset." See
[docs/SECURITY_MODEL.md](SECURITY_MODEL.md) section 6.
