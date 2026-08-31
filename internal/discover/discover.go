// Package discover finds skills, plugins, and hooks registered in known
// Claude Code config locations and reports what it found - it never
// modifies anything.
package discover

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillEntry is the base discover/conflict contract (Lampiran E), extended
// with Event/Matcher/Command so a Type=="hook" entry carries what
// conflict-check needs (FASE 4 input: "event, matcher, command, source_system").
type SkillEntry struct {
	SourceSystem   string
	Path           string
	Type           string // "skill" | "plugin" | "hook"
	HasFrontmatter bool
	AlwaysOn       bool
	Description    string // frontmatter/manifest description, when present - used by trust-report's generic-copy check
	Event          string // set only when Type == "hook"
	Matcher        string // set only when Type == "hook"
	Command        string // set only when Type == "hook"
}

type Scanner interface {
	Scan(paths []string) ([]SkillEntry, []error)
}

// maxDepth: directories are read at depth 0 (the root itself) through
// maxDepth inclusive - four ReadDir calls deep - matching FASE 3's "scan
// rekursif maksimal 3 level kedalaman" measured as levels below the root.
// This is deep enough to reach a marketplace-installed plugin's manifest
// (plugins/marketplaces/<vendor>/.claude-plugin/plugin.json is exactly 3
// directories below the plugins/ root).
const maxDepth = 3

// knownVendorPrefixes is a best-effort heuristic for inferring which
// project a raw (non-plugin-manifest) skill or hook belongs to, based on
// naming conventions observed during FASE 0 research (e.g.
// "ecc-tdd-workflow", "gstack-context-bill"). When no prefix matches, the
// literal directory name / "unknown" is used instead of guessing - see
// design principle 3 (jujur soal ketidakpastian).
var knownVendorPrefixes = []string{
	"ecc", "gstack", "superpowers", "claude-mem", "watermarks-remover", "graphify",
}

type FSScanner struct{}

func NewFSScanner() *FSScanner { return &FSScanner{} }

func (s *FSScanner) Scan(paths []string) ([]SkillEntry, []error) {
	var entries []SkillEntry
	var errs []error

	for _, root := range paths {
		expanded := expandHome(root)
		scanRoot(expanded, &entries, &errs)
	}

	return entries, errs
}

// FilterExcluded drops entries whose path is or is inside any excluded
// path. Exclude filtering happens against found entries rather than
// top-level roots because .agent-stack-audit.yml excludes name specific
// sub-paths discovered during the walk (one private skill dir), not the
// scan roots themselves.
func FilterExcluded(entries []SkillEntry, exclude []string) []SkillEntry {
	if len(exclude) == 0 {
		return entries
	}
	expandedExclude := make([]string, len(exclude))
	for i, e := range exclude {
		expandedExclude[i] = filepath.Clean(expandHome(e))
	}
	var result []SkillEntry
	for _, e := range entries {
		excluded := false
		cleaned := filepath.Clean(e.Path)
		for _, ex := range expandedExclude {
			if cleaned == ex || strings.HasPrefix(cleaned, ex+string(filepath.Separator)) {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, e)
		}
	}
	return result
}

func scanRoot(root string, entries *[]SkillEntry, errs *[]error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return // skip, not an error (FASE 3 skenario gagal)
		}
		if os.IsPermission(err) {
			*errs = append(*errs, fmt.Errorf("permission denied: %s", root))
			return
		}
		*errs = append(*errs, err)
		return
	}

	if !info.IsDir() {
		handleFile(filepath.Base(root), root, entries, errs)
		return
	}

	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		realRoot = root
	}
	ancestors := map[string]bool{realRoot: true}
	walkDir(root, 0, ancestors, entries, errs)
}

