package anthropic

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
	_, err := client.Complete(context.Background(), &driver.Request{
		Model:    "claude-3-haiku-20240307",
		Messages: []content.Message{{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "hi"}}}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "api key")
}

func TestClientSendsCorrectHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Anthropic-specific headers
		require.Equal(t, "/messages", r.URL.Path)
		require.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
		require.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Should NOT have Authorization header (that's OpenAI style)
		require.Empty(t, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "Hello!"}],
			"model": "claude-3-haiku-20240307",
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 5}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	client.HTTPClient = server.Client()

	resp, err := client.Complete(context.Background(), &driver.Request{
		Model: "claude-3-haiku-20240307",
		Messages: []content.Message{
			{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "hi"}}},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestClientSendsRequestAndParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		// Verify request structure
		require.Equal(t, "claude-3-haiku-20240307", payload["model"])
		require.NotNil(t, payload["max_tokens"])
		require.Equal(t, "You are helpful.", payload["system"])

		messages, ok := payload["messages"].([]any)
		require.True(t, ok)
		require.Len(t, messages, 1)

		msg := messages[0].(map[string]any)
		require.Equal(t, "user", msg["role"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "I can help with that!"}],
			"model": "claude-3-haiku-20240307",
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 15, "output_tokens": 10}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	client.HTTPClient = server.Client()

	resp, err := client.Complete(context.Background(), &driver.Request{
		Model: "claude-3-haiku-20240307",
		Messages: []content.Message{
			{Role: "system", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "You are helpful."}}},
			{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "Help me!"}}},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "stop", resp.FinishReason)
	require.NotNil(t, resp.Usage)
	require.Equal(t, 15, resp.Usage.PromptTokens)
	require.Equal(t, 10, resp.Usage.CompletionTokens)
	require.Equal(t, 25, resp.Usage.TotalTokens)
	require.Len(t, resp.Content, 1)
	require.Contains(t, resp.Content[0].Text, "help with that")
}

func TestClientHandlesMultipleContentBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "msg_456",
			"type": "message",
			"role": "assistant",
			"content": [
				{"type": "text", "text": "First part."},
				{"type": "text", "text": "Second part."}
			],
			"model": "claude-3-haiku-20240307",
			"stop_reason": "end_turn"
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	client.HTTPClient = server.Client()

	resp, err := client.Complete(context.Background(), &driver.Request{
		Model: "claude-3-haiku-20240307",
		Messages: []content.Message{
			{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "hi"}}},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Content, 1)
	require.Contains(t, resp.Content[0].Text, "First part.")
	require.Contains(t, resp.Content[0].Text, "Second part.")
}

func TestClientErrorsOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-key")
	client.HTTPClient = server.Client()

	_, err := client.Complete(context.Background(), &driver.Request{
		Model: "claude-3-haiku-20240307",
		Messages: []content.Message{
			{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "hi"}}},
		},
	})
	require.Error(t, err)

	var perr *driver.ProviderError
	require.ErrorAs(t, err, &perr)
	require.Equal(t, 401, perr.StatusCode)
	require.Contains(t, perr.Message, "authentication_error")
}

func TestClientHandlesMaxTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		// Verify custom max_tokens is sent
		maxTokens, ok := payload["max_tokens"].(float64)
		require.True(t, ok)
		require.Equal(t, float64(1000), maxTokens)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "msg_789",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "ok"}],
			"model": "claude-3-haiku-20240307",
			"stop_reason": "max_tokens"
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	client.HTTPClient = server.Client()

	maxTokens := 1000
	resp, err := client.Complete(context.Background(), &driver.Request{
		Model:     "claude-3-haiku-20240307",
		MaxTokens: &maxTokens,
		Messages: []content.Message{
			{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "hi"}}},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "length", resp.FinishReason)
}

func TestClientName(t *testing.T) {
	client := NewClient("", "key")
	require.Equal(t, "anthropic", client.Name())
}

func TestClientCapabilities(t *testing.T) {
	client := NewClient("", "key")
	caps := client.Capabilities()
	require.False(t, caps.SupportsTools)
	require.False(t, caps.SupportsImages) // Claude doesn't generate images
	require.False(t, caps.SupportsStreaming)
}

func TestBuildMessagesRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     *driver.Request
		wantErr string
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: "request is required",
		},
		{
			name:    "empty model",
			req:     &driver.Request{Model: "", Messages: []content.Message{{Role: "user"}}},
			wantErr: "model is required",
		},
		{
			name:    "no messages",
			req:     &driver.Request{Model: "claude-3-haiku-20240307", Messages: nil},
			wantErr: "messages are required",
		},
		{
			name: "only system message",
			req: &driver.Request{
				Model:    "claude-3-haiku-20240307",
				Messages: []content.Message{{Role: "system", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "sys"}}}},
			},
			wantErr: "at least one non-system message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildMessagesRequest(tt.req)
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), tt.wantErr), "error %q should contain %q", err.Error(), tt.wantErr)
		})
	}
}

func TestStripMarkdownFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no fences",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "json fence",
			input: "```json\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "plain fence",
			input: "```\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "fence with whitespace",
			input: "  ```json\n{\"key\": \"value\"}\n```  ",
			want:  `{"key": "value"}`,
		},
		{
			name:  "nested content preserved",
			input: "```json\n{\"code\": \"```nested```\"}\n```",
			want:  `{"code": "` + "```nested```" + `"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMarkdownFences(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestClientStripsMarkdownFencesFromResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Simulate Anthropic wrapping JSON in markdown fences
		_, _ = w.Write([]byte(`{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "` + "```json\\n{\\\"name\\\": \\\"test\\\"}\\n```" + `"}],
			"model": "claude-sonnet-4-5-20250929",
			"stop_reason": "end_turn"
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	client.HTTPClient = server.Client()

	resp, err := client.Complete(context.Background(), &driver.Request{
		Model: "claude-sonnet-4-5-20250929",
		Messages: []content.Message{
			{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "hi"}}},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Content, 1)
	// Should have stripped the markdown fences
	require.Equal(t, `{"name": "test"}`, resp.Content[0].Text)
}
