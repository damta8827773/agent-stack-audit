// SPDX-License-Identifier: MIT

// Package vulnaudit matches dependency manifests found inside discovered
// skills/plugins against OSV.dev's public vulnerability database - never
// static/dynamic code analysis, purely a version-vs-known-advisory match
// (Lampiran K.2/K.7). Manifest discovery/parsing (this file) is pure local
// file reading with zero network calls; querying OSV (query.go) is the
// only network-touching code in this package, and it is never invoked
// without the CLI's explicit --vuln-check flag and interactive consent.
package vulnaudit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
)

// Dependency is one package+version pair extracted from a manifest,
// resolved to an OSV ecosystem name.
type Dependency struct {
	Ecosystem  string // OSV ecosystem name: "Go", "npm", "PyPI"
	Name       string
	Version    string
	SourcePath string
}

// FindManifests looks for known dependency manifest files directly inside
// each given directory (not recursive - a skill/plugin's manifest lives at
// its own root, same assumption FASE 3's discover makes about SKILL.md)
// and parses whichever ones it finds. A missing or unparseable manifest is
// skipped, never fatal - consistent with every other module's "malformed
// input is a warning, not an error" handling.
func FindManifests(dirs []string) []Dependency {
	var deps []Dependency
	seen := map[string]bool{}
	for _, dir := range dirs {
		if seen[dir] {
			continue
		}
		seen[dir] = true
		deps = append(deps, parseGoMod(dir)...)
		deps = append(deps, parseNodeManifest(dir)...)
		deps = append(deps, parseRequirementsTxt(dir)...)
		deps = append(deps, parsePipfileLock(dir)...)
		deps = append(deps, parseComposerManifest(dir)...)
		deps = append(deps, parseGemfileLock(dir)...)
	}
	return deps
}

func parseGoMod(dir string) []Dependency {
	path := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	f, err := modfile.Parse(path, data, nil)
	if err != nil {
		return nil
	}
	var deps []Dependency
	for _, req := range f.Require {
		if req.Indirect {
			continue // indirect deps aren't something this skill/plugin chose directly
		}
		deps = append(deps, Dependency{
			Ecosystem:  "Go",
			Name:       req.Mod.Path,
			Version:    strings.TrimPrefix(req.Mod.Version, "v"),
			SourcePath: path,
		})
	}
	return deps
}

// parseNodeManifest prefers package-lock.json (exact installed versions)
// over package.json (declared ranges like "^1.2.3", which aren't
// necessarily what's actually installed) when both are present.
func parseNodeManifest(dir string) []Dependency {
	lockPath := filepath.Join(dir, "package-lock.json")
	if data, err := os.ReadFile(lockPath); err == nil {
		if deps := parsePackageLock(data, lockPath); deps != nil {
			return deps
		}
	}

	pkgPath := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return nil
	}
	var deps []Dependency
	for name, rangeSpec := range pkg.Dependencies {
		if v := exactVersion(rangeSpec); v != "" {
			deps = append(deps, Dependency{Ecosystem: "npm", Name: name, Version: v, SourcePath: pkgPath})
		}
	}
	for name, rangeSpec := range pkg.DevDependencies {
		if v := exactVersion(rangeSpec); v != "" {
			deps = append(deps, Dependency{Ecosystem: "npm", Name: name, Version: v, SourcePath: pkgPath})
		}
	}
	return deps
}

// exactVersion returns rangeSpec with a leading ^/~/>=/<=/=/> stripped if
// what's left looks like a plain version - it does NOT resolve semver
// ranges to a real installed version. Anything that still isn't a plain
// dotted version after stripping (a range like ">=1.0 <2.0", "*", a git
// URL) is skipped rather than guessed at, since querying OSV with a wrong
// guessed version would be worse than not checking that package at all.
func exactVersion(rangeSpec string) string {
	v := strings.TrimLeft(rangeSpec, "^~=<>")
	v = strings.TrimSpace(v)
	if v == "" || strings.ContainsAny(v, " |*x") {
		return ""
	}
	for _, r := range v {
		if !(r >= '0' && r <= '9') && r != '.' && r != '-' {
			return ""
		}
	}
	return v
}

func parsePackageLock(data []byte, path string) []Dependency {
	// npm 7+ lockfile format ("lockfileVersion" 2 or 3): a flat
	// "packages" map keyed by node_modules path, each with a "version".
	var modern struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
	}
	if json.Unmarshal(data, &modern) == nil && len(modern.Packages) > 0 {
		var deps []Dependency
		for pkgPath, info := range modern.Packages {
			if pkgPath == "" || info.Version == "" {
				continue // "" is the root project itself, not a dependency
			}
			name := strings.TrimPrefix(pkgPath, "node_modules/")
			deps = append(deps, Dependency{Ecosystem: "npm", Name: name, Version: info.Version, SourcePath: path})
		}
		return deps
	}

	// Legacy (lockfileVersion 1) format: nested "dependencies" map.
	var legacy struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if json.Unmarshal(data, &legacy) == nil && len(legacy.Dependencies) > 0 {
		var deps []Dependency
		for name, info := range legacy.Dependencies {
			if info.Version == "" {
				continue
			}
			deps = append(deps, Dependency{Ecosystem: "npm", Name: name, Version: info.Version, SourcePath: path})
		}
		return deps
	}
	return nil
}