func walkDir(dir string, depth int, ancestors map[string]bool, entries *[]SkillEntry, errs *[]error) {
	if depth > maxDepth {
		return
	}

	items, err := os.ReadDir(dir)
	if err != nil {
		if os.IsPermission(err) {
			*errs = append(*errs, fmt.Errorf("permission denied: %s", dir))
		}
		return
	}

	for _, item := range items {
		full := filepath.Join(dir, item.Name())

		if item.Type()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(full)
			if err != nil {
				continue
			}
			target, err := os.Stat(resolved)
			if err != nil {
				continue
			}
			if !target.IsDir() {
				handleFile(item.Name(), resolved, entries, errs)
				continue
			}
			if ancestors[resolved] {
				*errs = append(*errs, fmt.Errorf("symlink loop detected: %s -> %s", full, resolved))
				continue
			}
			next := cloneAncestors(ancestors, resolved)
			walkDir(resolved, depth+1, next, entries, errs)
			continue
		}

		if item.IsDir() {
			walkDir(full, depth+1, ancestors, entries, errs)
			continue
		}

		handleFile(item.Name(), full, entries, errs)
	}
}

func cloneAncestors(m map[string]bool, add string) map[string]bool {
	next := make(map[string]bool, len(m)+1)
	for k, v := range m {
		next[k] = v
	}
	next[add] = true
	return next
}

func handleFile(name, path string, entries *[]SkillEntry, errs *[]error) {
	switch name {
	case "SKILL.md":
		parseSkillMD(path, entries, errs)
	case "plugin.json":
		parseManifestJSON(path, "plugin.json", entries, errs)
	case "marketplace.json":
		parseManifestJSON(path, "marketplace.json", entries, errs)
	case "settings.json":
		parseHooksFile(path, "settings.json", entries, errs)
	case "hooks.json":
		// FASE 0 research found that real hook registrations from
		// installed plugins (claude-mem, superpowers, watermarks-remover,
		// gstack) live in each plugin's own hooks/hooks.json, not merged
		// into the user's settings.json as FASE 3's text alone implies.
		// Scanning only settings.json would make conflict-check blind to
		// the real, verified conflicts found during research - so
		// hooks.json is treated the same way.
		parseHooksFile(path, "hooks.json", entries, errs)
	}
}

type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func parseSkillMD(path string, entries *[]SkillEntry, errs *[]error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsPermission(err) {
			*errs = append(*errs, fmt.Errorf("permission denied: %s", path))
		}
		return
	}

	data = stripBOM(data)
	fm, hasFrontmatter, fmErr := parseFrontmatter(data)
	if fmErr != nil {
		*errs = append(*errs, fmt.Errorf("malformed frontmatter in %s: %w", path, fmErr))
	}

	dirName := filepath.Base(filepath.Dir(path))
	*entries = append(*entries, SkillEntry{
		SourceSystem:   inferSourceSystem(dirName),
		Path:           path,
		Type:           "skill",
		HasFrontmatter: hasFrontmatter,
		AlwaysOn:       isAlwaysOn(path),
		Description:    fm.Description,
	})
}

// parseFrontmatter reports whether YAML frontmatter delimited by --- lines
// was found, and a non-nil error if it was found but malformed (per FASE 3:
// malformed files are a warning, not fatal - the caller decides how to
// surface that).
func parseFrontmatter(content []byte) (frontmatter, bool, error) {
	text := string(content)
	if !strings.HasPrefix(text, "---") {
		return frontmatter{}, false, nil
	}
	rest := text[3:]
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return frontmatter{}, false, nil
	}
	block := rest[:end]
	var fm frontmatter
	if err := yaml.Unmarshal([]byte(block), &fm); err != nil {
		return frontmatter{}, true, err
	}
	return fm, true, nil
}

type jsonManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func parseManifestJSON(path, kind string, entries *[]SkillEntry, errs *[]error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsPermission(err) {
			*errs = append(*errs, fmt.Errorf("permission denied: %s", path))
		}
		return
	}

	var m jsonManifest
	if err := json.Unmarshal(stripBOM(data), &m); err != nil {
		*errs = append(*errs, fmt.Errorf("malformed %s at %s: %w", kind, path, err))
		return
	}

	name := m.Name
	if name == "" {
		name = filepath.Base(filepath.Dir(path))
	}
	*entries = append(*entries, SkillEntry{
		SourceSystem: name,
		Path:         path,
		Type:         "plugin",
		AlwaysOn:     false,
		Description:  m.Description,
	})
}

