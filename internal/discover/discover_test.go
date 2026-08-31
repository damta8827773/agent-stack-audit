package discover

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScan_MissingDirIsNotAnError(t *testing.T) {
	entries, errs := NewFSScanner().Scan([]string{filepath.Join(t.TempDir(), "nope")})
	if len(errs) != 0 {
		t.Errorf("expected no errors for a missing directory, got: %v", errs)
	}
	if len(entries) != 0 {
		t.Errorf("expected no entries, got: %v", entries)
	}
}

func TestScan_SkillMDWithFrontmatter(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	writeFile(t, filepath.Join(skillsDir, "ecc-tdd-workflow", "SKILL.md"), `---
name: tdd-workflow
description: TDD helper
---

Body text here.
`)

	entries, errs := NewFSScanner().Scan([]string{skillsDir})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d: %+v", len(entries), entries)
	}
	e := entries[0]
	if e.Type != "skill" {
		t.Errorf("Type = %q, want skill", e.Type)
	}
	if !e.HasFrontmatter {
		t.Errorf("HasFrontmatter = false, want true")
	}
	if e.SourceSystem != "ecc" {
		t.Errorf("SourceSystem = %q, want ecc (inferred from ecc-tdd-workflow dir name)", e.SourceSystem)
	}
	if !e.AlwaysOn {
		t.Errorf("AlwaysOn = false, want true for a raw skills/ dir entry")
	}
}

func TestScan_SkillMDWithoutFrontmatter(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	writeFile(t, filepath.Join(skillsDir, "plain-skill", "SKILL.md"), "Just a body, no frontmatter.\n")

	entries, _ := NewFSScanner().Scan([]string{skillsDir})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].HasFrontmatter {
		t.Errorf("HasFrontmatter = true, want false")
	}
}

func TestScan_MalformedFrontmatterIsWarningNotFatal(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	writeFile(t, filepath.Join(skillsDir, "broken", "SKILL.md"), "---\nname: [unclosed\n---\nbody\n")

	entries, errs := NewFSScanner().Scan([]string{skillsDir})
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 warning error, got %d: %v", len(errs), errs)
	}
	if len(entries) != 1 {
		t.Fatalf("scan should still produce an entry despite malformed frontmatter, got %d", len(entries))
	}
}

func TestScan_PluginJSONNotAlwaysOn(t *testing.T) {
	root := t.TempDir()
	pluginsDir := filepath.Join(root, "plugins")
	writeFile(t, filepath.Join(pluginsDir, "marketplaces", "thedotmack", ".claude-plugin", "plugin.json"),
		`{"name": "claude-mem", "version": "1.0.0"}`)

	entries, errs := NewFSScanner().Scan([]string{pluginsDir})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (plugin.json 3 dirs below plugins/ root must be reachable), got %d", len(entries))
	}
	e := entries[0]
	if e.Type != "plugin" {
		t.Errorf("Type = %q, want plugin", e.Type)
	}
	if e.SourceSystem != "claude-mem" {
		t.Errorf("SourceSystem = %q, want claude-mem", e.SourceSystem)
	}
	if e.AlwaysOn {
		t.Errorf("AlwaysOn = true, want false for a plugin-managed entry")
	}
}

func TestScan_MalformedJSONIsWarningNotFatal(t *testing.T) {
	root := t.TempDir()
	pluginsDir := filepath.Join(root, "plugins")
	writeFile(t, filepath.Join(pluginsDir, "broken", "plugin.json"), `{not valid json`)

	entries, errs := NewFSScanner().Scan([]string{pluginsDir})
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 warning error, got %d: %v", len(errs), errs)
	}
	if len(entries) != 0 {
		t.Fatalf("malformed manifest should not produce an entry, got %d", len(entries))
	}
}

