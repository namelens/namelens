package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink/driver"
)

func TestClientGenerateImageDALLEOmitsOutputFormatAndBackground(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))

		require.Equal(t, "dall-e-3", payload["model"])
		require.Equal(t, "b64_json", payload["response_format"])
		require.Equal(t, "standard", payload["quality"])
		_, hasOutput := payload["output_format"]
		require.False(t, hasOutput)
		_, hasBackground := payload["background"]
		require.False(t, hasBackground)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":1,"data":[{"b64_json":"aGVsbG8="}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	client.HTTPClient = server.Client()

	resp, err := client.GenerateImage(context.Background(), &driver.ImageRequest{Model: "dall-e-3", Prompt: "hello", Count: 1, OutputFormat: "webp", Background: "transparent", Quality: "auto"})
	require.NoError(t, err)
	require.NotNil(t, resp)
}
