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

func TestClientGenerateImageSendsRequestAndDecodesBase64(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/images/generations", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "test-model", payload["model"])
		require.Equal(t, "hello", payload["prompt"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":1,"output_format":"png","data":[{"b64_json":"aGVsbG8="}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	client.HTTPClient = server.Client()

	resp, err := client.GenerateImage(context.Background(), &driver.ImageRequest{Model: "test-model", Prompt: "hello", Count: 1})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Images, 1)
	require.Equal(t, []byte("hello"), resp.Images[0].Data)
}