// TestScan_RealConfirmedConflict reproduces the actual verified finding from
// FASE 0 research: claude-mem and superpowers both register SessionStart
// with the identical matcher "startup|clear|compact" via their own
// hooks/hooks.json files (not settings.json).
func TestScan_RealConfirmedConflict(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "plugins", "marketplaces", "thedotmack", ".claude-plugin", "plugin.json"), `{"name": "claude-mem"}`)
	writeFile(t, filepath.Join(root, "plugins", "marketplaces", "thedotmack", "hooks", "hooks.json"), `{
		"SessionStart": [
			{"matcher": "startup|clear|compact", "hooks": [{"type": "command", "command": "claude-mem-session-start.js"}]}
		]
	}`)
	writeFile(t, filepath.Join(root, "plugins", "marketplaces", "obra", ".claude-plugin", "plugin.json"), `{"name": "superpowers"}`)
	writeFile(t, filepath.Join(root, "plugins", "marketplaces", "obra", "hooks", "hooks.json"), `{
		"SessionStart": [
			{"matcher": "startup|clear|compact", "hooks": [{"type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/hooks/run-hook.cmd session-start"}]}
		]
	}`)

	entries, errs := NewFSScanner().Scan([]string{filepath.Join(root, "plugins")})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	var hooks []SkillEntry
	for _, e := range entries {
		if e.Type == "hook" {
			hooks = append(hooks, e)
		}
	}
	if len(hooks) != 2 {
		t.Fatalf("expected 2 hook entries, got %d: %+v", len(hooks), hooks)
	}
	for _, h := range hooks {
		if h.Event != "SessionStart" || h.Matcher != "startup|clear|compact" {
			t.Errorf("unexpected hook entry: %+v", h)
		}
	}

	sources := []string{hooks[0].SourceSystem, hooks[1].SourceSystem}
	sort.Strings(sources)
	if sources[0] != "claude-mem" || sources[1] != "superpowers" {
		t.Errorf("SourceSystem inference = %v, want [claude-mem superpowers]", sources)
	}
}

func TestScan_SettingsJSONHooksWrapper(t *testing.T) {
	root := t.TempDir()
	settingsPath := filepath.Join(root, "settings.json")
	writeFile(t, settingsPath, `{
		"hooks": {
			"PostToolUse": [
				{"matcher": "Write|Edit|MultiEdit|NotebookEdit", "hooks": [{"type": "command", "command": "watermarks-remover/run_hook.js"}]}
			]
		}
	}`)

	entries, errs := NewFSScanner().Scan([]string{settingsPath})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 hook entry, got %d", len(entries))
	}
	if entries[0].Event != "PostToolUse" || entries[0].Matcher != "Write|Edit|MultiEdit|NotebookEdit" {
		t.Errorf("unexpected entry: %+v", entries[0])
	}
	if entries[0].SourceSystem != "watermarks-remover" {
		t.Errorf("SourceSystem = %q, want watermarks-remover", entries[0].SourceSystem)
	}
}

func TestScan_DirectSettingsFileRoot(t *testing.T) {
	root := t.TempDir()
	settingsPath := filepath.Join(root, "settings.json")
	writeFile(t, settingsPath, `{"hooks": {}}`)

	// settings.json itself as a root (a file, not a dir) must be handled,
	// matching config.DiscoverPaths() which lists settings.json directly.
	entries, errs := NewFSScanner().Scan([]string{settingsPath})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for empty hooks map, got %d", len(entries))
	}
}

func TestScan_MaxDepthReachesMarketplaceNestedManifest(t *testing.T) {
	// plugins/(0) -> marketplaces(1) -> vendor(2) -> .claude-plugin(3) -> plugin.json
	// is exactly 3 directory levels below the plugins/ root; depth must
	// reach it or every real marketplace-installed plugin would be missed.
	root := t.TempDir()
	pluginsDir := filepath.Join(root, "plugins")
	writeFile(t, filepath.Join(pluginsDir, "marketplaces", "vendor", ".claude-plugin", "plugin.json"), `{"name": "deep-plugin"}`)

	entries, _ := NewFSScanner().Scan([]string{pluginsDir})
	if len(entries) != 1 {
		t.Fatalf("expected to find the deeply nested plugin.json, got %d entries", len(entries))
	}
}

