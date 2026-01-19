package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/namelens/namelens/internal/ailink/driver"
)

const defaultBaseURL = "https://api.openai.com/v1"

// Client implements the OpenAI driver via direct HTTP.
//
// Note: This is distinct from the xAI driver, which speaks an OpenAI-compatible
// API shape but targets x.ai.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// NewClient returns a client with defaults applied.
func NewClient(baseURL, apiKey string) *Client {
	url := strings.TrimSpace(baseURL)
	if url == "" {
		url = defaultBaseURL
	}

	return &Client{
		BaseURL: url,
		APIKey:  strings.TrimSpace(apiKey),
	}
}

// Name returns the driver identifier.
func (c *Client) Name() string {
	return "openai"
}

// Capabilities describes supported features.
func (c *Client) Capabilities() driver.Capabilities {
	return driver.Capabilities{
		SupportsTools:     true,
		SupportsImages:    false,
		SupportsStreaming: false,
	}
}

// Complete sends a chat completion request.
func (c *Client) Complete(ctx context.Context, req *driver.Request) (*driver.Response, error) {
	if c == nil {
		return nil, fmt.Errorf("openai client not configured")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("api key is required")
	}

	payload, err := buildChatRequest(req)
	if err != nil {
		return nil, err
	}

	ctx, cancel := withTimeout(ctx, c.Timeout)
	if cancel != nil {
		defer cancel()
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck // best-effort cleanup

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &driver.ProviderError{Provider: "openai", StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(respBody)), RawResponse: respBody}
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return toDriverResponse(&parsed)
}

func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, nil
	}
	return context.WithTimeout(ctx, timeout)
}
