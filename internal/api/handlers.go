package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/namelens/namelens/internal/core"
	"github.com/namelens/namelens/internal/core/engine"
)

// Server implements the ServerInterface for the control plane API.
type Server struct {
	orchestrator *engine.Orchestrator
	version      string
}

// Ensure Server implements ServerInterface at compile time.
var _ ServerInterface = (*Server)(nil)

// NewServer creates a new API server.
func NewServer(orchestrator *engine.Orchestrator, version string) *Server {
	return &Server{
		orchestrator: orchestrator,
		version:      version,
	}
}

// GetHealth returns the server health status.
// (GET /health)
func (s *Server) GetHealth(w http.ResponseWriter, r *http.Request) {
	// Simple health check - if we're responding, we're healthy
	// The existing /health endpoint in the server package handles detailed checks
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  Healthy,
		Version: &s.version,
	})
}

// GetStatus returns server status including rate limits.
// (GET /v1/status)
func (s *Server) GetStatus(w http.ResponseWriter, r *http.Request) {
	providers := make(map[string]ProviderStatus)

	// Report provider availability based on configured checkers
	if s.orchestrator != nil {
		for checkType := range s.orchestrator.Checkers {
			providers[string(checkType)] = ProviderStatus{
				Available: true,
			}
		}
		for name := range s.orchestrator.RegistryCheckers {
			providers[name] = ProviderStatus{
				Available: true,
			}
		}
		for name := range s.orchestrator.HandleCheckers {
			providers[name] = ProviderStatus{
				Available: true,
			}
		}
	}

	writeJSON(w, http.StatusOK, StatusResponse{
		Providers: providers,
	})
}

// CheckName performs a name availability check.
// (POST /v1/check)
func (s *Server) CheckName(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "bad_request", "invalid JSON: "+err.Error())
		return
	}

	// Validate name
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeErrorJSON(w, http.StatusBadRequest, "bad_request", "name is required")
		return
	}
	if len(name) > 63 {
		writeErrorJSON(w, http.StatusBadRequest, "bad_request", "name exceeds maximum length of 63 characters")
		return
	}

	// Build profile from request
	profile, err := s.buildProfile(req.Profile, req.Tlds, req.Registries, req.Handles)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	// Run checks
	results, err := s.orchestrator.Check(r.Context(), name, profile)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// Convert results
	apiResults := make([]CheckResult, 0, len(results))
	for _, result := range results {
		apiResults = append(apiResults, toAPICheckResult(result))
	}

	// Calculate summary
	summary := calculateSummary(results)

	response := CheckResponse{
		Name:    name,
		Results: apiResults,
		Summary: summary,
	}

	// TODO: Add expert analysis if req.Expert is true

	writeJSON(w, http.StatusOK, response)
}

// CompareCandidates compares multiple name candidates.
// (POST /v1/compare)
func (s *Server) CompareCandidates(w http.ResponseWriter, r *http.Request) {
	var req CompareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "bad_request", "invalid JSON: "+err.Error())
		return
	}

	// Validate names
	if len(req.Names) < 2 {
		writeErrorJSON(w, http.StatusBadRequest, "bad_request", "at least 2 names required for comparison")
		return
	}
	if len(req.Names) > 10 {
		writeErrorJSON(w, http.StatusBadRequest, "bad_request", "maximum 10 names for comparison")
		return
	}

	// Build profile
	profile, err := s.buildCompareProfile(req.Profile, req.Tlds, req.Registries, req.Handles)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	candidates := make([]CompareCandidate, 0, len(req.Names))

	for _, name := range req.Names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		results, err := s.orchestrator.Check(r.Context(), name, profile)
		if err != nil {
			// Include error in results
			candidates = append(candidates, CompareCandidate{
				Name:    name,
				Results: []CheckResult{},
				Summary: CheckSummary{Total: 0},
			})
			continue
		}

		apiResults := make([]CheckResult, 0, len(results))
		for _, result := range results {
			apiResults = append(apiResults, toAPICheckResult(result))
		}

		candidates = append(candidates, CompareCandidate{
			Name:    name,
			Results: apiResults,
			Summary: calculateSummary(results),
		})
	}

	writeJSON(w, http.StatusOK, CompareResponse{
		Candidates: candidates,
	})
}

// ListProfiles returns all available check profiles.
// (GET /v1/profiles)
func (s *Server) ListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles := make([]Profile, 0, len(core.BuiltInProfiles))

	for _, p := range core.BuiltInProfiles {
		isBuiltin := true
		profiles = append(profiles, Profile{
			Name:        p.Name,
			Description: &p.Description,
			Tlds:        &p.TLDs,
			Registries:  &p.Registries,
			Handles:     &p.Handles,
			IsBuiltin:   &isBuiltin,
		})
	}

	writeJSON(w, http.StatusOK, ProfileListResponse{
		Profiles: profiles,
	})
}

