package conflict

import (
	"sort"
	"strings"
)

// alternatives splits a Claude Code hook matcher into its literal
// alternatives ("Write|Edit|MultiEdit" -> [Edit MultiEdit Write], sorted).
// An empty matcher is treated as the same wildcard as "*" - both are
// common real-world conventions meaning "every tool" (confirmed for
// PostToolUse in claude-mem's hooks.json during FASE 0 research).
//
// This is intentionally a plain string/set heuristic, not a regex engine:
// FASE 4 explicitly scopes conflict-check to pattern matching on the
// matcher string, not static analysis of arbitrary regex semantics.
func alternatives(matcher string) []string {
	if matcher == "" || matcher == "*" {
		return []string{"*"}
	}
	parts := strings.Split(matcher, "|")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	sort.Strings(result)
	if len(result) == 0 {
		return []string{"*"}
	}
	return result
}

func isWildcard(alts []string) bool {
	return len(alts) == 1 && alts[0] == "*"
}

func equalSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// matcherRelation reports whether two matchers are effectively identical
// and/or overlapping, driving the CONFIRMED/LIKELY/INFORMATIONAL split
// from FASE 4 / Lampiran B.
func matcherRelation(a, b string) (identical, overlapping bool) {
	altA := alternatives(a)
	altB := alternatives(b)

	if isWildcard(altA) || isWildcard(altB) {
		return isWildcard(altA) && isWildcard(altB), true
	}

	if equalSets(altA, altB) {
		return true, true
	}

	for _, x := range altA {
		for _, y := range altB {
			if x == y {
				return false, true
			}
		}
	}
	return false, false
}
