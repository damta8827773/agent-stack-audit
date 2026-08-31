# Contributing

## Local setup

```
go build ./...
go test ./...
```

No CGO toolchain is required - `modernc.org/sqlite` is pure Go.

## Adding a test fixture

Module unit tests build their own fixture directories with `t.TempDir()` at
test time rather than checking in static fixture trees - see
`internal/discover/discover_test.go` for the pattern (`writeFile` helper +
inline content). If you need a fixture that's reused across a module's
tests or across module boundaries, put it under `testdata/fixtures/` instead
and document what real-world install it's modeling.

Never point a test at your own real `~/.claude` directory - fixtures must be
self-contained so tests are reproducible on any machine.

## Commit messages

[Conventional Commits](https://www.conventionalcommits.org/): `feat:`,
`fix:`, `docs:`, `test:`, `refactor:`, `chore:`.

## Pull requests

- New logic needs a test. `go vet ./...` and `gofmt -l .` must both be
  clean.
- Changes to `internal/memoryaudit` need two reviewer approvals, not one -
  it's the module with the hard non-negotiable constraint (never read
  memory-store content), so a second pair of eyes matters more there than
  anywhere else in the codebase.
- If you're changing the `report.json` schema, bump `schema_version` and
  say so explicitly in the PR description - see
  [Lampiran I / versioning policy](docs/ROADMAP.md).

## Reporting a false positive

If `conflict-check` or `trust-report` flags something that isn't actually a
problem, open an issue with the `false_positive` template - include the
finding ID (e.g. `conflict-001`), the skill/hook configuration that
triggered it, and why you believe it's not a real issue. conflict-check is
explicitly a heuristic (pattern matching on matcher strings, not full static
analysis - see [docs/LIMITATIONS.md](docs/LIMITATIONS.md)), so false
positives are expected and useful to track.
