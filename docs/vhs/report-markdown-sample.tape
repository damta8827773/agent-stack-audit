# Regenerates docs/assets/report-markdown-sample.png - the real report.md
# a scan against the demo config produces, displayed in-terminal. Run:
#   vhs docs/vhs/report-markdown-sample.tape
# See docs/CAPTURING_SCREENSHOTS.md for why this doesn't set `Set Shell`.

Output docs/vhs/report-markdown-sample.gif

Set FontFamily "Cascadia Code"
Set FontSize 14
Set Width 1000
Set Height 1600
Set Padding 20
Set Theme "Dracula"

Env CLAUDE_CONFIG_DIR "testdata/demo/claude-config"
Env CLAUDE_MEM_DATA_DIR "testdata/demo/claude-mem-data"

Sleep 1s
Type ".\agent-stack-audit.exe scan --quiet --destination testdata\demo\out & type testdata\demo\out\report.md"
Enter
Sleep 5s
Screenshot docs/assets/report-markdown-sample.png
