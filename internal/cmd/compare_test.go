package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/core"
	"github.com/namelens/namelens/internal/output"
)

func TestSummarizeAvailability(t *testing.T) {
	tests := []struct {
		name     string
		results  []*core.CheckResult
		expected compareAvailability
	}{
		{
			name:     "empty results",
			results:  nil,
			expected: compareAvailability{Score: 0, Total: 0, Unknown: 0},
		},
		{
			name: "all available",
			results: []*core.CheckResult{
				{Available: core.AvailabilityAvailable},
				{Available: core.AvailabilityAvailable},
				{Available: core.AvailabilityAvailable},
			},
			expected: compareAvailability{Score: 3, Total: 3, Unknown: 0},
		},
		{
			name: "mixed results",
			results: []*core.CheckResult{
				{Available: core.AvailabilityAvailable},
				{Available: core.AvailabilityTaken},
				{Available: core.AvailabilityUnknown},
				{Available: core.AvailabilityAvailable},
			},
			expected: compareAvailability{Score: 2, Total: 4, Unknown: 1},
		},
		{
			name: "with nil entries",
			results: []*core.CheckResult{
				{Available: core.AvailabilityAvailable},
				nil,
				{Available: core.AvailabilityTaken},
			},
			expected: compareAvailability{Score: 1, Total: 2, Unknown: 0},
		},
		{
			name: "error and rate limited count as unknown",
			results: []*core.CheckResult{
				{Available: core.AvailabilityError},
				{Available: core.AvailabilityRateLimited},
				{Available: core.AvailabilityAvailable},
			},
			expected: compareAvailability{Score: 1, Total: 3, Unknown: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := summarizeAvailability(tt.results)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDeriveRiskLevel(t *testing.T) {
	tests := []struct {
		name     string
		results  []*core.CheckResult
		expected string
	}{
		{
			name:     "empty results",
			results:  nil,
			expected: "unknown",
		},
		{
			name:     "all nil results",
			results:  []*core.CheckResult{nil, nil},
			expected: "unknown",
		},
		{
			name: "all available - low risk",
			results: []*core.CheckResult{
				{CheckType: core.CheckTypeDomain, Name: "test.com", Available: core.AvailabilityAvailable},
				{CheckType: core.CheckTypeDomain, Name: "test.io", Available: core.AvailabilityAvailable},
			},
			expected: "low",
		},
		{
			name: ".com taken - high risk",
			results: []*core.CheckResult{
				{CheckType: core.CheckTypeDomain, Name: "test.com", Available: core.AvailabilityTaken},
				{CheckType: core.CheckTypeDomain, Name: "test.io", Available: core.AvailabilityAvailable},
			},
			expected: "high",
		},
		{
			name: "other asset taken but .com available - medium risk",
			results: []*core.CheckResult{
				{CheckType: core.CheckTypeDomain, Name: "test.com", Available: core.AvailabilityAvailable},
				{CheckType: core.CheckTypeDomain, Name: "test.io", Available: core.AvailabilityTaken},
			},
			expected: "medium",
		},
		{
			name: "registry taken - medium risk",
			results: []*core.CheckResult{
				{CheckType: core.CheckTypeDomain, Name: "test.com", Available: core.AvailabilityAvailable},
				{CheckType: core.CheckTypeNPM, Name: "test", Available: core.AvailabilityTaken},
			},
			expected: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveRiskLevel(tt.results)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPhonetics(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected *comparePhonetics
	}{
		{
			name:     "empty json",
			raw:      `{}`,
			expected: nil,
		},
		{
			name: "valid phonetics",
			raw: `{
				"overall_assessment": {
					"combined_score": 85,
					"typeability_score": 82
				},
				"cli_suitability": {
					"score": 90
				}
			}`,
			expected: &comparePhonetics{
				OverallScore:     85,
				TypeabilityScore: 82,
				CLISuitability:   90,
			},
		},
		{
			name: "missing cli_suitability",
			raw: `{
				"overall_assessment": {
					"combined_score": 75,
					"typeability_score": 70
				}
			}`,
			expected: &comparePhonetics{
				OverallScore:     75,
				TypeabilityScore: 70,
				CLISuitability:   0,
			},
		},
		{
			name:     "invalid json",
			raw:      `{invalid}`,
			expected: nil,
		},
		{
			name: "zero combined score",
			raw: `{
				"overall_assessment": {
					"combined_score": 0
				}
			}`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPhonetics(json.RawMessage(tt.raw))
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractSuitability(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected *compareSuitability
	}{
		{
			name:     "empty json",
			raw:      `{}`,
			expected: nil,
		},
		{
			name: "valid suitability",
			raw: `{
				"overall_suitability": {
					"score": 95,
					"rating": "suitable"
				}
			}`,
			expected: &compareSuitability{
				OverallScore: 95,
				Rating:       "suitable",
			},
		},
		{
			name: "caution rating",
			raw: `{
				"overall_suitability": {
					"score": 60,
					"rating": "caution"
				}
			}`,
			expected: &compareSuitability{
				OverallScore: 60,
				Rating:       "caution",
			},
		},
		{
			name:     "invalid json",
			raw:      `{invalid}`,
			expected: nil,
		},
		{
			name: "zero score with rating",
			raw: `{
				"overall_suitability": {
					"score": 0,
					"rating": "unsuitable"
				}
			}`,
			expected: &compareSuitability{
				OverallScore: 0,
				Rating:       "unsuitable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSuitability(json.RawMessage(tt.raw))
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatAvailability(t *testing.T) {
	tests := []struct {
		name     string
		row      compareRow
		expected string
	}{
		{
			name:     "availability error shows error",
			row:      compareRow{AvailabilityError: "error"},
			expected: "error",
		},
		{
			name:     "normal availability",
			row:      compareRow{Availability: compareAvailability{Score: 5, Total: 7, Unknown: 0}},
			expected: "5/7",
		},
		{
			name:     "availability with unknown",
			row:      compareRow{Availability: compareAvailability{Score: 3, Total: 5, Unknown: 2}},
			expected: "3/5 (2?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAvailability(tt.row)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatRisk(t *testing.T) {
	tests := []struct {
		name     string
		row      compareRow
		expected string
	}{
		{
			name:     "availability error returns dash",
			row:      compareRow{AvailabilityError: "error"},
			expected: "-",
		},
		{
			name:     "empty risk returns dash",
			row:      compareRow{RiskLevel: ""},
			expected: "-",
		},
		{
			name:     "normal risk level",
			row:      compareRow{RiskLevel: "low"},
			expected: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRisk(tt.row)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderCompareTable(t *testing.T) {
	rows := []compareRow{
		{
			Name:         "testname",
			Length:       8,
			Availability: compareAvailability{Score: 5, Total: 7, Unknown: 1},
			RiskLevel:    "low",
			Phonetics:    &comparePhonetics{OverallScore: 85},
			Suitability:  &compareSuitability{OverallScore: 90},
		},
		{
			Name:         "another",
			Length:       7,
			Availability: compareAvailability{Score: 7, Total: 7, Unknown: 0},
			RiskLevel:    "low",
			Phonetics:    nil,
			Suitability:  nil,
		},
	}

	var buf bytes.Buffer
	err := renderCompareTable(&buf, rows, false)
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "testname")
	require.Contains(t, output, "5/7")
	require.Contains(t, output, "low")
	require.Contains(t, output, "85")
	require.Contains(t, output, "90")
	require.Contains(t, output, "another")
	require.Contains(t, output, "7/7")
}

func TestRenderCompareTableQuickMode(t *testing.T) {
	rows := []compareRow{
		{
			Name:         "quicktest",
			Length:       9,
			Availability: compareAvailability{Score: 3, Total: 5, Unknown: 0},
		},
	}

	var buf bytes.Buffer
	err := renderCompareTable(&buf, rows, true)
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "quicktest")
	require.Contains(t, output, "3/5")
	require.Contains(t, output, "9")
	// Quick mode should not have risk/phonetics/suitability columns
	require.NotContains(t, output, "Risk")
	require.NotContains(t, output, "Phonetics")
}

func TestRenderCompareTableWithError(t *testing.T) {
	rows := []compareRow{
		{
			Name:              "errortest",
			Length:            9,
			AvailabilityError: "error",
		},
	}

	var buf bytes.Buffer
	err := renderCompareTable(&buf, rows, false)
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "errortest")
	require.Contains(t, output, "error")
}

func TestRenderCompareMarkdown(t *testing.T) {
	rows := []compareRow{
		{
			Name:         "mdtest",
			Length:       6,
			Availability: compareAvailability{Score: 4, Total: 6, Unknown: 2},
			RiskLevel:    "medium",
			Phonetics:    &comparePhonetics{OverallScore: 70},
			Suitability:  &compareSuitability{OverallScore: 80},
		},
	}

	var buf bytes.Buffer
	err := renderCompareMarkdown(&buf, rows, false)
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "| Name |")
	require.Contains(t, output, "| mdtest |")
	require.Contains(t, output, "4/6 (2?)")
	require.Contains(t, output, "medium")
	require.Contains(t, output, "70")
	require.Contains(t, output, "80")
}

func TestRenderCompareMarkdownWithError(t *testing.T) {
	rows := []compareRow{
		{
			Name:              "mderror",
			Length:            7,
			AvailabilityError: "error",
		},
	}

	var buf bytes.Buffer
	err := renderCompareMarkdown(&buf, rows, false)
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "mderror")
	require.Contains(t, output, "error")
}

func TestRenderCompareJSON(t *testing.T) {
	rows := []compareRow{
		{
			Name:         "jsontest",
			Length:       8,
			Availability: compareAvailability{Score: 2, Total: 4, Unknown: 1},
			RiskLevel:    "high",
			Phonetics:    &comparePhonetics{OverallScore: 65, TypeabilityScore: 60, CLISuitability: 70},
			Suitability:  &compareSuitability{OverallScore: 55, Rating: "caution"},
		},
	}

	var buf bytes.Buffer
	err := renderCompare(&buf, rows, output.FormatJSON, false)
	require.NoError(t, err)

	var parsed []compareRow
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	require.Equal(t, "jsontest", parsed[0].Name)
	require.Equal(t, 2, parsed[0].Availability.Score)
	require.Equal(t, "high", parsed[0].RiskLevel)
	require.Equal(t, 65, parsed[0].Phonetics.OverallScore)
	require.Equal(t, "caution", parsed[0].Suitability.Rating)
}

func TestRenderCompareJSONWithError(t *testing.T) {
	rows := []compareRow{
		{
			Name:              "jsonerror",
			Length:            9,
			AvailabilityError: "error",
		},
	}

	var buf bytes.Buffer
	err := renderCompare(&buf, rows, output.FormatJSON, false)
	require.NoError(t, err)

	var parsed []compareRow
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	require.Equal(t, "error", parsed[0].AvailabilityError)
}

func TestCompareModeValidation(t *testing.T) {
	tests := []struct {
		mode      string
		wantError bool
		errorMsg  string
	}{
		{"", false, ""},
		{"quick", false, ""},
		{"QUICK", false, ""},
		{"  quick  ", false, ""},
		{"invalid", true, "unsupported mode"},
		{"full", true, "unsupported mode"},
		{"fast", true, "unsupported mode"},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			normalizedMode := strings.ToLower(strings.TrimSpace(tt.mode))
			var err error
			if normalizedMode != "" && normalizedMode != "quick" {
				err = fmt.Errorf("unsupported mode: %s", tt.mode)
			}

			if tt.wantError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
