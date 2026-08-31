// Package trustreport checks basic trust indicators on discovered skills,
// plugins, and hooks. It never blocks anything - purely passive reporting,
// per FASE 7's explicit design principle: the decision stays with the user.
package trustreport

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/damta8827773/agent-stack-audit/internal/discover"
)

type Finding struct {
	ID         string
	Confidence string
	Finding    string
	Path       string
}

type Reporter interface {
	Report(entries []discover.SkillEntry) []Finding
}

type Checker struct{}

func NewChecker() *Checker { return &Checker{} }

var licenseNames = []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING"}
var securityNames = []string{"SECURITY.md"}
var networkCallPattern = regexp.MustCompile(`(?i)\b(curl|wget)\b|fetch\(|https?://`)
var scriptExtensions = []string{".sh", ".py", ".js", ".rb", ".ps1"}

// ipv4URLPattern and ipv6URLPattern match a raw IP literal used directly in
// a URL (http://192.168.1.1/... or http://[::1]:8080/...) rather than a
// domain name. A hardcoded IP bypasses DNS and domain-reputation checks
// entirely - a pattern more associated with C2/exfiltration endpoints than
// ordinary API calls - so it gets a more specific finding than the generic
// external-call check below, though still LIKELY: plenty of legitimate
// internal tooling also calls raw IPs (e.g. localhost, a LAN service).
var ipv4URLPattern = regexp.MustCompile(`https?://(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?::[0-9]+)?(?:[/\s"']|$)`)
var ipv6URLPattern = regexp.MustCompile(`https?://\[[0-9a-fA-F:]+\](?::[0-9]+)?`)

// genericMarketingPhrases mirrors Lampiran C's forbidden-word list for this
// project's own docs (words barred without a concrete number backing them
// up), plus a couple of well-known AI-generated-copy clichés. A skill
// description leaning on this vocabulary is a weak, INFORMATIONAL-only
// signal of low-effort templated copy - NOT a claim that the text was
// AI-written. There is no reliable way to detect that from surface
// features alone, and this tool doesn't pretend otherwise.
var genericMarketingPhrases = []string{
	"revolutionary", "game-changing", "game changer", "seamless", "seamlessly",
	"blazing fast", "next-generation", "next generation", "cutting-edge",
	"cutting edge", "state-of-the-art", "state of the art",
	"unleash the power", "unlock the power", "take your workflow to the next level",
	"supercharge your", "elevate your",
}

func (c *Checker) Report(entries []discover.SkillEntry) []Finding {
	var findings []Finding
	n := 1

	seenDirs := map[string]bool{}
	for _, e := range entries {
		switch e.Type {
		case "skill", "plugin":
			dir := filepath.Dir(e.Path)
			if seenDirs[dir] {
				continue
			}
			seenDirs[dir] = true
			findings = append(findings, checkDir(dir, e.Description, &n)...)
		case "hook":
			if f, ok := checkHookCommand(e, &n); ok {
				findings = append(findings, f)
			}
		}
	}
	return findings
}

func checkDir(dir, description string, n *int) []Finding {
	var findings []Finding

	if info, err := os.Stat(dir); err == nil && isWorldWritable(info) {
		findings = append(findings, newFinding(n, "CONFIRMED",
			"Direktori skill/plugin world-writable - pengguna lain di mesin ini bisa memodifikasinya", dir))
	}

	if !hasAnyFile(dir, licenseNames) {
		findings = append(findings, newFinding(n, "INFORMATIONAL", "Skill/plugin tanpa file LICENSE", dir))
	}
	if !hasAnyFile(dir, securityNames) {
		findings = append(findings, newFinding(n, "INFORMATIONAL", "Tidak ada SECURITY.md", dir))
	}

	if f, ok := checkGenericDescription(description, dir, n); ok {
		findings = append(findings, f)
	}

	findings = append(findings, checkExecutableScripts(dir, n)...)
	return findings
}

func checkHookCommand(e discover.SkillEntry, n *int) (Finding, bool) {
	if ipv4URLPattern.MatchString(e.Command) || ipv6URLPattern.MatchString(e.Command) {
		return newFinding(n, "LIKELY",
			fmt.Sprintf("Command hook (%s, event %s) memanggil alamat IP mentah, bukan nama domain - pola ini lebih umum dipakai untuk bypass DNS/reputation check ketimbang panggilan API biasa, butuh review manual", e.SourceSystem, e.Event),
			e.Path), true
	}
	if !networkCallPattern.MatchString(e.Command) {
		return Finding{}, false
	}
	return newFinding(n, "LIKELY",
		fmt.Sprintf("Command hook (%s, event %s) memanggil curl/wget/fetch ke domain eksternal - butuh review manual, bukan otomatis diblokir", e.SourceSystem, e.Event),
		e.Path), true
}

// checkGenericDescription flags a skill/plugin description leaning on
// generic marketing vocabulary (see genericMarketingPhrases). This is
// INFORMATIONAL only and explicitly not a claim that the content is
// AI-generated - see the doc comment on genericMarketingPhrases.
func checkGenericDescription(description, dir string, n *int) (Finding, bool) {
	if description == "" {
		return Finding{}, false
	}
	lower := strings.ToLower(description)
	for _, phrase := range genericMarketingPhrases {
		if strings.Contains(lower, phrase) {
			return newFinding(n, "INFORMATIONAL",
				fmt.Sprintf("Deskripsi memakai frasa generik/superlatif (%q) tanpa klaim konkret di baliknya - bukan bukti tulisan AI, cuma sinyal lemah copy template", phrase),
				dir), true
		}
	}
	return Finding{}, false
}

// checkExecutableScripts flags scripts under scripts/ or bin/ that look
// executable (script extension, or the Unix executable bit - meaningful
// on Unix only) and whose filename is never mentioned in the skill's own
// SKILL.md, per FASE 7's "tidak ada dokumentasi apa fungsinya" check.
func checkExecutableScripts(dir string, n *int) []Finding {
	var findings []Finding
	docText := readDocText(dir)

	for _, sub := range []string{"scripts", "bin"} {
		subdir := filepath.Join(dir, sub)
		items, err := os.ReadDir(subdir)
		if err != nil {
			continue
		}
		for _, item := range items {
			if item.IsDir() || !looksExecutable(item) {
				continue
			}
			if strings.Contains(docText, item.Name()) {
				continue
			}
			findings = append(findings, newFinding(n, "LIKELY",
				fmt.Sprintf("Skill memuat script executable (%s) tapi tidak ada dokumentasi apa fungsinya", item.Name()),
				filepath.Join(subdir, item.Name())))
		}
	}
	return findings
}

func looksExecutable(item os.DirEntry) bool {
	lower := strings.ToLower(item.Name())
	for _, ext := range scriptExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	info, err := item.Info()
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0o111 != 0
}

func readDocText(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return ""
	}
	return string(data)
}

func hasAnyFile(dir string, names []string) bool {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// isWorldWritable checks group/other write bits. Windows has no POSIX-style
// permission bit model - Go synthesizes a fixed mode there that would
// false-flag ordinary directories as world-writable, so the check is
// skipped (not reported) rather than risk a false CONFIRMED finding.
func isWorldWritable(info os.FileInfo) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	return info.Mode().Perm()&0o022 != 0
}

func newFinding(n *int, confidence, finding, path string) Finding {
	f := Finding{
		ID:         fmt.Sprintf("trust-%03d", *n),
		Confidence: confidence,
		Finding:    finding,
		Path:       path,
	}
	*n++
	return f
}
