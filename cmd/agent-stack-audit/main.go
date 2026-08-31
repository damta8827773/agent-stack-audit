// Command agent-stack-audit is a read-only CLI that scans installed Claude
// Code skills, plugins, and hooks and reports conflicts, token overhead,
// memory-store metadata, and basic trust findings. It never modifies
// anything on its own.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/damta8827773/agent-stack-audit/internal/config"
	"github.com/damta8827773/agent-stack-audit/internal/conflict"
	"github.com/damta8827773/agent-stack-audit/internal/discover"
	"github.com/damta8827773/agent-stack-audit/internal/fix"
	"github.com/damta8827773/agent-stack-audit/internal/memoryaudit"
	"github.com/damta8827773/agent-stack-audit/internal/report"
	"github.com/damta8827773/agent-stack-audit/internal/tokencost"
	"github.com/damta8827773/agent-stack-audit/internal/trustreport"
	"github.com/damta8827773/agent-stack-audit/internal/tui"
	"github.com/damta8827773/agent-stack-audit/internal/version"
)

func main() {
	os.Exit(Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// Run is the whole CLI, parameterized on args/stdin/stdout/stderr so it can
// be exercised directly from tests without touching the real process
// streams (stdin matters for --fix's interactive confirmation prompt).
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 2
	}

	switch args[0] {
	case "scan":
		return runScan(args[1:], stdin, stdout, stderr)
	case "init-config":
		return runInitConfig(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "agent-stack-audit v%s\n", version.Version)
		return 0
	case "-h", "--help", "help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `agent-stack-audit - read-only audit of installed Claude Code skills, plugins, and hooks

Usage:
  agent-stack-audit scan [flags]
  agent-stack-audit init-config
  agent-stack-audit version

Scan flags:
  --only <modules>      comma-separated: discover,conflict-check,token-cost,memory-audit,trust-report
  --format <fmt>        markdown | json | both (default: both)
  --tui                 force interactive terminal summary
  --quiet               no terminal output, only write files
  --verbose             log every skipped/errored path
  --config <path>       path to .agent-stack-audit.yml (default: .agent-stack-audit.yml)
  --destination <path>  output directory (default: audit-report)
  --fix                 after scanning, suggest manual fixes for CONFIRMED conflicts and
                         offer to write them to <destination>/suggested-fixes.md (asks for
                         confirmation first; never edits any plugin/skill file itself)

Exit codes:
  0  scan succeeded, no CONFIRMED findings
  1  scan succeeded, at least one CONFIRMED finding (useful as a CI gate)
  2  fatal error
`)
}

var allModules = []string{"discover", "conflict-check", "token-cost", "memory-audit", "trust-report"}

func runInitConfig(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init-config", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if _, err := os.Stat(config.DefaultConfigFile); err == nil {
		fmt.Fprintf(stderr, "%s already exists, not overwriting\n", config.DefaultConfigFile)
		return 2
	}

	const template = `scan_paths:
  extra: []
exclude: []
output:
  format: [markdown, json]
  destination: ./audit-report
thresholds:
  token_overhead_warning: 10000
`
	if err := os.WriteFile(config.DefaultConfigFile, []byte(template), 0o644); err != nil {
		fmt.Fprintf(stderr, "error writing %s: %v\n", config.DefaultConfigFile, err)
		return 2
	}
	fmt.Fprintf(stdout, "wrote %s\n", config.DefaultConfigFile)
	return 0
}

func runScan(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	only := fs.String("only", "", "comma-separated modules to run")
	format := fs.String("format", "", "markdown | json | both")
	// --tui is accepted for FASE 8 command-reference compatibility; see the
	// comment above the render call below for why it's a no-op in v0.1.
	fs.Bool("tui", false, "force interactive terminal summary")
	quiet := fs.Bool("quiet", false, "no terminal output, only write files")
	verbose := fs.Bool("verbose", false, "log every skipped/errored path")
	configPath := fs.String("config", config.DefaultConfigFile, "path to .agent-stack-audit.yml")
	destination := fs.String("destination", "", "output directory")
	fixFlag := fs.Bool("fix", false, "suggest manual fixes for CONFIRMED conflicts (never edits plugin/skill files)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "error loading %s: %v\n", *configPath, err)
		return 2
	}

	dest := *destination
	if dest == "" {
		dest = cfg.Output.Destination
	}
	if dest == "" {
		dest = "audit-report"
	}

	outputFormat := *format
	if outputFormat == "" && len(cfg.Output.Format) > 0 {
		outputFormat = strings.Join(cfg.Output.Format, ",")
	}
	if outputFormat == "" {
		outputFormat = "both"
	}

	enabled := resolveModules(*only)

	var discoverEntries []discover.SkillEntry
	var scanErrs []error
	if enabled["discover"] || enabled["conflict-check"] || enabled["token-cost"] || enabled["trust-report"] {
		discoverEntries, scanErrs = discover.NewFSScanner().Scan(cfg.ResolveDiscoverPaths())
		discoverEntries = discover.FilterExcluded(discoverEntries, cfg.Exclude)
	}

	if *verbose {
		for _, e := range scanErrs {
			fmt.Fprintf(stderr, "warning: %v\n", e)
		}
	}

	var conflicts []conflict.Conflict
	if enabled["conflict-check"] {
		conflicts = conflict.NewPatternChecker().Check(discoverEntries)
	}

	var tokenCost []tokencost.Entry
	if enabled["token-cost"] {
		tokenCost = tokencost.NewCharRatioEstimator().Estimate(discoverEntries)
	}

	var memEntries []memoryaudit.Entry
	if enabled["memory-audit"] {
		memEntries = memoryaudit.NewFSAuditor().Audit(config.MemoryTargets())
	}

	var trustFindings []trustreport.Finding
	if enabled["trust-report"] {
		trustFindings = trustreport.NewChecker().Report(discoverEntries)
	}

	r := report.Build(report.BuildInput{
		Host:        runtime.GOOS + "-" + runtime.GOARCH,
		Discover:    discoverEntries,
		Conflicts:   conflicts,
		TokenCost:   tokenCost,
		MemoryAudit: memEntries,
		TrustReport: trustFindings,
	})

	if outputFormat == "json" || outputFormat == "both" {
		if err := report.NewJSONWriter().Write(r, dest); err != nil {
			fmt.Fprintf(stderr, "error writing report.json: %v\n", err)
			return 2
		}
	}
	if outputFormat == "markdown" || outputFormat == "both" {
		if err := report.NewMarkdownWriter().Write(r, dest); err != nil {
			fmt.Fprintf(stderr, "error writing report.md: %v\n", err)
			return 2
		}
	}

	// tui.Render always goes through lipgloss's default renderer, which
	// detects terminal capability on its own and downgrades to plain text
	// automatically when stdout isn't a color-capable TTY - this alone
	// satisfies FASE 8's "default: on kalau terminal mendukung, otomatis
	// fallback ke plain text kalau tidak". --tui is accepted for interface
	// compatibility with the FASE 8 command reference but doesn't change
	// behavior in v0.1: there is no separate non-TUI terminal summary to
	// fall back to.
	if !*quiet {
		fmt.Fprintln(stdout, tui.Render(r))
	}

	if *fixFlag {
		if err := runFix(conflicts, dest, stdin, stdout); err != nil {
			fmt.Fprintf(stderr, "error writing suggested-fixes.md: %v\n", err)
			return 2
		}
	}

	if hasConfirmed(r) {
		return 1
	}
	return 0
}

func resolveModules(only string) map[string]bool {
	enabled := make(map[string]bool, len(allModules))
	if only == "" {
		for _, m := range allModules {
			enabled[m] = true
		}
		return enabled
	}
	for _, m := range strings.Split(only, ",") {
		enabled[strings.TrimSpace(m)] = true
	}
	return enabled
}

// runFix prints a manual-fix suggestion for every CONFIRMED conflict, then
// asks for explicit confirmation before writing them to a file - it never
// writes anything without a "y" answer, and never touches any file outside
// dest (see internal/fix's own doc comment: it doesn't even know where a
// plugin's real config file lives).
func runFix(conflicts []conflict.Conflict, dest string, stdin io.Reader, stdout io.Writer) error {
	suggestions := fix.Suggest(conflicts)
	if len(suggestions) == 0 {
		fmt.Fprintln(stdout, "\n--fix: tidak ada konflik CONFIRMED, tidak ada saran untuk ditulis.")
		return nil
	}

	fmt.Fprintln(stdout, "\n--fix: saran untuk konflik CONFIRMED (manual, tidak ada file plugin lain yang diubah):")
	for _, s := range suggestions {
		fmt.Fprintf(stdout, "\n%s\n%s", s.ConflictID, s.Text)
	}

	fixPath := filepath.Join(dest, "suggested-fixes.md")
	fmt.Fprintf(stdout, "\nTulis saran ini ke %s? [y/N]: ", fixPath)
	answer := readLine(stdin)
	if !isYes(answer) {
		fmt.Fprintln(stdout, "Tidak ditulis.")
		return nil
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(fixPath, []byte(fix.RenderMarkdown(suggestions)), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Ditulis ke %s\n", fixPath)
	return nil
}

func readLine(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return ""
	}
	return scanner.Text()
}

func isYes(answer string) bool {
	a := strings.ToLower(strings.TrimSpace(answer))
	return a == "y" || a == "yes"
}

func hasConfirmed(r report.Report) bool {
	for _, c := range r.Conflicts {
		if c.Confidence == "CONFIRMED" {
			return true
		}
	}
	for _, t := range r.TrustReport {
		if t.Confidence == "CONFIRMED" {
			return true
		}
	}
	return false
}
