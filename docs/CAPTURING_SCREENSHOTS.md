# Capturing Screenshots

The three images in `docs/assets/` must be genuine captures of the tool
actually running - never a mockup of a layout that hasn't been run. If you
regenerate them, follow this process.

## macOS

```
agent-stack-audit scan --tui
```
then `Cmd+Shift+4` to select a region, or `screencapture -w shot.png` and
click the terminal window.

## Linux

`scrot -s scan-output-terminal.png` (select the window with the mouse), or
GNOME: `gnome-screenshot -w -f scan-output-terminal.png`.

## Windows

Windows PowerShell 5.1's default console host (legacy conhost with the
raster font) mangles multi-byte UTF-8 characters like the em dash in
piped/formatted text - `Get-Content`, `cmd /c type`, `chcp 65001` alone
none of it fixes this reliably. Two options that actually work:

- **`Win+Shift+S`** (Snipping Tool) after running the command in Windows
  Terminal (which renders UTF-8 correctly) - the simplest path if you have
  Windows Terminal installed.
- **Notepad** for `report.md` specifically: `notepad report.md` renders
  Unicode correctly regardless of console quirks, and FASE 13 explicitly
  allows an editor screenshot for the Markdown sample, not just a terminal.
- If you must automate it (as this repo's own screenshots were produced):
  `Add-Type -AssemblyName System.Drawing` + `Graphics.CopyFromScreen` after
  positioning/focusing the window with a small P/Invoke `user32.dll`
  wrapper (`SetForegroundWindow`, `MoveWindow`). A freshly-restored window
  needs ~1-2 seconds before it repaints - capturing immediately after
  `ShowWindow`/`MoveWindow` can grab a blank frame.

## What to capture

1. `docs/assets/scan-output-terminal.png` - `agent-stack-audit scan`
   against a real (or realistic) config, showing the summary block.
2. `docs/assets/conflict-example.png` - run against a fixture with a
   deliberate hook conflict (see `internal/discover/discover_test.go`'s
   `TestScan_RealConfirmedConflict` for the exact fixture shape - it
   reproduces the real claude-mem/superpowers `SessionStart` conflict found
   during FASE 0 research) so there's a CONFIRMED finding to show.
3. `docs/assets/report-markdown-sample.png` - the resulting `report.md`,
   opened in an editor or `cat`/`type` in a terminal that renders UTF-8
   correctly.

Crop to just the window content - no need to show the whole desktop.
