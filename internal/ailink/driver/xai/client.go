package xai

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

const defaultBaseURL = "https://api.x.ai/v1"

// Client implements the OpenAI-compatible driver via direct HTTP.
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
	return "xai"
}

// Capabilities describes supported features.
func (c *Client) Capabilities() driver.Capabilities {
	return driver.Capabilities{
		SupportsTools:     true,
		SupportsImages:    true,
		SupportsStreaming: false,
	}
}

// Complete sends a chat completion request.
// Routes to /v1/responses for tool-enabled requests, /v1/chat/completions otherwise.
func (c *Client) Complete(ctx context.Context, req *driver.Request) (*driver.Response, error) {
	if c == nil {
		return nil, fmt.Errorf("xai client not configured")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("api key is required")
	}

	ctx, cancel := withTimeout(ctx, c.Timeout)
	if cancel != nil {
		defer cancel()
	}

	// Route to appropriate endpoint based on whether tools are requested
	if useResponsesAPI(req) {
		return c.completeWithResponses(ctx, req)
	}
	return c.completeWithChat(ctx, req)
}

// completeWithResponses uses the /v1/responses endpoint for tool-enabled requests.
func (c *Client) completeWithResponses(ctx context.Context, req *driver.Request) (*driver.Response, error) {
	payload, err := buildResponsesRequest(req)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/responses"
	start := time.Now()

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
	duration := time.Since(start)
	if err != nil {
		driver.Trace(driver.TraceEntry{
			Driver:      "xai",
			Endpoint:    url,
			Method:      "POST",
			Model:       payload.Model,
			RequestBody: body,
			Error:       err.Error(),
			DurationMs:  duration.Milliseconds(),
		})
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck // best-effort cleanup

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Trace the request/response
	driver.Trace(driver.TraceEntry{
		Driver:      "xai",
		Endpoint:    url,
		Method:      "POST",
		Model:       payload.Model,
		RequestBody: body,
		StatusCode:  resp.StatusCode,
		Response:    respBody,
		DurationMs:  duration.Milliseconds(),
	})

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &driver.ProviderError{Provider: "xai", StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(respBody)), RawResponse: respBody}
	}

	var parsed responsesAPIResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return toDriverResponseFromResponses(&parsed)
}

// completeWithChat uses the legacy /v1/chat/completions endpoint.
func (c *Client) completeWithChat(ctx context.Context, req *driver.Request) (*driver.Response, error) {
	payload, err := buildChatRequest(req)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"
	start := time.Now()

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
	duration := time.Since(start)
	if err != nil {
		driver.Trace(driver.TraceEntry{
			Driver:      "xai",
			Endpoint:    url,
			Method:      "POST",
			Model:       payload.Model,
			RequestBody: body,
			Error:       err.Error(),
			DurationMs:  duration.Milliseconds(),
		})
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck // best-effort cleanup

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Trace the request/response
	driver.Trace(driver.TraceEntry{
		Driver:      "xai",
		Endpoint:    url,
		Method:      "POST",
		Model:       payload.Model,
		RequestBody: body,
		StatusCode:  resp.StatusCode,
		Response:    respBody,
		DurationMs:  duration.Milliseconds(),
	})

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &driver.ProviderError{Provider: "xai", StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(respBody)), RawResponse: respBody}
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
