# Security Policy

agent-stack-audit is a read-only scanner. It never modifies files it finds,
never uploads data by default, and never reads the content of memory stores
it audits (see [docs/SECURITY_MODEL.md](docs/SECURITY_MODEL.md) for the full
threat model). If you find a way around any of those three guarantees,
that's a security bug, not a feature request.

## Reporting a vulnerability

Please report security issues privately rather than opening a public issue:

- Open a [GitHub Security Advisory](https://github.com/damtafaiz/agent-stack-audit/security/advisories/new)
  on this repo, or
- Email damtafaiz@gmail.com with a description and, if possible, steps to
  reproduce.

We aim to acknowledge reports within 48 hours and to have a fix or a
mitigation plan within 14 days for confirmed issues. You'll get credit in
the release notes unless you ask not to.

## Supported versions

agent-stack-audit is pre-1.0 (see [docs/ROADMAP.md](docs/ROADMAP.md)). Only
the latest tagged release receives security fixes until a 1.0 is cut.

## Scope

In scope: the `agent-stack-audit` binary itself - its scanning logic,
parsing of untrusted config files (SKILL.md, plugin.json, settings.json,
hooks.json), and report generation.

Out of scope: the third-party tools it scans (ECC, gstack, claude-mem,
superpowers, etc.) - report issues in those to their own repositories.
