// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
)

// ClaudeConfigDir returns the base Claude Code config directory, honoring
// the CLAUDE_CONFIG_DIR environment variable (default: ~/.claude). Found
// during FASE 0 research (watermarks-remover honors this override) - a
// hardcoded ~/.claude would miss any machine that customizes it.
func ClaudeConfigDir() string {
	if v := os.Getenv("CLAUDE_CONFIG_DIR"); v != "" {
		return v
	}
	return filepath.Join(homeDir(), ".claude")
}

// ClaudeMemDataDir returns claude-mem's data directory, honoring
// CLAUDE_MEM_DATA_DIR (default: ~/.claude-mem).
func ClaudeMemDataDir() string {
	if v := os.Getenv("CLAUDE_MEM_DATA_DIR"); v != "" {
		return v
	}
	return filepath.Join(homeDir(), ".claude-mem")
}

// AuditLogPath returns the path to the append-only, hash-chained scan
// history file (CLAUDE.md section 3 / see internal/auditlog), honoring
// AGENT_STACK_AUDIT_HOME (default: ~/.agent-stack-audit).
func AuditLogPath() string {
	if v := os.Getenv("AGENT_STACK_AUDIT_HOME"); v != "" {
		return filepath.Join(v, "audit-log.jsonl")
	}
	return filepath.Join(homeDir(), ".agent-stack-audit", "audit-log.jsonl")
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// MemoryTarget describes one local memory/state store known to
// agent-stack-audit's memory-audit module. Metadata only, never content.
type MemoryTarget struct {
	System string
	Path   string
}

// DiscoverPaths returns the v0.1 (Claude Code-only) default locations the
// discover module walks looking for SKILL.md / plugin.json /
// marketplace.json / settings.json. Each entry is a distinct, non-nested
// location (skills/ and plugins/ are scanned separately, settings.json is
// a direct file target) so discover never double-counts the same file
// through two overlapping roots.
func DiscoverPaths() []string {
	claude := ClaudeConfigDir()
	return []string{
		filepath.Join(claude, "skills"),
		filepath.Join(claude, "plugins"),
		filepath.Join(claude, "settings.json"),
		filepath.Join(".claude", "skills"),
		filepath.Join(".claude", "settings.json"),
	}
}

// MemoryTargets returns the v0.1 default local memory/state stores that
// memory-audit reports metadata for. gstack's ~/.gstack/ was added here
// (not to DiscoverPaths) after FASE 0 research: its config.yaml and
// egress.jsonl are state/ledger files, not SKILL.md/plugin.json artifacts,
// so they get the same metadata-only treatment as claude-mem/ECC rather
// than being fed through the skill/plugin/hook parser.
func MemoryTargets() []MemoryTarget {
	claudeMem := ClaudeMemDataDir()
	home := homeDir()
	return []MemoryTarget{
		{System: "claude-mem", Path: filepath.Join(claudeMem, "claude-mem.db")},
		{System: "claude-mem", Path: filepath.Join(claudeMem, "chroma")},
		{System: "ECC Memory Vault", Path: filepath.Join(home, ".ecc", "memory")},
		{System: "ECC Memory Vault", Path: filepath.Join(".ecc", "memory")},
		{System: "gstack", Path: filepath.Join(home, ".gstack", "config.yaml")},
		{System: "gstack", Path: filepath.Join(home, ".gstack", "security", "egress.jsonl")},
	}
}

// ResolveDiscoverPaths applies user-configured extra paths on top of the
// built-in defaults. Excludes are NOT applied here: the sample
// .agent-stack-audit.yml excludes specific sub-paths discovered during the
// walk (e.g. one private skill directory), not top-level scan roots, so
// exclude filtering happens against found entries (see discover.FilterExcluded).
func (c *Config) ResolveDiscoverPaths() []string {
	paths := DiscoverPaths()
	paths = append(paths, c.ScanPaths.Extra...)
	return paths
}
