// SPDX-License-Identifier: MIT

package vulnaudit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mockOSVServer(t *testing.T, handler http.HandlerFunc) *Querier {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Querier{Endpoint: srv.URL, Client: srv.Client()}
}

func TestQuery_FindsVulnerability(t *testing.T) {
	q := mockOSVServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req osvQueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("bad request body: %v", err)
		}
		if req.Package.Name != "left-pad" || req.Package.Ecosystem != "npm" || req.Version != "1.3.0" {
			t.Errorf("unexpected request: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(osvQueryResponse{
			Vulns: []osvVuln{
				{
					ID:               "GHSA-xxxx-xxxx-xxxx",
					DatabaseSpecific: map[string]any{"severity": "HIGH"},
					Affected: []osvAffect{
						{
							Ranges: []osvRange{
								{Events: []osvEvent{{Fixed: "1.3.1"}}},
							},
						},
					},
				},
			},
		})
	})

	findings := q.Query([]Dependency{
		{Ecosystem: "npm", Name: "left-pad", Version: "1.3.0", SourcePath: "/fake/package.json"},
	})

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Confidence != "CONFIRMED" {
		t.Errorf("Confidence = %q, want CONFIRMED", f.Confidence)
	}
	if f.VulnerabilityID != "GHSA-xxxx-xxxx-xxxx" {
		t.Errorf("VulnerabilityID = %q", f.VulnerabilityID)
	}
	if f.Severity != "HIGH" {
		t.Errorf("Severity = %q, want HIGH (must come from the response, not be computed)", f.Severity)
	}
	if f.FixedVersion != "1.3.1" {
		t.Errorf("FixedVersion = %q, want 1.3.1", f.FixedVersion)
	}
	if f.AdvisoryURL != "https://osv.dev/vulnerability/GHSA-xxxx-xxxx-xxxx" {
		t.Errorf("AdvisoryURL = %q", f.AdvisoryURL)
	}
	if f.SourcePath != "/fake/package.json" {
		t.Errorf("SourcePath = %q, want the dependency's own source path", f.SourcePath)
	}
}

func TestQuery_NoVulnerabilitiesReturnsEmpty(t *testing.T) {
	q := mockOSVServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(osvQueryResponse{})
	})

	findings := q.Query([]Dependency{{Ecosystem: "Go", Name: "example.com/clean", Version: "1.0.0"}})
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %+v", findings)
	}
}

func TestQuery_ServerErrorSkippedNotFatal(t *testing.T) {
	q := mockOSVServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	findings := q.Query([]Dependency{{Ecosystem: "npm", Name: "x", Version: "1.0.0"}})
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings on server error (skipped, not panicked), got %+v", findings)
	}
}

func TestQuery_MalformedResponseSkippedNotFatal(t *testing.T) {
	q := mockOSVServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	})

	findings := q.Query([]Dependency{{Ecosystem: "npm", Name: "x", Version: "1.0.0"}})
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for malformed response, got %+v", findings)
	}
}

func TestQuery_MultipleDependenciesEachQueried(t *testing.T) {
	callCount := 0
	q := mockOSVServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(osvQueryResponse{})
	})

	q.Query([]Dependency{
		{Ecosystem: "npm", Name: "a", Version: "1.0.0"},
		{Ecosystem: "PyPI", Name: "b", Version: "2.0.0"},
		{Ecosystem: "Go", Name: "c", Version: "3.0.0"},
	})

	if callCount != 3 {
		t.Errorf("expected 3 separate requests (one per dependency), got %d", callCount)
	}
}

func TestQuery_EmptyInput(t *testing.T) {
	q := NewQuerier()
	if f := q.Query(nil); len(f) != 0 {
		t.Fatalf("expected 0 findings for nil input, got %+v", f)
	}
}

func TestQuery_SeverityFallsBackWhenAbsent(t *testing.T) {
	q := mockOSVServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(osvQueryResponse{
			Vulns: []osvVuln{{ID: "OSV-1", Affected: []osvAffect{{}}}},
		})
	})
	findings := q.Query([]Dependency{{Ecosystem: "Go", Name: "x", Version: "1.0.0"}})
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %+v", findings)
	}
	if findings[0].Severity != "" {
		t.Errorf("expected empty Severity when the response has none, got %q (must never invent one)", findings[0].Severity)
	}
}

func TestNewQuerier_DefaultsToRealEndpoint(t *testing.T) {
	q := NewQuerier()
	if q.Endpoint != DefaultOSVEndpoint {
		t.Errorf("Endpoint = %q, want %q", q.Endpoint, DefaultOSVEndpoint)
	}
	if q.Client == nil {
		t.Error("Client should not be nil")
	}
}
