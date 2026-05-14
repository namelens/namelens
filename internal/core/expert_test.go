package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink"
)

func TestNewExpertDigest_NilWhenNeitherResponseNorError(t *testing.T) {
	require.Nil(t, NewExpertDigest("alpha", nil, nil, nil))
}

func TestNewExpertDigest_SuccessShape(t *testing.T) {
	results := []*CheckResult{
		{CheckType: CheckTypeDomain, Name: "alpha.com", TLD: "com", Available: AvailabilityTaken},
	}
	resp := &ailink.SearchResponse{RiskLevel: "low", Summary: "looks clear"}

	digest := NewExpertDigest("alpha", results, resp, nil)
	require.NotNil(t, digest)
	require.Equal(t, "alpha", digest.Name)
	require.Equal(t, "risk: low", digest.Status)
	require.Equal(t, "looks clear", digest.Notes)
	require.Equal(t, "low", digest.RiskLevel)
}

func TestNewExpertDigest_UnknownRiskLevel(t *testing.T) {
	resp := &ailink.SearchResponse{Summary: "no level"}
	digest := NewExpertDigest("alpha", nil, resp, nil)
	require.NotNil(t, digest)
	require.Equal(t, "risk: unknown", digest.Status)
	require.Equal(t, "", digest.RiskLevel)
}

func TestNewExpertDigest_EmptySummaryWithRawHint(t *testing.T) {
	resp := &ailink.SearchResponse{RiskLevel: "medium", Raw: json.RawMessage(`{"foo":1}`)}
	digest := NewExpertDigest("alpha", nil, resp, nil)
	require.NotNil(t, digest)
	require.Equal(t, "expert analysis complete (see raw JSON in --output=json)", digest.Notes)
}

func TestNewExpertDigest_ErrorShape(t *testing.T) {
	errResp := &ailink.SearchError{Code: "timeout", Message: "request timed out"}
	digest := NewExpertDigest("alpha", nil, nil, errResp)
	require.NotNil(t, digest)
	require.Equal(t, "error", digest.Status)
	require.Equal(t, "request timed out", digest.Notes)
	require.Equal(t, "", digest.RiskLevel)
}

func TestNewExpertDigest_ErrorFallsBackToDetailsWhenMessageEmpty(t *testing.T) {
	errResp := &ailink.SearchError{Code: "x", Message: "  ", Details: "deep diagnostic"}
	digest := NewExpertDigest("alpha", nil, nil, errResp)
	require.NotNil(t, digest)
	require.Equal(t, "deep diagnostic", digest.Notes)
}

func TestExpertDisplayName_PrefersBatchNameWhenItAppearsInResults(t *testing.T) {
	results := []*CheckResult{
		{CheckType: CheckTypeDomain, Name: "alpha.com", TLD: "com"},
	}
	require.Equal(t, "alpha", ExpertDisplayName("alpha", results))
}

func TestExpertDisplayName_InfersWhenBatchNameMismatches(t *testing.T) {
	results := []*CheckResult{
		{CheckType: CheckTypeDomain, Name: "idpbolt.com", TLD: "com"},
		{CheckType: CheckTypeNPM, Name: "idpbolt"},
	}
	require.Equal(t, "idpbolt", ExpertDisplayName("ailink", results))
}

func TestExpertDisplayName_FallsBackToBatchNameWhenNoInference(t *testing.T) {
	require.Equal(t, "alpha", ExpertDisplayName("alpha", nil))
}

func TestExpertDigest_JSONShape(t *testing.T) {
	digest := &ExpertDigest{
		Name:      "alpha",
		Status:    "risk: low",
		Notes:     "looks clear",
		RiskLevel: "low",
	}
	encoded, err := json.Marshal(digest)
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"alpha","status":"risk: low","notes":"looks clear","risk_level":"low"}`, string(encoded))
}

func TestExpertDigest_JSONOmitsRiskLevelWhenEmpty(t *testing.T) {
	digest := &ExpertDigest{Name: "alpha", Status: "error", Notes: "boom"}
	encoded, err := json.Marshal(digest)
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"alpha","status":"error","notes":"boom"}`, string(encoded))
}
