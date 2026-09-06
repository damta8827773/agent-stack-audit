# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); this project uses
[Semantic Versioning](https://semver.org/) once it reaches 1.0 - until then,
v0.x may change the CLI surface without a MAJOR bump, but every such change
is called out here.

## [Unreleased]

### Added
- Initial implementation: `discover`, `conflict-check`, `token-cost`,
  `memory-audit`, `trust-report` modules, Markdown/JSON report writers,
  terminal summary rendering, and the `scan` / `init-config` / `version`
  CLI commands.
- `report.json` schema v1.0.
- `scan --fix`: manual suggestions for CONFIRMED conflicts, written to
  `suggested-fixes.md` only after explicit confirmation.
- `scan --vuln-check`: matches Go/npm/PyPI dependency manifests against
  OSV.dev's public advisory database. The only network-touching flag;
  always prompts for consent first. New `vuln_audit` array in
  `report.json` and a new `vulnerabilities_found` summary field.
- `trust-report`: base64-blob and decode-then-execute (`eval`/`exec`/
  `base64 -d`/`atob`) pattern detection in hook commands and script
  content.
- `token-cost`: separate char-ratio divisors for YAML frontmatter vs.
  body content (`estimation_method` renamed to
  `char_ratio_per_content_type_approximate`).
- Fuzz tests for `discover`'s YAML/JSON parsers, run as a permanent CI job.
- `.github/CODEOWNERS`, `.github/AI_AGENT_NOTICE.md`, `NOTICE`, SPDX
  license headers on every `.go` file, and a documented alignment with
  NIST Cybersecurity Framework 2.0 (see docs/ARCHITECTURE.md).
- Branch protection enabled on `main`: PR + 1 approval + all 9 CI checks
  required, `enforce_admins: false` (single-maintainer bypass - see
  docs/SECURITY_MODEL.md §5).
- `docs/assets/*.png` are now regenerated with `vhs` (Charm) against a
  checked-in demo fixture (`testdata/demo/`) instead of manual OS
  screenshot tools - see docs/CAPTURING_SCREENSHOTS.md. Never
  generative-AI video/images for product demos (CLAUDE.md section 2).
- `internal/auditlog`: every scan appends a summary-only, SHA-256
  hash-chained entry to `~/.agent-stack-audit/audit-log.jsonl` (override
  with `AGENT_STACK_AUDIT_HOME`). New `agent-stack-audit verify-log`
  command recomputes the chain and reports the first tampered/deleted
  entry; new `agent-stack-audit diff` command shows the delta between
  the two most recent scans. Detects tampering, does not prevent it -
  see docs/SECURITY_MODEL.md section 6.
- `vuln-audit`: three more ecosystems parsed - `Pipfile.lock` (PyPI),
  `composer.lock`/`composer.json` (Packagist), `Gemfile.lock`
  (RubyGems) - alongside the existing Go/npm/PyPI support, completing
  Lampiran K.3's five-ecosystem scope. Verified live against OSV.dev
  during development (`rack 2.0.6` via a `Gemfile.lock` fixture
  returned 35 real GHSA advisories).
- Moved `CLAUDE.md` into the repo it governs - it was previously one
  directory up, outside any git repository, despite its own header
  claiming to be "checked into the codebase." Also fixed a dangling
  reference to a file (`agent-stack-audit-spec.md`) that never existed.
