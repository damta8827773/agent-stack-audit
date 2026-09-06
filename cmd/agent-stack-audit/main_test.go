// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
// used across the discover/conflict golden fixtures - this is the
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
	code := Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &stdout, &stderr)

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

func TestRun_FixWritesFileOnYesConfirmation(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--destination", dest, "--fix"}, strings.NewReader("y\n"), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1, stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "conflict-001") {
		t.Errorf("expected the suggestion to be printed to stdout, got: %s", stdout.String())
	}

	fixPath := filepath.Join(dest, "suggested-fixes.md")
	data, err := os.ReadFile(fixPath)
	if err != nil {
		t.Fatalf("suggested-fixes.md not written despite 'y' confirmation: %v", err)
	}
	if !strings.Contains(string(data), "conflict-001") {
		t.Errorf("suggested-fixes.md missing the conflict ID, got: %s", data)
	}
	if !strings.Contains(string(data), "bukan perubahan otomatis") {
		t.Errorf("suggested-fixes.md missing the manual-only disclaimer, got: %s", data)
	}
}

func TestRun_FixDoesNotWriteOnNoConfirmation(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--destination", dest, "--fix"}, strings.NewReader("n\n"), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1, stderr: %s", code, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(dest, "suggested-fixes.md")); !os.IsNotExist(err) {
		t.Errorf("suggested-fixes.md should not exist after declining confirmation, err: %v", err)
	}
	if !strings.Contains(stdout.String(), "Tidak ditulis") {
		t.Errorf("expected a 'not written' acknowledgement on stdout, got: %s", stdout.String())
	}
}

func TestRun_FixWithoutFlagNeverPrompts(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	// No stdin available at all (empty reader) - if --fix were somehow
	// triggered without the flag, readLine would just return "", but the
	// real assertion here is that suggested-fixes.md never appears.
	var stdout, stderr bytes.Buffer
	Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &stdout, &stderr)

	if _, err := os.Stat(filepath.Join(dest, "suggested-fixes.md")); !os.IsNotExist(err) {
		t.Errorf("suggested-fixes.md should never be written without --fix, err: %v", err)
	}
}

func TestRun_FixWithNoConfirmedConflicts(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root) // empty: discover finds nothing
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--destination", dest, "--fix"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "tidak ada konflik CONFIRMED") {
		t.Errorf("expected the no-CONFIRMED-conflicts message, got: %s", stdout.String())
	}
}

func TestRun_VulnCheckPromptsAndDoesNothingWithoutConsent(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	// "n" declines consent - if this test ever reaches a real network call
	// it's a bug: declining must short-circuit before any HTTP request.
	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--destination", dest, "--vuln-check"}, strings.NewReader("n\n"), &stdout, &stderr)
	if code != 1 { // the fixture's own CONFIRMED conflict still applies
		t.Fatalf("exit code = %d, want 1, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "osv.dev") {
		t.Errorf("expected the consent prompt mentioning osv.dev, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "dibatalkan") {
		t.Errorf("expected a cancellation acknowledgement, got: %s", stdout.String())
	}

	data, err := os.ReadFile(filepath.Join(dest, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)
	if vulns, ok := parsed["vuln_audit"]; ok && len(vulns.([]interface{})) != 0 {
		t.Errorf("expected no vuln_audit entries when consent was declined, got: %v", vulns)
	}
}

func TestRun_VulnCheckWithoutFlagNeverPrompts(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	var stdout, stderr bytes.Buffer
	Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &stdout, &stderr)
	if strings.Contains(stdout.String(), "osv.dev") {
		t.Errorf("vuln-check prompt should never appear without --vuln-check, got: %s", stdout.String())
	}
}

func TestRun_VulnCheckNoManifestsFound(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)

	dest := filepath.Join(t.TempDir(), "audit-report")
	origWD, _ := os.Getwd()
	t.Chdir(t.TempDir())
	defer os.Chdir(origWD)

	// "y" consents, but the fixture has no go.mod/package.json/requirements.txt
	// anywhere, so this must stop before ever making an HTTP request.
	var stdout, stderr bytes.Buffer
	Run([]string{"scan", "--destination", dest, "--vuln-check"}, strings.NewReader("y\n"), &stdout, &stderr)
	if !strings.Contains(stdout.String(), "tidak ada manifest dependency") {
		t.Errorf("expected the no-manifests message, got: %s", stdout.String())
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
	code := Run([]string{"scan", "--only", "discover", "--destination", dest}, strings.NewReader(""), &stdout, &stderr)
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
	Run([]string{"scan", "--format", "json", "--destination", dest}, strings.NewReader(""), &stdout, &stderr)

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
	Run([]string{"scan", "--quiet", "--destination", dest}, strings.NewReader(""), &stdout, &stderr)

	if stdout.Len() != 0 {
		t.Errorf("expected no stdout output with --quiet, got: %s", stdout.String())
	}
}

func TestRun_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("agent-stack-audit v")) {
		t.Errorf("unexpected version output: %s", stdout.String())
	}
}

