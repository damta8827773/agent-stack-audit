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
