// Package trustreport checks basic trust indicators on discovered skills,
// plugins, and hooks. It never blocks anything — purely passive reporting,
// per FASE 7's explicit design principle: the decision stays with the user.
package trustreport

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/damtafaiz/agent-stack-audit/internal/discover"
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
			findings = append(findings, checkDir(dir, &n)...)
		case "hook":
			if f, ok := checkHookCommand(e, &n); ok {
				findings = append(findings, f)
			}
		}
	}
	return findings
}

func checkDir(dir string, n *int) []Finding {
	var findings []Finding

	if info, err := os.Stat(dir); err == nil && isWorldWritable(info) {
		findings = append(findings, newFinding(n, "CONFIRMED",
			"Direktori skill/plugin world-writable — pengguna lain di mesin ini bisa memodifikasinya", dir))
	}

	if !hasAnyFile(dir, licenseNames) {
		findings = append(findings, newFinding(n, "INFORMATIONAL", "Skill/plugin tanpa file LICENSE", dir))
	}
	if !hasAnyFile(dir, securityNames) {
		findings = append(findings, newFinding(n, "INFORMATIONAL", "Tidak ada SECURITY.md", dir))
	}

	findings = append(findings, checkExecutableScripts(dir, n)...)
	return findings
}

func checkHookCommand(e discover.SkillEntry, n *int) (Finding, bool) {
	if !networkCallPattern.MatchString(e.Command) {
		return Finding{}, false
	}
	return newFinding(n, "LIKELY",
		fmt.Sprintf("Command hook (%s, event %s) memanggil curl/wget/fetch ke domain eksternal — butuh review manual, bukan otomatis diblokir", e.SourceSystem, e.Event),
		e.Path), true
}

// checkExecutableScripts flags scripts under scripts/ or bin/ that look
// executable (script extension, or the Unix executable bit — meaningful
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
// permission bit model — Go synthesizes a fixed mode there that would
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
