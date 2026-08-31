# Regenerates docs/assets/scan-output-terminal.png from a real run against
# the checked-in demo config (testdata/demo/), not a mockup. Run from the
# repo root:
#   vhs docs/vhs/scan-output-terminal.tape
# Requires the agent-stack-audit binary built and on PATH first:
#   go build -o agent-stack-audit.exe ./cmd/agent-stack-audit
# See docs/CAPTURING_SCREENSHOTS.md for why this doesn't set `Set Shell`.

Output docs/vhs/scan-output-terminal.gif

Set FontFamily "Cascadia Code"
Set FontSize 15
Set Width 1100
Set Height 650
Set Padding 20
Set Theme "Dracula"

Env CLAUDE_CONFIG_DIR "testdata/demo/claude-config"
Env CLAUDE_MEM_DATA_DIR "testdata/demo/claude-mem-data"

Sleep 1s
Type ".\agent-stack-audit.exe scan --destination testdata\demo\out"
Enter
Sleep 5s
Screenshot docs/assets/scan-output-terminal.png