func TestScan_BeyondMaxDepthIsNotFound(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	// 5 directories deep - beyond the max depth budget.
	writeFile(t, filepath.Join(skillsDir, "a", "b", "c", "d", "e", "SKILL.md"), "---\nname: too-deep\n---\n")

	entries, _ := NewFSScanner().Scan([]string{skillsDir})
	if len(entries) != 0 {
		t.Fatalf("expected the too-deep SKILL.md to be out of reach, got %d entries", len(entries))
	}
}

func TestScan_SymlinkLoopDetected(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	loopLink := filepath.Join(skillsDir, "loop")
	if err := os.Symlink(skillsDir, loopLink); err != nil {
		t.Skipf("symlinks not supported in this environment: %v", err)
	}

	_, errs := NewFSScanner().Scan([]string{skillsDir})
	found := false
	for _, e := range errs {
		if e != nil && strings.Contains(e.Error(), "symlink loop") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a symlink loop warning, got errors: %v", errs)
	}
}

// TestScan_UTF8BOMDoesNotBreakParsing reproduces a real bug found while
// generating a demo fixture on Windows: PowerShell's `Set-Content -Encoding
// utf8` writes a UTF-8 BOM, and encoding/json doesn't skip it, so a
// perfectly valid plugin.json/hooks.json/SKILL.md would otherwise be
// silently misreported as malformed on Windows.
func TestScan_UTF8BOMDoesNotBreakParsing(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	root := t.TempDir()

	skillsDir := filepath.Join(root, "skills")
	skillContent := append(bom, []byte("---\nname: bom-skill\n---\nbody\n")...)
	writeFile(t, filepath.Join(skillsDir, "bom-skill", "SKILL.md"), string(skillContent))

	pluginsDir := filepath.Join(root, "plugins")
	pluginContent := append(bom, []byte(`{"name": "bom-plugin"}`)...)
	writeFile(t, filepath.Join(pluginsDir, "vendor", "plugin.json"), string(pluginContent))

	settingsPath := filepath.Join(root, "settings.json")
	settingsContent := append(bom, []byte(`{"hooks": {"Stop": [{"matcher": "*", "hooks": [{"type": "command", "command": "x"}]}]}}`)...)
	writeFile(t, settingsPath, string(settingsContent))

	entries, errs := NewFSScanner().Scan([]string{skillsDir, pluginsDir, settingsPath})
	if len(errs) != 0 {
		t.Fatalf("BOM-prefixed but otherwise valid files must not produce warnings, got: %v", errs)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries (skill + plugin + hook), got %d: %+v", len(entries), entries)
	}

	var sawSkill, sawPlugin, sawHook bool
	for _, e := range entries {
		switch e.Type {
		case "skill":
			sawSkill = true
			if !e.HasFrontmatter {
				t.Errorf("BOM-prefixed SKILL.md should still parse frontmatter, got HasFrontmatter=false")
			}
		case "plugin":
			sawPlugin = true
			if e.SourceSystem != "bom-plugin" {
				t.Errorf("SourceSystem = %q, want bom-plugin", e.SourceSystem)
			}
		case "hook":
			sawHook = true
		}
	}
	if !sawSkill || !sawPlugin || !sawHook {
		t.Errorf("missing an expected entry type, got: %+v", entries)
	}
}

