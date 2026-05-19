package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink"
)

func TestExpertBulkKey_LowersAndTrims(t *testing.T) {
	cases := []struct{ in, want string }{
		{"ACME", "acme"},
		{"  Acme  ", "acme"},
		{"\tBeta\n", "beta"},
		{"Gamma", "gamma"},
		{"already-lower", "already-lower"},
		{"", ""},
	}
	for _, c := range cases {
		require.Equal(t, c.want, expertBulkKey(c.in), "expertBulkKey(%q)", c.in)
	}
}

func TestExpertBulkKey_LookupRoundTripSurvivesCaseAndWhitespace(t *testing.T) {
	// Pre-v0.2.5 PR-3, three different normalizations across lookup / live /
	// cache produced silent misses on any name whose raw input differed from
	// the bulk response's item.Name in case or surrounding whitespace. With
	// expertBulkKey applied at every site, any pair of strings that compares
	// equal post-normalization must round-trip through the merge.
	pairs := []struct{ requested, returned string }{
		{"Acme", "acme"},
		{"  Acme  ", "ACME"},
		{"beta", "Beta"},
		{"GAMMA\t", "gamma\n"},
	}
	for _, p := range pairs {
		require.Equal(t, expertBulkKey(p.requested), expertBulkKey(p.returned),
			"requested %q vs returned %q should hash to the same key", p.requested, p.returned)
	}
}

func TestBulkItemsToMap_KeysCanonicalizedAndPayloadPreserved(t *testing.T) {
	yes := true
	conf := 0.85
	items := []ailink.BulkSearchItem{
		{Name: "ACME", Summary: "first", RiskLevel: "low", LikelyAvailable: &yes, Confidence: &conf},
		{Name: "  Beta  ", Summary: "second", RiskLevel: "medium",
			Insights: []string{"i1"}, Recommendations: []string{"r1"}},
		{Name: "gamma", Summary: "third", RiskLevel: "high"},
	}

	got := bulkItemsToMap(items)
	require.Len(t, got, 3)

	// Keys are canonicalized.
	require.Contains(t, got, "acme")
	require.Contains(t, got, "beta")
	require.Contains(t, got, "gamma")
	require.NotContains(t, got, "ACME")
	require.NotContains(t, got, "  Beta  ")

	// Payload preserved end-to-end.
	require.Equal(t, "first", got["acme"].Summary)
	require.Equal(t, "low", got["acme"].RiskLevel)
	require.NotNil(t, got["acme"].LikelyAvailable)
	require.True(t, *got["acme"].LikelyAvailable)
	require.NotNil(t, got["acme"].Confidence)
	require.InDelta(t, 0.85, *got["acme"].Confidence, 0.0001)

	require.Equal(t, []string{"i1"}, got["beta"].Insights)
	require.Equal(t, []string{"r1"}, got["beta"].Recommendations)
}

func TestBulkItemsToMap_EmptyInput(t *testing.T) {
	got := bulkItemsToMap(nil)
	require.NotNil(t, got)
	require.Len(t, got, 0)
}

func TestWaitForFallbackSlot_ReturnsNilImmediatelyWhenTargetInPast(t *testing.T) {
	ctx := context.Background()
	target := time.Now().Add(-1 * time.Second)
	start := time.Now()
	err := waitForFallbackSlot(ctx, target)
	elapsed := time.Since(start)
	require.Nil(t, err)
	require.Less(t, elapsed, 50*time.Millisecond, "past target should return immediately")
}

func TestWaitForFallbackSlot_WaitsUntilTargetWhenContextHealthy(t *testing.T) {
	ctx := context.Background()
	target := time.Now().Add(80 * time.Millisecond)
	start := time.Now()
	err := waitForFallbackSlot(ctx, target)
	elapsed := time.Since(start)
	require.Nil(t, err)
	require.GreaterOrEqual(t, elapsed, 70*time.Millisecond, "should wait ~80ms")
	require.Less(t, elapsed, 200*time.Millisecond, "wait should not overshoot meaningfully")
}

func TestWaitForFallbackSlot_ReturnsCanceledErrorWhenContextExpiresDuringWait(t *testing.T) {
	// Pre-PR-3 the worker code did `continue` here, dropping the batch slot
	// for any name whose fallback was scheduled but canceled before retry.
	// Post-PR-3 the cancel surfaces as an AILINK_FALLBACK_CANCELED error
	// that the caller writes into the batch slot via summarizeResults, so
	// every requested name gets an entry in the output.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	target := time.Now().Add(5 * time.Second)
	start := time.Now()
	err := waitForFallbackSlot(ctx, target)
	elapsed := time.Since(start)

	require.NotNil(t, err, "ctx-cancel during wait must produce an error, not nil")
	require.Equal(t, "AILINK_FALLBACK_CANCELED", err.Code)
	require.NotEmpty(t, err.Message)
	require.Less(t, elapsed, 500*time.Millisecond, "should return promptly on cancel, not wait the full 5s")
}
