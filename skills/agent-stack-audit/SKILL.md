---
name: agent-stack-audit
description: Read-only audit of installed Claude Code skills, plugins, and hooks - reports conflicting hook registrations, estimated token overhead from always-on skills, local memory-store metadata, basic trust findings, and (opt-in, network) known dependency vulnerabilities via OSV.dev. Use when the user asks to audit, check, or list what skills/plugins/hooks are installed, whether any hooks conflict, how much context budget always-on skills are costing, what memory stores exist locally, or whether any dependency has a known CVE.
---

# agent-stack-audit

Runs the `agent-stack-audit` binary against this machine's Claude Code
config and reports what it finds. Read-only - it never modifies any skill,
plugin, hook, or memory-store file it discovers.

## When to use this skill

- "What skills/plugins do I have installed?"
- "Do any of my hooks conflict with each other?"
- "How much token overhead are my always-on skills costing?"
- "What memory stores are on this machine, and are they safe to leave
  unattended?"

## How to run it

1. Check the binary is available: `agent-stack-audit version`. If not
   found, build it from this repo with `go build ./cmd/agent-stack-audit`
   or `go install github.com/damta8827773/agent-stack-audit/cmd/agent-stack-audit@latest`.
2. Run `agent-stack-audit scan`. This writes `report.md` and `report.json`
   to `./audit-report/` and prints a terminal summary.
3. Read `audit-report/report.md` back and summarize the findings for the
   user - lead with any `CONFIRMED` conflicts or trust findings, since
   those are the highest-confidence results (see the confidence-level
   scheme in the main README).
4. If the user only cares about one thing (e.g. just token overhead), use
   `agent-stack-audit scan --only token-cost` instead of a full scan.
5. If the user asks about known vulnerabilities in dependencies, use
   `agent-stack-audit scan --vuln-check` - this is the ONE flag that makes
   a network call (to osv.dev, package name+version only). It always
   prints a consent prompt and waits for a `y` answer; never pipe an
   automatic "y" into it or assume consent on the user's behalf - let the
   user see the prompt and answer it themselves, or explicitly confirm
   with you first that they want it to proceed.

## What NOT to do

Do not edit, delete, or "fix" anything this tool reports without the user
explicitly asking - it's a read-only auditor by design. `scan --fix` exists
as a v0.2 preview, but it only ever prints suggestions and, after an
explicit `y` confirmation, writes them to `<destination>/suggested-fixes.md`
- it never edits a plugin/skill's own files itself. Applying a suggestion
is always a manual step the user takes elsewhere; don't do it on their
behalf without being asked. Do not treat `LIKELY` or `INFORMATIONAL`
findings as confirmed problems; report them with their actual confidence
level.
