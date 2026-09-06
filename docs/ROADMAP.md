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

- ~~`--fix` flag~~ **shipped early as a preview.** Scoped exactly as
  planned: CONFIRMED conflicts only, prints a suggestion, asks for
  explicit `y`/`N` confirmation before writing anything, and never touches
  a plugin/skill's own files - it only ever writes
  `<destination>/suggested-fixes.md`, its own output file. See
  `internal/fix`.
- ~~Per-content-type token divisors~~ **shipped early.** Frontmatter and
  body now get separate char-ratio divisors (see
  `internal/tokencost`'s `FrontmatterCharsPerToken` /
  `BodyCharsPerToken`) instead of one flat number - still hand-picked, not
  calibrated against real `count_tokens` output, and the
  `estimation_method` field says so explicitly
  (`char_ratio_per_content_type_approximate`).
- ~~Fuzz testing for the YAML/JSON parsers in `discover`~~ **shipped
  early.** `FuzzParseFrontmatter`, `FuzzExtractHooks`, `FuzzJSONManifest`,
  `FuzzStripBOM` run as a permanent CI job (15s each per run, not
  exhaustive, but catches a parser panic regression).
- `--exact` token counting via the real `count_tokens` API - gated behind
  an explicit egress warning before anything leaves the machine, following
  gstack's egress-receipt pattern as a design reference (not shared code).
  Would let the per-content-type divisors above actually be calibrated
  instead of hand-picked, if built.
- Additional `trust-report` heuristics beyond what shipped early (base64
  blob / eval-obfuscation pattern detection in hook commands and script
  content, see `internal/trustreport`): still LIKELY-only, still pattern
  matching, not a claim of actual malware analysis - see
  [docs/LIMITATIONS.md](LIMITATIONS.md).
- ~~`vuln-audit` module~~ **shipped early as a preview** (`--vuln-check`).
  Matches Go/npm/PyPI/Packagist/RubyGems dependency manifests against
  OSV.dev - version matching against a public advisory database, not
  SAST/DAST, not zero-day detection. The only module that ever makes a
  network call, and it always prompts for consent first, every scan. See
  `internal/vulnaudit` and [docs/LIMITATIONS.md](LIMITATIONS.md).
  - ~~`Pipfile.lock`, `composer.json`/`composer.lock`, `Gemfile.lock`
    parsing~~ **shipped**, completing Lampiran K.3's five-ecosystem
    scope. `Gemfile.lock` parsing verified live end to end against
    OSV.dev during development (`rack 2.0.6` - 35 real GHSA advisories
    returned with correct severities and fix versions), the same way
    the original Go/npm/PyPI path was verified against `lodash`/
    `minimist` when vuln-audit first shipped.
  - ~~an append-only hash-chained audit log~~ **shipped.** Every scan
    appends a summary-only entry to
    `~/.agent-stack-audit/audit-log.jsonl` (pattern adapted from
    `gstack-egress`, not shared code), chained via SHA-256
    `prev_hash`/`entry_hash`. `agent-stack-audit verify-log` recomputes
    the chain and reports the first tampered/deleted entry, if any;
    `agent-stack-audit diff` shows the delta between the two most recent
    scans, and flags when the two scans ran different `--only` module
    sets so the delta isn't read as more meaningful than it is. See
    `internal/auditlog` and
    [docs/SECURITY_MODEL.md](SECURITY_MODEL.md) section 6 for what this
    does and, just as importantly, does not guarantee (detection, not
    prevention - a from-scratch deleted file isn't detectable).
  - **Not yet shipped** (still real v0.2+ scope, larger and riskier to
    rush): scheduled daily runs via OS-native schedulers (cron/systemd
    timer/launchd/Task Scheduler - never a hidden background daemon),
    and log retention/rotation to `.jsonl.gz` after 90 days (the log
    just grows unbounded for now - correct but not yet size-managed).
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
- Code-signed release binaries (requires a paid certificate) to reduce
  Windows SmartScreen/Defender false-positive warnings on downloaded
  releases - see [docs/FAQ.md](FAQ.md). `go install` (source build)
  already sidesteps this entirely and stays the primary recommended
  install path either way.

## Repo integrity (Lampiran L) - partially done

Shipped: `.github/CODEOWNERS` (review required on `internal/memoryaudit`
and `internal/vulnaudit`), `.github/AI_AGENT_NOTICE.md`, `NOTICE`, SPDX
headers on every `.go` file, and branch protection on `main` (PR + 1
approval + all 9 CI checks required, `enforce_admins: false` so the
single maintainer can still merge their own work - see
[docs/SECURITY_MODEL.md](SECURITY_MODEL.md) §5).

Not yet shipped:
- Tightening branch protection to `enforce_admins: true` once there's a
  second regular contributor who can review the owner's own PRs.
- Signed commits (GPG or Sigstore/`gitsign`) requirement - documented
  intent, not yet enforced.
- `cosign`-signed release checksums and a CycloneDX SBOM per release -
  needs real Sigstore/OIDC wiring in `release.yml`, not just docs.

## Explicitly not planned

Auto-fixing anything without confirmation, default-on telemetry, or
support for any platform whose config format isn't publicly documented
enough to parse honestly (guessing at an undocumented format and labeling
findings CONFIRMED would violate the confidence-level principle this whole
tool is built on).
