# Dependencies

Kept deliberately minimal per the project's stack constraints (no
HTML/CSS/JS anywhere, including transitively). Three direct dependencies:

| Module | Why |
|---|---|
| `gopkg.in/yaml.v3` | Parses SKILL.md YAML frontmatter and `.agent-stack-audit.yml`. Required by the stack spec directly. |
| `modernc.org/sqlite` | Pure-Go SQLite driver for `memory-audit`'s schema-only read (table names + row counts, never row content) of claude-mem's `.db` file. Chosen over `github.com/mattn/go-sqlite3` specifically because it needs no CGO - keeps `go build` working with just the Go toolchain on every FASE 11 target platform (linux/amd64, darwin/arm64, windows/amd64) without a C compiler. |
| `github.com/charmbracelet/lipgloss` | Terminal styling for `internal/tui` - colors, borders, and automatic downgrade to plain text on non-TTY output (satisfies the "fallback to plain text" requirement in FASE 8 without extra detection code). `bubbletea` was left out: FASE 1 lists it as optional, and the scan summary is a static render, not an interactive event loop. |

Everything else in `go.sum` is a transitive dependency pulled in by these
three (mostly `modernc.org/*` support packages for the pure-Go SQLite
implementation, and `charmbracelet/x/*` / `muesli/termenv` for terminal
capability detection). None of them touch the network, and none require
CGO - `go build` works with a stock Go toolchain on every target platform.

No dependency in this tree pulls in HTML, CSS, or JavaScript/TypeScript, or
requires Node.js or a browser to build or run.