func TestRun_NoArgsPrintsUsageAndExits2(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(nil, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage:")) {
		t.Errorf("expected usage text, got: %s", stdout.String())
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"bogus"}, strings.NewReader(""), &stdout, &stderr)
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
	code := Run([]string{"init-config"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stderr: %s", code, stderr.String())
	}
	if _, err := os.Stat(".agent-stack-audit.yml"); err != nil {
		t.Fatalf(".agent-stack-audit.yml not written: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"init-config"}, strings.NewReader(""), &stdout, &stderr)
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
	code := Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 for an entirely empty/missing config dir, stderr: %s", code, stderr.String())
	}
}

func TestRun_ScanAppendsToAuditLog(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)
	logHome := t.TempDir()
	t.Setenv("AGENT_STACK_AUDIT_HOME", logHome)

	dest := t.TempDir()
	var stdout, stderr bytes.Buffer
	Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &stdout, &stderr)
	Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &stdout, &stderr)

	logPath := filepath.Join(logHome, "audit-log.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("expected audit log to exist at %s: %v", logPath, err)
	}
	lines := strings.Count(strings.TrimRight(string(data), "\n"), "\n") + 1
	if lines != 2 {
		t.Errorf("expected 2 audit log entries after 2 scans, got %d", lines)
	}
	if !strings.Contains(string(data), `"conflicts_found":1`) {
		t.Errorf("expected the fixture's real conflict to be reflected in the audit log, got: %s", data)
	}
}

func TestRun_VerifyLogOnEmptyLog(t *testing.T) {
	t.Setenv("AGENT_STACK_AUDIT_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	code := Run([]string{"verify-log"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0 when no scan has run yet", code)
	}
	if !strings.Contains(stdout.String(), "belum ada entri") {
		t.Errorf("expected a 'no entries yet' message, got: %s", stdout.String())
	}
}

func TestRun_VerifyLogOnIntactChain(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)
	t.Setenv("AGENT_STACK_AUDIT_HOME", t.TempDir())

	dest := t.TempDir()
	var buf bytes.Buffer
	Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &buf, &buf)
	Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &buf, &buf)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"verify-log"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0 for an intact chain, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "OK") {
		t.Errorf("expected 'OK' in output, got: %s", stdout.String())
	}
}

func TestRun_VerifyLogDetectsTampering(t *testing.T) {
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	isolateMemoryTargets(t)
	logHome := t.TempDir()
	t.Setenv("AGENT_STACK_AUDIT_HOME", logHome)

	dest := t.TempDir()
	var buf bytes.Buffer
	Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &buf, &buf)
	Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &buf, &buf)

	logPath := filepath.Join(logHome, "audit-log.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(data), `"conflicts_found":1`, `"conflicts_found":0`, 1)
	if tampered == string(data) {
		t.Fatal("test setup error: nothing was replaced")
	}
	if err := os.WriteFile(logPath, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"verify-log"}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1 for a tampered chain", code)
	}
	if !strings.Contains(stdout.String(), "RUSAK") {
		t.Errorf("expected 'RUSAK' in output, got: %s", stdout.String())
	}
}

func TestRun_DiffWithFewerThanTwoEntries(t *testing.T) {
	t.Setenv("AGENT_STACK_AUDIT_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	code := Run([]string{"diff"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "belum cukup riwayat") {
		t.Errorf("expected a 'not enough history' message, got: %s", stdout.String())
	}
}

func TestRun_DiffShowsDelta(t *testing.T) {
	isolateMemoryTargets(t)
	t.Setenv("AGENT_STACK_AUDIT_HOME", t.TempDir())
	dest := t.TempDir()
	var buf bytes.Buffer

	// First scan: an empty config, 0 conflicts.
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &buf, &buf)

	// Second scan: the real conflict fixture, 1 conflict.
	fixture := buildTwoSystemConflictFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", fixture)
	Run([]string{"scan", "--destination", dest}, strings.NewReader(""), &buf, &buf)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"diff"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Konflik ditemukan: 0 -> 1 (+1)") {
		t.Errorf("expected the conflict delta to be reported, got: %s", stdout.String())
	}
}
