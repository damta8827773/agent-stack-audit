// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFileReturnsZeroValue(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yml"))
	if err != nil {
		t.Fatalf("Load on missing file should not error, got: %v", err)
	}
	if len(cfg.ScanPaths.Extra) != 0 || len(cfg.Exclude) != 0 {
		t.Fatalf("expected zero-value Config, got %+v", cfg)
	}
}

func TestLoad_ParsesFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".agent-stack-audit.yml")
	content := `
scan_paths:
  extra:
    - ~/custom-agent-config
exclude:
  - ~/.claude/skills/my-private-skill
output:
  format: [markdown, json]
  destination: ./audit-report
thresholds:
  token_overhead_warning: 10000
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(cfg.ScanPaths.Extra) != 1 || cfg.ScanPaths.Extra[0] != "~/custom-agent-config" {
		t.Errorf("scan_paths.extra not parsed correctly: %+v", cfg.ScanPaths.Extra)
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "~/.claude/skills/my-private-skill" {
		t.Errorf("exclude not parsed correctly: %+v", cfg.Exclude)
	}
	if cfg.Thresholds.TokenOverheadWarning != 10000 {
		t.Errorf("thresholds.token_overhead_warning = %d, want 10000", cfg.Thresholds.TokenOverheadWarning)
	}
	if cfg.Output.Destination != "./audit-report" {
		t.Errorf("output.destination = %q, want ./audit-report", cfg.Output.Destination)
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".agent-stack-audit.yml")
	if err := os.WriteFile(path, []byte("scan_paths: [this is not: valid: yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestClaudeConfigDir_DefaultAndOverride(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "")
		got := ClaudeConfigDir()
		want := filepath.Join(homeDir(), ".claude")
		if got != want {
			t.Errorf("ClaudeConfigDir() = %q, want %q", got, want)
		}
	})

	t.Run("override", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "/custom/claude/dir")
		got := ClaudeConfigDir()
		if got != "/custom/claude/dir" {
			t.Errorf("ClaudeConfigDir() = %q, want override to be honored", got)
		}
	})
}

func TestClaudeMemDataDir_DefaultAndOverride(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("CLAUDE_MEM_DATA_DIR", "")
		got := ClaudeMemDataDir()
		want := filepath.Join(homeDir(), ".claude-mem")
		if got != want {
			t.Errorf("ClaudeMemDataDir() = %q, want %q", got, want)
		}
	})

	t.Run("override", func(t *testing.T) {
		t.Setenv("CLAUDE_MEM_DATA_DIR", "/custom/claude-mem")
		got := ClaudeMemDataDir()
		if got != "/custom/claude-mem" {
			t.Errorf("ClaudeMemDataDir() = %q, want override to be honored", got)
		}
	})
}

func TestDiscoverPaths_NoOverlappingRoots(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/home/user/.claude")
	paths := DiscoverPaths()
	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			t.Errorf("duplicate path in DiscoverPaths: %s", p)
		}
		seen[p] = true
	}
	// skills/ and plugins/ must not be nested inside each other, and
	// settings.json must be a direct file target, not re-derivable from
	// walking skills/ or plugins/.
	want := []string{
		filepath.Join("/home/user/.claude", "skills"),
		filepath.Join("/home/user/.claude", "plugins"),
		filepath.Join("/home/user/.claude", "settings.json"),
		filepath.Join(".claude", "skills"),
		filepath.Join(".claude", "settings.json"),
	}
	if len(paths) != len(want) {
		t.Fatalf("DiscoverPaths() = %v, want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("DiscoverPaths()[%d] = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestResolveDiscoverPaths_AppendsExtra(t *testing.T) {
	cfg := &Config{ScanPaths: ScanPaths{Extra: []string{"~/custom-agent-config"}}}
	paths := cfg.ResolveDiscoverPaths()
	found := false
	for _, p := range paths {
		if p == "~/custom-agent-config" {
			found = true
		}
	}
	if !found {
		t.Errorf("ResolveDiscoverPaths() did not include extra path: %v", paths)
	}
}

func TestMemoryTargets_IncludesGstack(t *testing.T) {
	targets := MemoryTargets()
	var systems []string
	for _, m := range targets {
		systems = append(systems, m.System)
	}
	hasGstack := false
	for _, s := range systems {
		if s == "gstack" {
			hasGstack = true
		}
	}
	if !hasGstack {
		t.Errorf("MemoryTargets() missing gstack entries, got systems: %v", systems)
	}
}
