---
name: False positive
about: conflict-check or trust-report flagged something that isn't actually a problem
title: ''
labels: false-positive
---

conflict-check and trust-report are heuristics (pattern matching, not full
static analysis — see docs/LIMITATIONS.md), so false positives are expected
and genuinely useful to track. Please include:

**Finding ID**
e.g. `conflict-001`, `trust-003`

**The skill/hook configuration that triggered it**
Paste the relevant SKILL.md frontmatter, hooks.json entry, or settings.json
entry (redact anything sensitive).

**Why this isn't actually a problem**

