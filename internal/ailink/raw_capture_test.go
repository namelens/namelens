package ailink

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTruncateJSONRaw(t *testing.T) {
	input := json.RawMessage(`{"a":"0123456789"}`)
	out := truncateJSONRaw(input, 8)
	require.Len(t, out, 8)
	require.Equal(t, string(input[:8]), string(out))

	require.Equal(t, input, truncateJSONRaw(input, 1024))
	require.Nil(t, truncateJSONRaw(input, 0))
}
