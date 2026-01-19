package ailink

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink/driver"
)

func TestMapProviderErrorStatusCodes(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		wantCode   string
	}{
		{"auth", 401, "AILINK_PROVIDER_AUTH"},
		{"forbidden", 403, "AILINK_PROVIDER_AUTH"},
		{"rate", 429, "AILINK_PROVIDER_RATE_LIMIT"},
		{"bad", 400, "AILINK_PROVIDER_BAD_REQUEST"},
		{"unavail", 503, "AILINK_PROVIDER_UNAVAILABLE"},
	}

	for _, tc := range cases {
		err := &driver.ProviderError{Provider: "openai", StatusCode: tc.statusCode, Message: "boom"}
		mapped := mapProviderError(err)
		require.NotNil(t, mapped)
		require.Equal(t, tc.wantCode, mapped.Code)
	}
}
