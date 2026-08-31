---
name: gstack-context-bill
description: Tracks estimated token cost of everything loaded into context this session and warns when cumulative always-on overhead crosses a configured budget threshold.
---

# gstack-context-bill

Runs a running tally of context budget spent on always-on skill
instructions versus on-demand content loaded during the session.

## What it tracks

- Frontmatter + body size of every always-on skill, converted to an
  estimated token count.
- A running total compared against `budget.warn_threshold` in
  `~/.gstack/config.yaml`.
- A per-source breakdown so a specific noisy skill can be identified
  and trimmed or moved to on-demand loading.

## What it does not do

It does not call the official token-counting API by default - the
estimate is a character-ratio approximation, same caveat as any
other char-based estimator. Exact counts require an explicit opt-in
flag that sends data off-machine.
