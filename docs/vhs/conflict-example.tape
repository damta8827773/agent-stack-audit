# Regenerates docs/assets/conflict-example.png - a focused conflict-check
# run against the same demo config, which deliberately reproduces the real
# claude-mem/superpowers SessionStart matcher collision found during FASE 0
# research (see internal/discover/discover_test.go's
# TestScan_RealConfirmedConflict for the fixture this is modeled on). Run:
#   vhs docs/vhs/conflict-example.tape
# See docs/CAPTURING_SCREENSHOTS.md for why this doesn't set `Set Shell`.

Output docs/vhs/conflict-example.gif

Set FontFamily "Cascadia Code"
Set FontSize 15
Set Width 1100
Set Height 500
Set Padding 20
Set Theme "Dracula"

Env CLAUDE_CONFIG_DIR "testdata/demo/claude-config"
Env CLAUDE_MEM_DATA_DIR "testdata/demo/claude-mem-data"

Sleep 1s
Type ".\agent-stack-audit.exe scan --only discover,conflict-check --destination testdata\demo\out"
Enter
Sleep 5s
Screenshot docs/assets/conflict-example.png
