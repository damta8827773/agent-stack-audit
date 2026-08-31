package trustreport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/damta8827773/agent-stack-audit/internal/discover"
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

func TestReport_MissingLicenseAndSecurity(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "some-skill", "SKILL.md")
	writeFile(t, skillPath, "---\nname: some-skill\n---\nbody")

	entries := []discover.SkillEntry{{Type: "skill", Path: skillPath, SourceSystem: "some-skill"}}
	findings := NewChecker().Report(entries)

	var reasons []string
	for _, f := range findings {
		reasons = append(reasons, f.Finding)
	}
	if !containsSubstr(reasons, "LICENSE") {
		t.Errorf("expected a missing-LICENSE finding, got: %v", reasons)
	}
	if !containsSubstr(reasons, "SECURITY.md") {
		t.Errorf("expected a missing-SECURITY.md finding, got: %v", reasons)
	}
	for _, f := range findings {
		if f.Confidence != "INFORMATIONAL" {
			t.Errorf("expected INFORMATIONAL confidence, got %q for %q", f.Confidence, f.Finding)
		}
	}
}

func TestReport_LicenseAndSecurityPresentSuppressesFindings(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "good-skill")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: good-skill\n---\n")
	writeFile(t, filepath.Join(skillDir, "LICENSE"), "MIT")
	writeFile(t, filepath.Join(skillDir, "SECURITY.md"), "report issues here")

	entries := []discover.SkillEntry{{Type: "skill", Path: filepath.Join(skillDir, "SKILL.md")}}
	findings := NewChecker().Report(entries)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings when LICENSE and SECURITY.md exist, got: %+v", findings)
	}
}

func TestReport_DeduplicatesSameDirAcrossMultipleEntries(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skill")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "body")

	// Two entries pointing at files within the same directory (a
	// realistic case: SKILL.md discovered plus a co-located manifest)
	// must only be checked once.
	entries := []discover.SkillEntry{
		{Type: "skill", Path: filepath.Join(skillDir, "SKILL.md")},
		{Type: "skill", Path: filepath.Join(skillDir, "SKILL.md")},
	}
	findings := NewChecker().Report(entries)
	licenseCount := 0
	for _, f := range findings {
		if f.Finding == "Skill/plugin tanpa file LICENSE" {
			licenseCount++
		}
	}
	if licenseCount != 1 {
		t.Errorf("expected exactly 1 missing-LICENSE finding (deduped), got %d", licenseCount)
	}
}

func TestReport_HookCallingCurlIsLikely(t *testing.T) {
	entries := []discover.SkillEntry{
		{
			Type:         "hook",
			SourceSystem: "some-tool",
			Event:        "PostToolUse",
			Path:         "/fake/hooks.json",
			Command:      "curl -X POST https://example.com/telemetry",
		},
	}
	findings := NewChecker().Report(entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].Confidence != "LIKELY" {
		t.Errorf("Confidence = %q, want LIKELY", findings[0].Confidence)
	}
}

func TestReport_HookCallingRawIPv4IsFlaggedDistinctly(t *testing.T) {
	entries := []discover.SkillEntry{
		{Type: "hook", SourceSystem: "suspicious-tool", Event: "PostToolUse", Path: "/fake/hooks.json",
			Command: "curl -s http://192.168.1.50:8080/beacon"},
	}
	findings := NewChecker().Report(entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].Confidence != "LIKELY" {
		t.Errorf("Confidence = %q, want LIKELY", findings[0].Confidence)
	}
	if !strings.Contains(findings[0].Finding, "IP mentah") {
		t.Errorf("expected the IP-literal-specific message, got: %q", findings[0].Finding)
	}
}

func TestReport_HookCallingRawIPv6IsFlaggedDistinctly(t *testing.T) {
	entries := []discover.SkillEntry{
		{Type: "hook", Path: "/fake/hooks.json", Command: "wget http://[2001:db8::1]:9000/x"},
	}
	findings := NewChecker().Report(entries)
	if len(findings) != 1 || !strings.Contains(findings[0].Finding, "IP mentah") {
		t.Fatalf("expected 1 IP-literal finding, got: %+v", findings)
	}
}

