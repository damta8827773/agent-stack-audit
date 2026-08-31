---
name: ecc-tdd-workflow
description: Enforces a strict red-green-refactor TDD loop for every code change - writes a failing test first, confirms it fails for the right reason, then implements the minimal code to pass it, then refactors with tests green throughout.
---

# ecc-tdd-workflow

This skill is always loaded at session start. Before writing any
implementation code, it requires:

1. A failing test that demonstrates the missing behavior.
2. Confirmation the test fails with the expected error, not a
   collateral one (import error, syntax error, wrong file).
3. The minimal implementation needed to make that one test pass -
   no speculative extra branches, no handling for cases the test
   doesn't cover yet.
4. A refactor pass with the full suite green, only if the
   implementation introduced duplication or an unclear name.

## When this applies

Any change to application logic - not documentation-only edits,
not pure config changes, not generated code.

## Failure modes to avoid

- Writing the implementation before the test ("test-after" is not
  TDD, it's documentation of what you already built).
- Writing a test that passes on the first run without the
  implementation - it isn't testing anything yet.
- Skipping the refactor step because "it works" - working and
  clean are different bars.