// parseRequirementsTxt only extracts exact pins ("name==1.2.3") - every
// other operator (>=, ~=, a bare "name" with no version) means the actual
// installed version isn't knowable from this file alone, so it's skipped
// rather than guessed at (same reasoning as exactVersion above).
func parseRequirementsTxt(dir string) []Dependency {
	path := filepath.Join(dir, "requirements.txt")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		if idx := strings.Index(line, "=="); idx > 0 {
			name := strings.TrimSpace(line[:idx])
			if bracket := strings.Index(name, "["); bracket >= 0 {
				name = name[:bracket] // strip extras, e.g. "requests[socks]"
			}
			version := strings.TrimSpace(line[idx+2:])
			// Drop any trailing environment marker / inline comment.
			if semi := strings.IndexAny(version, " ;#"); semi >= 0 {
				version = version[:semi]
			}
			if name != "" && version != "" {
				deps = append(deps, Dependency{Ecosystem: "PyPI", Name: name, Version: version, SourcePath: path})
			}
		}
	}
	return deps
}

// parsePipfileLock reads Pipfile.lock's "default" and "develop" sections.
// Unlike requirements.txt, every entry here is already a resolved lock
// ("==x.y.z"), not a declared range - the only entries skipped are ones
// with no "version" field at all (a VCS/editable/local-path install,
// which has no PyPI version to look up).
func parsePipfileLock(dir string) []Dependency {
	path := filepath.Join(dir, "Pipfile.lock")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var lock struct {
		Default map[string]struct {
			Version string `json:"version"`
		} `json:"default"`
		Develop map[string]struct {
			Version string `json:"version"`
		} `json:"develop"`
	}
	if json.Unmarshal(data, &lock) != nil {
		return nil
	}
	var deps []Dependency
	for name, info := range lock.Default {
		if v := strings.TrimPrefix(info.Version, "=="); v != "" {
			deps = append(deps, Dependency{Ecosystem: "PyPI", Name: name, Version: v, SourcePath: path})
		}
	}
	for name, info := range lock.Develop {
		if v := strings.TrimPrefix(info.Version, "=="); v != "" {
			deps = append(deps, Dependency{Ecosystem: "PyPI", Name: name, Version: v, SourcePath: path})
		}
	}
	return deps
}

// parseComposerManifest mirrors parseNodeManifest's lockfile-over-manifest
// preference: composer.lock has exact installed versions, composer.json's
// "require"/"require-dev" only has declared ranges ("^2.5"). PHP itself
// and extension pseudo-packages ("php", "ext-json", ...) are filtered out
// - they have no "/" in their name and aren't real Packagist packages, so
// querying OSV for them would be meaningless.
func parseComposerManifest(dir string) []Dependency {
	lockPath := filepath.Join(dir, "composer.lock")
	if data, err := os.ReadFile(lockPath); err == nil {
		if deps := parseComposerLock(data, lockPath); deps != nil {
			return deps
		}
	}

	manifestPath := filepath.Join(dir, "composer.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil
	}
	var manifest struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return nil
	}
	var deps []Dependency
	for name, rangeSpec := range manifest.Require {
		if !strings.Contains(name, "/") {
			continue // "php", "ext-*": platform requirements, not Packagist packages
		}
		if v := exactVersion(rangeSpec); v != "" {
			deps = append(deps, Dependency{Ecosystem: "Packagist", Name: name, Version: v, SourcePath: manifestPath})
		}
	}
	for name, rangeSpec := range manifest.RequireDev {
		if !strings.Contains(name, "/") {
			continue
		}
		if v := exactVersion(rangeSpec); v != "" {
			deps = append(deps, Dependency{Ecosystem: "Packagist", Name: name, Version: v, SourcePath: manifestPath})
		}
	}
	return deps
}

func parseComposerLock(data []byte, path string) []Dependency {
	var lock struct {
		Packages []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"packages"`
		PackagesDev []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"packages-dev"`
	}
	if json.Unmarshal(data, &lock) != nil {
		return nil
	}
	var deps []Dependency
	for _, p := range append(lock.Packages, lock.PackagesDev...) {
		if p.Name == "" || p.Version == "" {
			continue
		}
		// Composer versions are sometimes tagged "v2.5.0" - strip the
		// leading "v" the same way parseGoMod strips Go's "v" prefix.
		v := strings.TrimPrefix(p.Version, "v")
		deps = append(deps, Dependency{Ecosystem: "Packagist", Name: p.Name, Version: v, SourcePath: path})
	}
	return deps
}

// gemfileLockSpecPattern matches exactly a top-level resolved spec line
// inside Gemfile.lock's "specs:" block: 4 spaces of indent, a gem name, a
// version in parens. A dependency-constraint line one level deeper (e.g.
// "      activesupport (= 7.0.4)" under "actionpack (7.0.4)") has 6+
// spaces of indent and deliberately does not match - it names a
// constraint on another gem, not a second resolved entry for it.
var gemfileLockSpecPattern = regexp.MustCompile(`^    (\S+) \(([^)]+)\)\s*$`)

// parseGemfileLock reads every "specs:" block in Gemfile.lock (there can
// be more than one - a GEM source and a GIT source each have their own).
// Every top-level spec line is already a resolved, exact version - this
// is a lockfile, not a manifest with ranges.
func parseGemfileLock(dir string) []Dependency {
	path := filepath.Join(dir, "Gemfile.lock")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []Dependency
	inSpecs := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "  specs:" {
			inSpecs = true
			continue
		}
		if !inSpecs {
			continue
		}
		if trimmed == "" || !strings.HasPrefix(line, "    ") {
			inSpecs = false // dedent out of the specs: block (blank line or a new top-level section)
			continue
		}
		if m := gemfileLockSpecPattern.FindStringSubmatch(line); m != nil {
			deps = append(deps, Dependency{Ecosystem: "RubyGems", Name: m[1], Version: m[2], SourcePath: path})
		}
	}
	return deps
}