// TestScan_UTF8BOMDoesNotBreakHookSourceInference reproduces a second
// instance of the same BOM bug: inferHookSource reads a sibling
// .claude-plugin/plugin.json through its own os.ReadFile+json.Unmarshal
// call, separate from parseManifestJSON's - fixing stripBOM in one place
// and not the other left this path silently falling back to the vendor
// directory name instead of the real plugin name.
func TestScan_UTF8BOMDoesNotBreakHookSourceInference(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	root := t.TempDir()

	pluginJSON := append(bom, []byte(`{"name": "superpowers"}`)...)
	writeFile(t, filepath.Join(root, "plugins", "marketplaces", "obra", ".claude-plugin", "plugin.json"), string(pluginJSON))
	hooksJSON := append(bom, []byte(`{"SessionStart": [{"matcher": "startup|clear|compact", "hooks": [{"type": "command", "command": "run-hook.cmd session-start"}]}]}`)...)
	writeFile(t, filepath.Join(root, "plugins", "marketplaces", "obra", "hooks", "hooks.json"), string(hooksJSON))

	entries, errs := NewFSScanner().Scan([]string{filepath.Join(root, "plugins")})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	var hook *SkillEntry
	for i := range entries {
		if entries[i].Type == "hook" {
			hook = &entries[i]
		}
	}
	if hook == nil {
		t.Fatalf("expected a hook entry, got: %+v", entries)
	}
	if hook.SourceSystem != "superpowers" {
		t.Errorf("SourceSystem = %q, want superpowers (from the sibling plugin.json's own name, not the vendor dir fallback)", hook.SourceSystem)
	}
}

func TestScan_MarketplaceJSON(t *testing.T) {
	root := t.TempDir()
	pluginsDir := filepath.Join(root, "plugins")
	writeFile(t, filepath.Join(pluginsDir, "marketplaces", "mattpocock-skills", "marketplace.json"),
		`{"name": "mattpocock-skills"}`)

	entries, errs := NewFSScanner().Scan([]string{pluginsDir})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(entries) != 1 || entries[0].SourceSystem != "mattpocock-skills" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestScan_ManifestNameFallsBackToDirName(t *testing.T) {
	root := t.TempDir()
	pluginsDir := filepath.Join(root, "plugins")
	writeFile(t, filepath.Join(pluginsDir, "some-vendor", "plugin.json"), `{}`)

	entries, _ := NewFSScanner().Scan([]string{pluginsDir})
	if len(entries) != 1 || entries[0].SourceSystem != "some-vendor" {
		t.Fatalf("expected fallback to dir name, got: %+v", entries)
	}
}

func TestScan_RootIsPlainFileNotSkillOrHook(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "README.md")
	writeFile(t, path, "not a recognized filename")

	entries, errs := NewFSScanner().Scan([]string{path})
	if len(errs) != 0 || len(entries) != 0 {
		t.Fatalf("unrecognized root file should be ignored, got entries=%v errs=%v", entries, errs)
	}
}

func TestCloneAncestors(t *testing.T) {
	original := map[string]bool{"a": true}
	clone := cloneAncestors(original, "b")

	if !clone["a"] || !clone["b"] {
		t.Fatalf("clone missing expected keys: %+v", clone)
	}
	if len(original) != 1 {
		t.Fatalf("cloneAncestors must not mutate the original map, got: %+v", original)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir available in this environment")
	}
	got := expandHome(filepath.ToSlash("~/foo/bar"))
	want := filepath.Join(home, "foo", "bar")
	if got != want {
		t.Errorf("expandHome(~/foo/bar) = %q, want %q", got, want)
	}
	if expandHome("/already/absolute") != "/already/absolute" {
		t.Errorf("expandHome should leave non-~ paths unchanged")
	}
}

func TestFilterExcluded(t *testing.T) {
	entries := []SkillEntry{
		{Path: filepath.Join("home", "user", ".claude", "skills", "my-private-skill", "SKILL.md")},
		{Path: filepath.Join("home", "user", ".claude", "skills", "public-skill", "SKILL.md")},
	}
	exclude := []string{filepath.Join("home", "user", ".claude", "skills", "my-private-skill")}

	filtered := FilterExcluded(entries, exclude)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 entry after exclude, got %d: %+v", len(filtered), filtered)
	}
	if filtered[0].Path != entries[1].Path {
		t.Errorf("wrong entry survived filtering: %+v", filtered[0])
	}
}

func TestFilterExcluded_NoExcludesReturnsAllUnchanged(t *testing.T) {
	entries := []SkillEntry{{Path: "a"}, {Path: "b"}}
	filtered := FilterExcluded(entries, nil)
	if len(filtered) != 2 {
		t.Fatalf("expected all entries preserved, got %d", len(filtered))
	}
}