// buildProfile constructs a core.Profile from API request parameters.
// Returns an error if an invalid profile name is specified.
func (s *Server) buildProfile(
	profileName *CheckRequestProfile,
	tlds *[]string,
	registries *[]CheckRequestRegistries,
	handles *[]CheckRequestHandles,
) (core.Profile, error) {
	// Start with a named profile if specified
	var profile core.Profile
	if profileName != nil {
		p, ok := core.FindBuiltInProfile(string(*profileName))
		if !ok {
			return core.Profile{}, fmt.Errorf("invalid profile: %s", string(*profileName))
		}
		profile = *p
	}

	// Override with custom values if provided
	if tlds != nil {
		profile.TLDs = *tlds
	}
	if registries != nil {
		regs := make([]string, len(*registries))
		for i, r := range *registries {
			regs[i] = string(r)
		}
		profile.Registries = regs
	}
	if handles != nil {
		hdls := make([]string, len(*handles))
		for i, h := range *handles {
			hdls[i] = string(h)
		}
		profile.Handles = hdls
	}

	// Default to minimal profile if nothing specified
	if len(profile.TLDs) == 0 && len(profile.Registries) == 0 && len(profile.Handles) == 0 {
		if p, ok := core.FindBuiltInProfile("minimal"); ok {
			profile = *p
		}
	}

	return profile, nil
}

// buildCompareProfile is like buildProfile but for CompareRequest types.
// Returns an error if an invalid profile name is specified.
func (s *Server) buildCompareProfile(
	profileName *CompareRequestProfile,
	tlds *[]string,
	registries *[]CompareRequestRegistries,
	handles *[]CompareRequestHandles,
) (core.Profile, error) {
	var profile core.Profile
	if profileName != nil {
		p, ok := core.FindBuiltInProfile(string(*profileName))
		if !ok {
			return core.Profile{}, fmt.Errorf("invalid profile: %s", string(*profileName))
		}
		profile = *p
	}

	if tlds != nil {
		profile.TLDs = *tlds
	}
	if registries != nil {
		regs := make([]string, len(*registries))
		for i, r := range *registries {
			regs[i] = string(r)
		}
		profile.Registries = regs
	}
	if handles != nil {
		hdls := make([]string, len(*handles))
		for i, h := range *handles {
			hdls[i] = string(h)
		}
		profile.Handles = hdls
	}

	if len(profile.TLDs) == 0 && len(profile.Registries) == 0 && len(profile.Handles) == 0 {
		if p, ok := core.FindBuiltInProfile("minimal"); ok {
			profile = *p
		}
	}

	return profile, nil
}

// toAPICheckResult converts a core.CheckResult to an API CheckResult.
func toAPICheckResult(result *core.CheckResult) CheckResult {
	if result == nil {
		return CheckResult{}
	}

	available := availabilityToString(result.Available)
	checkType := CheckResultCheckType(result.CheckType)

	apiResult := CheckResult{
		Name:      result.Name,
		CheckType: checkType,
		Available: available,
	}

	if result.TLD != "" {
		apiResult.Tld = &result.TLD
	}
	if result.Message != "" {
		apiResult.Message = &result.Message
	}

	// Convert provenance
	prov := Provenance{
		RequestedAt: &result.Provenance.RequestedAt,
		ResolvedAt:  &result.Provenance.ResolvedAt,
		FromCache:   &result.Provenance.FromCache,
	}
	if result.Provenance.Source != "" {
		prov.Source = &result.Provenance.Source
	}
	if result.Provenance.Server != "" {
		prov.Server = &result.Provenance.Server
	}
	if result.Provenance.CacheExpiresAt != nil {
		prov.CacheExpiresAt = result.Provenance.CacheExpiresAt
	}
	apiResult.Provenance = &prov

	return apiResult
}

// availabilityToString converts core.Availability to API string.
func availabilityToString(a core.Availability) CheckResultAvailable {
	switch a {
	case core.AvailabilityAvailable:
		return CheckResultAvailableAvailable
	case core.AvailabilityTaken:
		return CheckResultAvailableTaken
	case core.AvailabilityError:
		return CheckResultAvailableError
	case core.AvailabilityRateLimited:
		return CheckResultAvailableRateLimited
	case core.AvailabilityUnsupported:
		return CheckResultAvailableUnsupported
	default:
		return CheckResultAvailableUnknown
	}
}

// calculateSummary computes a CheckSummary from check results.
func calculateSummary(results []*core.CheckResult) CheckSummary {
	summary := CheckSummary{
		Total: len(results),
	}

	for _, r := range results {
		if r == nil {
			summary.Unknown++
			continue
		}
		switch r.Available {
		case core.AvailabilityAvailable:
			summary.Available++
		case core.AvailabilityTaken:
			summary.Taken++
		default:
			summary.Unknown++
		}
	}

	// Determine risk level
	riskLevel := deriveRiskLevel(results)
	summary.RiskLevel = &riskLevel

	return summary
}

// deriveRiskLevel determines risk based on check results.
func deriveRiskLevel(results []*core.CheckResult) CheckSummaryRiskLevel {
	if len(results) == 0 {
		return CheckSummaryRiskLevelLow
	}

	// High risk if .com is taken
	for _, r := range results {
		if r == nil {
			continue
		}
		if r.CheckType == core.CheckTypeDomain && strings.HasSuffix(r.Name, ".com") {
			if r.Available == core.AvailabilityTaken {
				return CheckSummaryRiskLevelHigh
			}
		}
	}

	// Medium risk if any asset is taken
	for _, r := range results {
		if r == nil {
			continue
		}
		if r.Available == core.AvailabilityTaken {
			return CheckSummaryRiskLevelMedium
		}
	}

	return CheckSummaryRiskLevelLow
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeErrorJSON writes a JSON error response.
func writeErrorJSON(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: Error{
			Code:    code,
			Message: message,
		},
	})
}
