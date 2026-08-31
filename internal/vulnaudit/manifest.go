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
