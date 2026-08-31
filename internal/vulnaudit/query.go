// SPDX-License-Identifier: MIT

package vulnaudit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DefaultOSVEndpoint is OSV.dev's single-package query endpoint (not the
// batch endpoint): one call per dependency is simpler to implement and
// test than batch-query-then-fetch-details, and returns full advisory
// data (severity, fixed version) in one round trip. For the dependency
// counts a single skill/plugin manifest realistically has, this is not a
// meaningful performance concern.
const DefaultOSVEndpoint = "https://api.osv.dev/v1/query"

type Finding struct {
	ID               string
	Confidence       string // always "CONFIRMED" - a database match, not a heuristic guess (Lampiran B)
	Ecosystem        string
	Package          string
	InstalledVersion string
	VulnerabilityID  string
	Severity         string // taken verbatim from OSV/GHSA data, never computed by this tool
	FixedVersion     string
	SourcePath       string
	AdvisoryURL      string
}

// Querier is the network-touching half of this package - the only part of
// agent-stack-audit that calls out to the internet, and only when the CLI
// wires it up behind --vuln-check's explicit consent prompt.
type Querier struct {
	Endpoint string
	Client   *http.Client
}

func NewQuerier() *Querier {
	return &Querier{
		Endpoint: DefaultOSVEndpoint,
		Client:   &http.Client{Timeout: 15 * time.Second},
	}
}

type osvQueryRequest struct {
	Version string     `json:"version"`
	Package osvPackage `json:"package"`
}

type osvPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

type osvQueryResponse struct {
	Vulns []osvVuln `json:"vulns"`
}

type osvVuln struct {
	ID               string         `json:"id"`
	DatabaseSpecific map[string]any `json:"database_specific"`
	Affected         []osvAffect    `json:"affected"`
}

type osvAffect struct {
	Package osvPackage `json:"package"`
	Ranges  []osvRange `json:"ranges"`
}

type osvRange struct {
	Events []osvEvent `json:"events"`
}

type osvEvent struct {
	Fixed string `json:"fixed"`
}

// Query checks each dependency against OSV.dev, one HTTP request per
// dependency. A request that fails (network error, malformed response) is
// skipped for that one dependency rather than aborting the whole scan -
// one unreachable/unrecognized package shouldn't hide findings for every
// other package that did resolve.
func (q *Querier) Query(deps []Dependency) []Finding {
	var findings []Finding
	for _, dep := range deps {
		findings = append(findings, q.queryOne(dep)...)
	}
	return findings
}

func (q *Querier) queryOne(dep Dependency) []Finding {
	body, err := json.Marshal(osvQueryRequest{
		Version: dep.Version,
		Package: osvPackage{Name: dep.Name, Ecosystem: dep.Ecosystem},
	})
	if err != nil {
		return nil
	}

	resp, err := q.Client.Post(q.Endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var parsed osvQueryResponse
	if json.NewDecoder(resp.Body).Decode(&parsed) != nil {
		return nil
	}

	var findings []Finding
	for _, v := range parsed.Vulns {
		findings = append(findings, Finding{
			ID:               fmt.Sprintf("vuln-%s", v.ID),
			Confidence:       "CONFIRMED",
			Ecosystem:        dep.Ecosystem,
			Package:          dep.Name,
			InstalledVersion: dep.Version,
			VulnerabilityID:  v.ID,
			Severity:         extractSeverity(v),
			FixedVersion:     extractFixedVersion(v),
			SourcePath:       dep.SourcePath,
			AdvisoryURL:      "https://osv.dev/vulnerability/" + v.ID,
		})
	}
	return findings
}

// extractSeverity reads whatever severity label OSV/GHSA already provides
// (top-level database_specific.severity is the simple string field most
// GHSA-backed advisories carry - confirmed against a real OSV.dev response
// during development, see internal/vulnaudit's own tests) rather than
// computing one from a CVSS vector - K.4's requirement that severity is
// the database's judgment, not this tool's.
func extractSeverity(v osvVuln) string {
	if v.DatabaseSpecific == nil {
		return ""
	}
	if s, ok := v.DatabaseSpecific["severity"].(string); ok {
		return s
	}
	return ""
}

func extractFixedVersion(v osvVuln) string {
	for _, a := range v.Affected {
		for _, r := range a.Ranges {
			for _, e := range r.Events {
				if e.Fixed != "" {
					return e.Fixed
				}
			}
		}
	}
	return ""
}
