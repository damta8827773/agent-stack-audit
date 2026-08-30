package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// buildTwoSystemConflictFixture reproduces the real, FASE 0-confirmed
// conflict (claude-mem vs superpowers, SessionStart, identical matcher)
// used across the discover/conflict golden fixtures — this is the
// integration-level version FASE 10 calls for: "Jalankan scan penuh
// terhadap fixture gabungan yang mensimulasikan 2+ skill system dengan
// hook konflik yang disengaja."
func buildTwoSystemConflictFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	writeFixtureFile(t, filepath.Join(root, "skills", "ecc-tdd-workflow", "SKILL.md"), `---
name: tdd-workflow
description: TDD helper
---
Always-on skill body.
`)
	writeFixtureFile(t, filepath.Join(root, "plugins", "marketplaces", "thedotmack", ".claude-plugin", "plugin.json"),
		`{"name": "claude-mem"}`)
	writeFixtureFile(t, filepath.Join(root, "plugins", "marketplaces", "thedotmack", "hooks", "hooks.json"), `{
		"SessionStart": [{"matcher": "startup|clear|compact", "hooks": [{"type": "command", "command": "claude-mem-session-start.js"}]}]
	}`)
	writeFixtureFile(t, filepath.Join(root, "plugins", "marketplaces", "obra", ".claude-plugin", "plugin.json"),
		`{"name": "superpowers"}`)
	writeFixtureFile(t, filepath.Join(root, "plugins", "marketplaces", "obra", "hooks", "hooks.json"), `{
		"SessionStart": [{"matcher": "startup|clear|compact", "hooks": [{"type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/hooks/run-hook.cmd session-start"}]}]
	}`)

	return root
}

// isolateMemoryTargets points every env-var-driven memory location at
// directories that don't exist, so the test's memory-audit results depend
// only on the fixture, never on whatever happens to be installed on the
// machine actually running the test.
func isolateMemoryTargets(t *testing.T) {
	t.Helper()
	empty := t.TempDir()
	t.Setenv("CLAUDE_MEM_DATA_DIR", filepath.Join(empty, "no-claude-mem"))
	t.Setenv("HOME", filepath.Join(empty, "no-home"))
	t.Setenv("USERPROFILE", filepath.Join(empty, "no-home"))
}

func TestRun_ScanEndToEnd_DetectsRealConfirmedConflict(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--destination", dest}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (a CONFIRMED conflict exists); stderr: %s", code, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(dest, "report.json"))
	if err != nil {
		t.Fatalf("report.json not written: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("report.json invalid: %v", err)
	}
	conflicts, _ := parsed["conflicts"].([]interface{})
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict in report.json, got %d: %v", len(conflicts), parsed["conflicts"])
	}
	first := conflicts[0].(map[string]interface{})
	if first["confidence"] != "CONFIRMED" {
		t.Errorf("confidence = %v, want CONFIRMED", first["confidence"])
	}

	if _, err := os.Stat(filepath.Join(dest, "report.md")); err != nil {
		t.Errorf("report.md not written: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Konflik ditemukan")) {
		t.Errorf("expected TUI summary on stdout, got: %s", stdout.String())
	}
}

func TestRun_ScanOnlyDiscoverSkipsOtherModules(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--only", "discover", "--destination", dest}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (conflict-check didn't run), stderr: %s", code, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(dest, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)
	if conflicts, _ := parsed["conflicts"].([]interface{}); len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts when --only discover, got %v", conflicts)
	}
	discoverEntries, _ := parsed["discover"].([]interface{})
	if len(discoverEntries) == 0 {
		t.Errorf("expected discover to still run and find entries")
	}
}

func TestRun_ScanFormatJSONOnly(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	var stdout, stderr bytes.Buffer
	Run([]string{"scan", "--format", "json", "--destination", dest}, &stdout, &stderr)

	if _, err := os.Stat(filepath.Join(dest, "report.json")); err != nil {
		t.Errorf("report.json should exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "report.md")); !os.IsNotExist(err) {
		t.Errorf("report.md should NOT exist when --format json, err: %v", err)
	}
}

func TestRun_ScanQuietSuppressesStdout(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	var stdout, stderr bytes.Buffer
	Run([]string{"scan", "--quiet", "--destination", dest}, &stdout, &stderr)

	if stdout.Len() != 0 {
		t.Errorf("expected no stdout output with --quiet, got: %s", stdout.String())
	}
}

func TestRun_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("agent-stack-audit v")) {
		t.Errorf("unexpected version output: %s", stdout.String())
	}
}

func TestRun_NoArgsPrintsUsageAndExits2(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage:")) {
		t.Errorf("expected usage text, got: %s", stdout.String())
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"bogus"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("unknown command")) {
		t.Errorf("expected unknown-command message on stderr, got: %s", stderr.String())
	}
}

func TestRun_InitConfigWritesFileAndRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	origWD, _ := os.Getwd()
	t.Chdir(dir)
	defer os.Chdir(origWD)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"init-config"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stderr: %s", code, stderr.String())
	}
	if _, err := os.Stat(".agent-stack-audit.yml"); err != nil {
		t.Fatalf(".agent-stack-audit.yml not written: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"init-config"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("expected exit 2 when config already exists, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("already exists")) {
		t.Errorf("expected 'already exists' warning, got: %s", stderr.String())
	}
}

func TestRun_MissingDirectoryIsNotFatal(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "does-not-exist"))
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--destination", dest}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 for an entirely empty/missing config dir, stderr: %s", code, stderr.String())
	}
}
