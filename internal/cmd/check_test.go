package cmd

import (
	"testing"

	"github.com/namelens/namelens/internal/core"
)

func TestValidateName(t *testing.T) {
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"goodname", false},
		{"good-name", false},
		{"-bad", true},
		{"bad-", true},
		{"bad name", true},
		{"BAD", true},
		{"", true},
	}

	for _, tc := range cases {
		err := validateName(tc.name)
		if tc.wantErr && err == nil {
			t.Fatalf("expected error for %q", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.name, err)
		}
	}
}

func TestNormalizeTLDs(t *testing.T) {
	input := []string{".com", " io ", "com", "dev,app"}
	result := normalizeTLDs(input)
	if len(result) != 4 {
		t.Fatalf("expected 4 tlds, got %d", len(result))
	}
}

func TestSummarizeResultsPrefersInferredNameWhenInputMismatchesChecks(t *testing.T) {
	results := []*core.CheckResult{
		{
			CheckType: core.CheckTypeDomain,
			Name:      "idpbolt.com",
			TLD:       "com",
			Available: core.AvailabilityTaken,
		},
		{
			CheckType: core.CheckTypeNPM,
			Name:      "idpbolt",
			Available: core.AvailabilityAvailable,
		},
	}

	batch := summarizeResults("ailink", results, nil, nil, nil, nil, nil, nil)
	if batch.Name != "idpbolt" {
		t.Fatalf("expected inferred batch name idpbolt, got %q", batch.Name)
	}
}

func TestSummarizeResultsKeepsInputNameWhenChecksMatch(t *testing.T) {
	results := []*core.CheckResult{
		{
			CheckType: core.CheckTypeDomain,
			Name:      "ailink.com",
			TLD:       "com",
			Available: core.AvailabilityTaken,
		},
	}

	batch := summarizeResults("ailink", results, nil, nil, nil, nil, nil, nil)
	if batch.Name != "ailink" {
		t.Fatalf("expected batch name ailink, got %q", batch.Name)
	}
}
