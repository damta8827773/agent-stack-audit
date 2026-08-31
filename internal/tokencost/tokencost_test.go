// SPDX-License-Identifier: MIT

package tokencost

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/damta8827773/agent-stack-audit/internal/discover"
)

func writeSkill(t *testing.T, dir, name string, size int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	content := make([]byte, size)
	for i := range content {
		content[i] = 'a'
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEstimate_AggregatesPerSource(t *testing.T) {
	dir := t.TempDir()
	eccFile := writeSkill(t, dir, "ecc.md", 400) // 400 chars / 4 = 100 tokens
	gstackFile := writeSkill(t, dir, "gstack.md", 800)

	entries := []discover.SkillEntry{
		{SourceSystem: "ecc", Type: "skill", AlwaysOn: true, Path: eccFile},
		{SourceSystem: "gstack", Type: "skill", AlwaysOn: true, Path: gstackFile},
	}

	result := NewCharRatioEstimator().Estimate(entries)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(result), result)
	}
	// sorted alphabetically: ecc before gstack
	if result[0].SourceSystem != "ecc" || result[0].EstimatedTokens != 100 {
		t.Errorf("unexpected ecc entry: %+v", result[0])
	}
	if result[1].SourceSystem != "gstack" || result[1].EstimatedTokens != 200 {
		t.Errorf("unexpected gstack entry: %+v", result[1])
	}
	for _, e := range result {
		if e.EstimationMethod != "char_ratio_per_content_type_approximate" {
			t.Errorf("EstimationMethod = %q, want char_ratio_per_content_type_approximate", e.EstimationMethod)
		}
		if !e.AlwaysOn {
			t.Errorf("AlwaysOn should be true for %s", e.SourceSystem)
		}
	}
}

func TestEstimate_SumsMultipleFilesFromSameSource(t *testing.T) {
	dir := t.TempDir()
	file1 := writeSkill(t, dir, "a.md", 400)
	file2 := writeSkill(t, dir, "b.md", 400)

	entries := []discover.SkillEntry{
		{SourceSystem: "ecc", Type: "skill", AlwaysOn: true, Path: file1},
		{SourceSystem: "ecc", Type: "skill", AlwaysOn: true, Path: file2},
	}
	result := NewCharRatioEstimator().Estimate(entries)
	if len(result) != 1 || result[0].EstimatedTokens != 200 {
		t.Fatalf("expected combined 200 tokens for ecc, got %+v", result)
	}
}

func TestEstimate_IgnoresNonAlwaysOnAndNonSkill(t *testing.T) {
	dir := t.TempDir()
	pluginFile := writeSkill(t, dir, "plugin.json", 4000)
	notAlwaysOn := writeSkill(t, dir, "ondemand.md", 4000)

	entries := []discover.SkillEntry{
		{SourceSystem: "gstack", Type: "plugin", AlwaysOn: false, Path: pluginFile},
		{SourceSystem: "gstack", Type: "skill", AlwaysOn: false, Path: notAlwaysOn},
		{SourceSystem: "gstack", Type: "hook", AlwaysOn: true, Path: pluginFile},
	}
	result := NewCharRatioEstimator().Estimate(entries)
	if len(result) != 0 {
		t.Fatalf("expected 0 entries (nothing is an always-on skill), got %+v", result)
	}
}

func TestEstimate_UnreadableFileSkippedNotFatal(t *testing.T) {
	entries := []discover.SkillEntry{
		{SourceSystem: "ghost", Type: "skill", AlwaysOn: true, Path: filepath.Join(t.TempDir(), "does-not-exist.md")},
	}
	result := NewCharRatioEstimator().Estimate(entries)
	if len(result) != 0 {
		t.Fatalf("expected 0 entries for unreadable file, got %+v", result)
	}
}

func TestEstimate_FrontmatterAndBodyUseDifferentRatios(t *testing.T) {
	dir := t.TempDir()
	fm := "name: x\ndescription: 0123456789012345\n"
	body := "0123456789012345678901234567890123456789"
	content := "---\n" + fm + "---\n" + body
	path := filepath.Join(dir, "skill.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []discover.SkillEntry{{SourceSystem: "ecc", Type: "skill", AlwaysOn: true, Path: path}}
	result := NewCharRatioEstimator().Estimate(entries)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %+v", result)
	}

	want := int(float64(len(fm))/FrontmatterCharsPerToken + float64(len(body))/BodyCharsPerToken)
	if result[0].EstimatedTokens != want {
		t.Errorf("EstimatedTokens = %d, want %d (frontmatter/body split applied)", result[0].EstimatedTokens, want)
	}
}

func TestSplitFrontmatterBody(t *testing.T) {
	fm, body := splitFrontmatterBody([]byte("---\nname: x\n---\nbody text"))
	if string(fm) != "\nname: x" {
		t.Errorf("frontmatter = %q, want %q", fm, "\nname: x")
	}
	if string(body) != "\nbody text" {
		t.Errorf("body = %q, want %q", body, "\nbody text")
	}
}

func TestSplitFrontmatterBody_NoFrontmatterReturnsAllBody(t *testing.T) {
	fm, body := splitFrontmatterBody([]byte("just plain content, no frontmatter"))
	if fm != nil {
		t.Errorf("frontmatter = %q, want nil", fm)
	}
	if string(body) != "just plain content, no frontmatter" {
		t.Errorf("body = %q, want full content", body)
	}
}

func TestSplitFrontmatterBody_UnclosedDelimiterReturnsAllBody(t *testing.T) {
	fm, body := splitFrontmatterBody([]byte("---\nname: x\nno closing delimiter"))
	if fm != nil {
		t.Errorf("frontmatter = %q, want nil for unclosed delimiter", fm)
	}
	if string(body) != "---\nname: x\nno closing delimiter" {
		t.Errorf("body = %q, want full content treated as body", body)
	}
}

func TestTotal(t *testing.T) {
	entries := []Entry{
		{SourceSystem: "a", EstimatedTokens: 100},
		{SourceSystem: "b", EstimatedTokens: 250},
	}
	if got := Total(entries); got != 350 {
		t.Errorf("Total() = %d, want 350", got)
	}
}

func TestTotal_Empty(t *testing.T) {
	if got := Total(nil); got != 0 {
		t.Errorf("Total(nil) = %d, want 0", got)
	}
}
