# Roadmap

## Versioning

[Semantic Versioning](https://semver.org/). Pre-1.0, the CLI surface may
still change without a MAJOR bump, but every such change is called out in
[CHANGELOG.md](../CHANGELOG.md). A breaking change to `report.json` (a
field removed or its type changed) always bumps `schema_version` inside the
JSON itself, regardless of what the binary's own version number does.

## v0.1 - dogfood, then soft launch

- Ship all 5 modules + aggregator, tested against real fixtures.
- Run it against the maintainer's own Claude Code setup for at least a week
  of real use; record actual findings (not invented scenarios) for the
  README and the first public write-up.
- Push to GitHub with a README that includes real captured output - no
  promotion push yet.

## v0.2

- `--fix` flag: opt-in, interactive confirmation per change, never silent.
  Scoped narrowly at first (e.g. suggesting a matcher split for a CONFIRMED
  hook conflict) rather than attempting broad auto-remediation.
- `--exact` token counting via the real `count_tokens` API - gated behind
  an explicit egress warning before anything leaves the machine, following
  gstack's egress-receipt pattern as a design reference (not shared code).
- Per-content-type token divisors (calibrated against `count_tokens`
  output), replacing or supplementing the flat char-ratio estimate - only
  if it can be done without overclaiming precision the estimate doesn't
  have.
- Fuzz testing for the YAML/JSON parsers in `discover`.
- Revisit whether `graphify-out/` (project-level generated output) belongs
  in the discover scan - deferred out of v0.1 because it's generated
  output, not installed skill config.

## v0.3+

- Additional platforms: Codex CLI (`${CODEX_HOME:-~/.codex}/skills/`),
  Cursor (`~/.cursor/skills/`), OpenCode
  (`~/.config/opencode/skills/`), Gemini CLI (`~/.gemini/`).
- Dedup design for tools (ECC, gstack) that already mirror their own
  skills into those other platforms' directories today - counting both the
  Claude Code copy and the mirrored copy as separate installs would
  overstate findings.
- Opt-in `--check-updates` network call to check for a newer release -
  explicit flag, never automatic during `scan`.

## Explicitly not planned

Auto-fixing anything without confirmation, default-on telemetry, or
support for any platform whose config format isn't publicly documented
enough to parse honestly (guessing at an undocumented format and labeling
findings CONFIRMED would violate the confidence-level principle this whole
tool is built on).
