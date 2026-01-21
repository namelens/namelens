package encode

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBase64RoundTrip(t *testing.T) {
	original := []byte("hello")
	encoded := EncodeBase64String(original)
	decoded, err := DecodeBase64String(encoded)
	require.NoError(t, err)
	require.Equal(t, original, decoded)
}
