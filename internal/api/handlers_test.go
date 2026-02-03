package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/namelens/namelens/internal/core"
	"github.com/namelens/namelens/internal/core/engine"
)

// Ensure core.CheckType is used in tests
var _ = core.CheckTypeDomain

func TestGetHealth(t *testing.T) {
	srv := NewServer(nil, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.GetHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != Healthy {
		t.Errorf("expected status %q, got %q", Healthy, resp.Status)
	}

	if resp.Version == nil || *resp.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %v", resp.Version)
	}
}

func TestListProfiles(t *testing.T) {
	srv := NewServer(nil, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	rec := httptest.NewRecorder()

	srv.ListProfiles(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp ProfileListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Profiles) == 0 {
		t.Error("expected at least one profile")
	}

	// Check that startup profile exists
	found := false
	for _, p := range resp.Profiles {
		if p.Name == "startup" {
			found = true
			if p.IsBuiltin == nil || !*p.IsBuiltin {
				t.Error("expected startup profile to be builtin")
			}
			break
		}
	}
	if !found {
		t.Error("startup profile not found")
	}
}

func TestGetStatus(t *testing.T) {
	orchestrator := &engine.Orchestrator{
		Checkers:         make(map[core.CheckType]engine.Checker),
		RegistryCheckers: make(map[string]engine.Checker),
		HandleCheckers:   make(map[string]engine.Checker),
	}

	srv := NewServer(orchestrator, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	rec := httptest.NewRecorder()

	srv.GetStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp StatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Providers == nil {
		t.Error("expected providers map")
	}
}

func TestCheckNameValidation(t *testing.T) {
	srv := NewServer(&engine.Orchestrator{
		Checkers: make(map[core.CheckType]engine.Checker),
	}, "1.0.0")

	tests := []struct {
		name     string
		body     string
		wantCode int
		wantErr  string
	}{
		{
			name:     "empty name",
			body:     `{"name":""}`,
			wantCode: http.StatusBadRequest,
			wantErr:  "name is required",
		},
		{
			name:     "name too long",
			body:     `{"name":"` + string(make([]byte, 100)) + `"}`,
			wantCode: http.StatusBadRequest,
			wantErr:  "exceeds maximum length",
		},
		{
			name:     "invalid json",
			body:     `{invalid`,
			wantCode: http.StatusBadRequest,
			wantErr:  "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/check", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			srv.CheckName(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("expected status %d, got %d", tt.wantCode, rec.Code)
			}

			var resp ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if resp.Error.Message == "" {
				t.Error("expected error message")
			}
		})
	}
}

func TestCompareCandidatesValidation(t *testing.T) {
	srv := NewServer(&engine.Orchestrator{
		Checkers: make(map[core.CheckType]engine.Checker),
	}, "1.0.0")

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "too few names",
			body:     `{"names":["one"]}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "too many names",
			body:     `{"names":["a","b","c","d","e","f","g","h","i","j","k"]}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/compare", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			srv.CompareCandidates(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("expected status %d, got %d", tt.wantCode, rec.Code)
			}
		})
	}
}