type settingsFile struct {
	Hooks map[string][]hookMatcherGroup `json:"hooks"`
}

type hookMatcherGroup struct {
	Matcher string    `json:"matcher"`
	Hooks   []hookCmd `json:"hooks"`
}

type hookCmd struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

func parseHooksFile(path, kind string, entries *[]SkillEntry, errs *[]error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsPermission(err) {
			*errs = append(*errs, fmt.Errorf("permission denied: %s", path))
		}
		return
	}

	hooks, err := extractHooks(stripBOM(data))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("malformed %s at %s: %w", kind, path, err))
		return
	}

	for event, groups := range hooks {
		for _, g := range groups {
			for _, h := range g.Hooks {
				*entries = append(*entries, SkillEntry{
					SourceSystem: inferHookSource(path, h.Command),
					Path:         path,
					Type:         "hook",
					AlwaysOn:     true,
					Event:        event,
					Matcher:      g.Matcher,
					Command:      h.Command,
				})
			}
		}
	}
}

// inferHookSource picks the most reliable source_system it can for a hook
// registration. A plugin's own hooks.json sits next to (or one level below)
// its .claude-plugin/plugin.json, which is authoritative and doesn't depend
// on the command string containing a recognizable name - many real hook
// commands only reference "${CLAUDE_PLUGIN_ROOT}/hooks/run-hook.cmd" with no
// vendor name in it at all (confirmed for superpowers during FASE 0
// research). Falls back to the command-string heuristic, then to the
// directory name, in that order of confidence.
func inferHookSource(path, command string) string {
	dir := filepath.Dir(path)
	candidates := []string{
		filepath.Join(dir, ".claude-plugin", "plugin.json"),
		filepath.Join(filepath.Dir(dir), ".claude-plugin", "plugin.json"),
	}
	for _, c := range candidates {
		if data, err := os.ReadFile(c); err == nil {
			var m jsonManifest
			if json.Unmarshal(stripBOM(data), &m) == nil && m.Name != "" {
				return m.Name
			}
		}
	}
	if s := inferSourceFromCommand(command); s != "unknown" {
		return s
	}
	return filepath.Base(filepath.Dir(dir))
}

// extractHooks handles two real-world shapes found during FASE 0 research:
// settings.json always wraps hook registrations under a "hooks" key, but a
// plugin's own hooks/hooks.json is often the event map directly, with no
// wrapper.
func extractHooks(data []byte) (map[string][]hookMatcherGroup, error) {
	var wrapped settingsFile
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, err
	}
	if len(wrapped.Hooks) > 0 {
		return wrapped.Hooks, nil
	}
	var bare map[string][]hookMatcherGroup
	if err := json.Unmarshal(data, &bare); err == nil && len(bare) > 0 {
		return bare, nil
	}
	return wrapped.Hooks, nil
}

func inferSourceSystem(dirName string) string {
	lower := strings.ToLower(dirName)
	for _, prefix := range knownVendorPrefixes {
		if lower == prefix || strings.HasPrefix(lower, prefix+"-") {
			return prefix
		}
	}
	return dirName
}

func inferSourceFromCommand(cmd string) string {
	lower := strings.ToLower(cmd)
	for _, prefix := range knownVendorPrefixes {
		if strings.Contains(lower, prefix) {
			return prefix
		}
	}
	return "unknown"
}

// isAlwaysOn implements FASE 3's "dilihat dari lokasi file - root skills
// dir vs plugin-managed": anything under a plugins/ tree is installed and
// invoked on-demand; anything directly under skills/ loads every session.
func isAlwaysOn(path string) bool {
	normalized := filepath.ToSlash(path)
	if strings.Contains(normalized, "/plugins/") {
		return false
	}
	return strings.Contains(normalized, "/skills/")
}

// stripBOM removes a leading UTF-8 byte-order mark. Editors and tools on
// Windows commonly write one (e.g. PowerShell's `Out-File`/`Set-Content
// -Encoding utf8`) - encoding/json and a plain "---" prefix check both
// treat it as invalid content, which would silently misreport a valid
// config file as malformed on Windows.
func stripBOM(data []byte) []byte {
	return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}
