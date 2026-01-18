package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
)

func TestClientRequiresAPIKey(t *testing.T) {
	client := NewClient("", "")
	_, err := client.Complete(context.Background(), &driver.Request{Model: "test", Messages: []content.Message{{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "hi"}}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "api key")
}

func TestClientRejectsSearchParameters(t *testing.T) {
	client := NewClient("", "test-key")
	_, err := client.Complete(context.Background(), &driver.Request{
		Model:    "test",
		Messages: []content.Message{{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "hi"}}}},
		SearchParameters: &driver.SearchParameters{
			Mode: "auto",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "search_parameters")
}

func TestClientSendsRequestAndParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		_, hasSearchParams := payload["search_parameters"]
		require.False(t, hasSearchParams)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"summary\":\"ok\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	client.HTTPClient = server.Client()

	resp, err := client.Complete(context.Background(), &driver.Request{
		Model: "test-model",
		Messages: []content.Message{
			{Role: "system", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "sys"}}},
			{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "usr"}}},
		},
		ResponseFormat: &driver.ResponseFormat{Type: "json_object"},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "stop", resp.FinishReason)
	require.NotNil(t, resp.Usage)
	require.Equal(t, 3, resp.Usage.TotalTokens)
	require.Len(t, resp.Content, 1)
	require.True(t, strings.Contains(resp.Content[0].Text, "summary"))
}

func TestClientErrorsOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("nope"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	client.HTTPClient = server.Client()

	_, err := client.Complete(context.Background(), &driver.Request{Model: "test", Messages: []content.Message{{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "hi"}}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 401")
	require.Contains(t, err.Error(), "nope")
}
