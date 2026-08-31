// SPDX-License-Identifier: MIT

package vulnaudit

import (
	"os"
	"path/filepath"
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

func TestFindManifests_GoMod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), `module example.com/skill

go 1.25

require (
	github.com/some/dep v1.2.3
	github.com/other/dep v0.5.0
)

require github.com/indirect/dep v1.0.0 // indirect
`)

	deps := FindManifests([]string{dir})
	if len(deps) != 2 {
		t.Fatalf("expected 2 direct deps (indirect excluded), got %d: %+v", len(deps), deps)
	}
	byName := map[string]Dependency{}
	for _, d := range deps {
		byName[d.Name] = d
	}
	if d, ok := byName["github.com/some/dep"]; !ok || d.Version != "1.2.3" || d.Ecosystem != "Go" {
		t.Errorf("unexpected entry for github.com/some/dep: %+v (found=%v)", d, ok)
	}
	if _, ok := byName["github.com/indirect/dep"]; ok {
		t.Errorf("indirect dependency should be excluded, got: %+v", deps)
	}
}

func TestFindManifests_GoMod_Malformed(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "this is not a valid go.mod file {{{")
	deps := FindManifests([]string{dir})
	if len(deps) != 0 {
		t.Fatalf("expected 0 deps for malformed go.mod (skip, not panic), got %+v", deps)
	}
}

func TestFindManifests_PackageJSONWithoutLockfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{
		"dependencies": {"left-pad": "1.3.0", "lodash": "^4.17.21"},
		"devDependencies": {"jest": "~29.0.0"}
	}`)

	deps := FindManifests([]string{dir})
	byName := map[string]Dependency{}
	for _, d := range deps {
		byName[d.Name] = d
	}
	if d, ok := byName["left-pad"]; !ok || d.Version != "1.3.0" || d.Ecosystem != "npm" {
		t.Errorf("unexpected entry for left-pad: %+v (found=%v)", d, ok)
	}
	if d, ok := byName["lodash"]; !ok || d.Version != "4.17.21" {
		t.Errorf("expected ^ stripped from lodash version, got: %+v (found=%v)", d, ok)
	}
	if d, ok := byName["jest"]; !ok || d.Version != "29.0.0" {
		t.Errorf("expected ~ stripped from jest version, got: %+v (found=%v)", d, ok)
	}
}

func TestFindManifests_PackageJSON_UnresolvableRangeSkipped(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{
		"dependencies": {"weird": ">=1.0.0 <2.0.0", "wild": "*", "gitdep": "git+https://example.com/x.git"}
	}`)
	deps := FindManifests([]string{dir})
	if len(deps) != 0 {
		t.Fatalf("expected 0 deps (none of these ranges resolve to an exact version), got %+v", deps)
	}
}

func TestFindManifests_PackageLockPreferredOverPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"left-pad": "^1.0.0"}}`)
	writeFile(t, filepath.Join(dir, "package-lock.json"), `{
		"lockfileVersion": 3,
		"packages": {
			"": {"name": "root"},
			"node_modules/left-pad": {"version": "1.3.0"}
		}
	}`)

	deps := FindManifests([]string{dir})
	if len(deps) != 1 {
		t.Fatalf("expected 1 dep from the lockfile (not package.json's range), got %+v", deps)
	}
	if deps[0].Name != "left-pad" || deps[0].Version != "1.3.0" {
		t.Errorf("unexpected dep: %+v", deps[0])
	}
}

func TestFindManifests_LegacyPackageLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package-lock.json"), `{
		"lockfileVersion": 1,
		"dependencies": {
			"left-pad": {"version": "1.3.0"}
		}
	}`)
	deps := FindManifests([]string{dir})
	if len(deps) != 1 || deps[0].Name != "left-pad" || deps[0].Version != "1.3.0" {
		t.Fatalf("unexpected deps for legacy lockfile: %+v", deps)
	}
}

func TestFindManifests_RequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "requirements.txt"), `# a comment
requests==2.31.0
flask>=2.0.0
django~=4.2
numpy
requests[socks]==2.31.0
-r other-requirements.txt

pillow==10.0.1  # inline comment
`)
	deps := FindManifests([]string{dir})
	byName := map[string]Dependency{}
	for _, d := range deps {
		byName[d.Name] = d
	}
	if d, ok := byName["requests"]; !ok || d.Version != "2.31.0" || d.Ecosystem != "PyPI" {
		t.Errorf("unexpected entry for requests: %+v (found=%v)", d, ok)
	}
	if d, ok := byName["pillow"]; !ok || d.Version != "10.0.1" {
		t.Errorf("expected inline comment stripped from pillow version, got: %+v (found=%v)", d, ok)
	}
	if _, ok := byName["flask"]; ok {
		t.Errorf("flask uses >=, no exact version knowable, should be skipped")
	}
	if _, ok := byName["django"]; ok {
		t.Errorf("django uses ~=, no exact version knowable, should be skipped")
	}
	if _, ok := byName["numpy"]; ok {
		t.Errorf("numpy has no version at all, should be skipped")
	}
}

func TestFindManifests_MissingFilesAreNotErrors(t *testing.T) {
	dir := t.TempDir() // completely empty
	deps := FindManifests([]string{dir})
	if len(deps) != 0 {
		t.Fatalf("expected 0 deps for a dir with no manifests, got %+v", deps)
	}
}

func TestFindManifests_MultipleDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	writeFile(t, filepath.Join(dir1, "requirements.txt"), "a==1.0.0\n")
	writeFile(t, filepath.Join(dir2, "requirements.txt"), "b==2.0.0\n")

	deps := FindManifests([]string{dir1, dir2})
	if len(deps) != 2 {
		t.Fatalf("expected 2 deps across both dirs, got %+v", deps)
	}
}

func TestFindManifests_DuplicateDirsScannedOnce(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "requirements.txt"), "a==1.0.0\n")
	deps := FindManifests([]string{dir, dir, dir})
	if len(deps) != 1 {
		t.Fatalf("expected the duplicate dir to only be scanned once, got %+v", deps)
	}
}