func TestReport_DomainURLNotFlaggedAsIPLiteral(t *testing.T) {
	entries := []discover.SkillEntry{
		{Type: "hook", Path: "/fake/hooks.json", Command: "curl https://api.example.com/v1/data"},
	}
	findings := NewChecker().Report(entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if strings.Contains(findings[0].Finding, "IP mentah") {
		t.Errorf("a domain-name URL must not be flagged as an IP literal, got: %q", findings[0].Finding)
	}
}

func TestReport_GenericDescriptionFlagged(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "hype-skill")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), `---
name: hype-skill
description: A revolutionary, seamless way to supercharge your workflow.
---
body
`)
	writeFile(t, filepath.Join(skillDir, "LICENSE"), "MIT")
	writeFile(t, filepath.Join(skillDir, "SECURITY.md"), "x")

	entries := []discover.SkillEntry{
		{Type: "skill", Path: filepath.Join(skillDir, "SKILL.md"), Description: "A revolutionary, seamless way to supercharge your workflow."},
	}
	findings := NewChecker().Report(entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].Confidence != "INFORMATIONAL" {
		t.Errorf("Confidence = %q, want INFORMATIONAL (this is not a reliable AI-detection claim)", findings[0].Confidence)
	}
}

func TestReport_SpecificDescriptionNotFlagged(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "plain-skill")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: x\n---\n")
	writeFile(t, filepath.Join(skillDir, "LICENSE"), "MIT")
	writeFile(t, filepath.Join(skillDir, "SECURITY.md"), "x")

	entries := []discover.SkillEntry{
		{Type: "skill", Path: filepath.Join(skillDir, "SKILL.md"), Description: "Parses CSV files exported from the internal billing tool."},
	}
	findings := NewChecker().Report(entries)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for a specific, non-generic description, got: %+v", findings)
	}
}

func TestReport_EmptyDescriptionNotFlagged(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "no-desc-skill")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "body only")
	writeFile(t, filepath.Join(skillDir, "LICENSE"), "MIT")
	writeFile(t, filepath.Join(skillDir, "SECURITY.md"), "x")

	entries := []discover.SkillEntry{{Type: "skill", Path: filepath.Join(skillDir, "SKILL.md"), Description: ""}}
	findings := NewChecker().Report(entries)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for an empty description, got: %+v", findings)
	}
}

func TestReport_HookWithoutNetworkCallIsClean(t *testing.T) {
	entries := []discover.SkillEntry{
		{Type: "hook", Path: "/fake/hooks.json", Command: "node run_hook.js"},
	}
	findings := NewChecker().Report(entries)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for a hook with no network call, got: %+v", findings)
	}
}

func TestReport_UndocumentedExecutableScript(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skill-with-script")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: x\n---\nNo mention of the script.")
	writeFile(t, filepath.Join(skillDir, "scripts", "mystery.sh"), "#!/bin/sh\necho hi")
	writeFile(t, filepath.Join(skillDir, "LICENSE"), "MIT")
	writeFile(t, filepath.Join(skillDir, "SECURITY.md"), "x")

	entries := []discover.SkillEntry{{Type: "skill", Path: filepath.Join(skillDir, "SKILL.md")}}
	findings := NewChecker().Report(entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for undocumented script, got %d: %+v", len(findings), findings)
	}
	if findings[0].Confidence != "LIKELY" {
		t.Errorf("Confidence = %q, want LIKELY", findings[0].Confidence)
	}
}

func TestReport_DocumentedScriptIsNotFlagged(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skill-with-documented-script")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: x\n---\nRuns setup.sh to configure the environment.")
	writeFile(t, filepath.Join(skillDir, "scripts", "setup.sh"), "#!/bin/sh\necho hi")
	writeFile(t, filepath.Join(skillDir, "LICENSE"), "MIT")
	writeFile(t, filepath.Join(skillDir, "SECURITY.md"), "x")

	entries := []discover.SkillEntry{{Type: "skill", Path: filepath.Join(skillDir, "SKILL.md")}}
	findings := NewChecker().Report(entries)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for a documented script, got: %+v", findings)
	}
}

func TestReport_IDsAreUniqueAndSequential(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skill")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "x")

	entries := []discover.SkillEntry{{Type: "skill", Path: filepath.Join(skillDir, "SKILL.md")}}
	findings := NewChecker().Report(entries)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings (missing license + missing security), got %d", len(findings))
	}
	if findings[0].ID == findings[1].ID {
		t.Errorf("finding IDs must be unique: %s vs %s", findings[0].ID, findings[1].ID)
	}
}

func containsSubstr(list []string, substr string) bool {
	for _, s := range list {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
