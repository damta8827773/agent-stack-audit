# Capturing Screenshots

The three images in `docs/assets/` must be genuine captures of the tool
actually running - never a mockup of a layout that hasn't been run. Per
CLAUDE.md section 7, they are captured with
[`vhs`](https://github.com/charmbracelet/vhs) (Charm - same ecosystem as
`lipgloss`, already a dependency), from a real, checked-in demo config,
never a hand-edited image and never generative-AI video.

## One-time setup

```
go install github.com/charmbracelet/vhs@latest
```

`vhs` also needs `ffmpeg` on PATH to encode frames:

- Windows: `winget install --id Gyan.FFmpeg.Essentials`
- macOS: `brew install ffmpeg`
- Linux: your package manager's `ffmpeg`

`vhs` also shells out to `ttyd` to host the terminal it records:

- Windows: `winget install --id tsl0922.ttyd`
- macOS/Linux: see the [ttyd README](https://github.com/tsl0922/ttyd)

All three binaries need to be on `PATH` (a fresh shell after `winget
install` picks up the PATH change automatically; the current shell does
not).

**Windows note:** the tapes in `docs/vhs/` deliberately don't set `Set
Shell` and use `cmd.exe` syntax (no `./` prefix, `type` instead of `cat`).
In testing on this platform, explicitly setting `Set Shell "bash"` or
`Set Shell "powershell"` produced a blank recording (ttyd never rendered
a visible prompt) even though the binaries themselves work fine - only
vhs's unset default (which resolves to `cmd.exe` here) actually recorded
real output. If you're on macOS/Linux and want `bash` explicitly, add
`Set Shell "bash"` back and switch the `Type` lines to their `./binary`
/ `cat` form - just verify it actually renders before trusting it.

## The demo config

`testdata/demo/claude-config/` is a small, self-contained,
checked-in fixture tree - not anyone's real `~/.claude`. It reproduces the
real claude-mem/superpowers `SessionStart` matcher collision found during
FASE 0 research (same shape as
`internal/discover/discover_test.go`'s `TestScan_RealConfirmedConflict`),
plus two always-on skills for a non-zero token-cost figure.

`testdata/demo/claude-mem-data/claude-mem.db` is generated, not checked
in (it's a binary SQLite file) - build it once with:

```
go run ./tools/makedemo-db testdata/demo/claude-mem-data/claude-mem.db
```

This gives memory-audit a real `.db` file to report metadata and table
names for, with placeholder row content only - the tool never reads that
content either way, but a demo screenshot showing table names next to a
"row content is never read" claim should be backed by an actual table, not
an empty file.

## Regenerating the three images

From the repo root, after building the binary:

```
go build -o agent-stack-audit.exe ./cmd/agent-stack-audit
vhs docs/vhs/scan-output-terminal.tape
vhs docs/vhs/conflict-example.tape
vhs docs/vhs/report-markdown-sample.tape
```

Each `.tape` script sets `CLAUDE_CONFIG_DIR` / `CLAUDE_MEM_DATA_DIR` to
point at the demo fixture (never a real machine's config), runs the
actual binary, and takes a `Screenshot` into `docs/assets/`. The `.gif`
files each tape also produces under `docs/vhs/` are scratch output
(gitignored) - only the `.png` screenshots are committed.

## If you need to change what the demo shows

Edit the fixture files under `testdata/demo/claude-config/`, not the
`.tape` scripts - the tapes just run `scan` and screenshot the result,
they don't construct any of the example data themselves.
