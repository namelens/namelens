package xai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
	"github.com/namelens/namelens/internal/ailink/encode"
)

type imageGenerationRequest struct {
	Model          string `json:"model,omitempty"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

type imageGenerationResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		B64JSON       string `json:"b64_json,omitempty"`
		URL           string `json:"url,omitempty"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
}

func (c *Client) GenerateImage(ctx context.Context, req *driver.ImageRequest) (*driver.ImageResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("xai client not configured")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("api key is required")
	}
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	count := req.Count
	if count <= 0 {
		count = 1
	}
	if count > 10 {
		return nil, fmt.Errorf("count must be between 1 and 10")
	}

	payload := imageGenerationRequest{
		Model:          strings.TrimSpace(req.Model),
		Prompt:         req.Prompt,
		N:              count,
		ResponseFormat: "b64_json",
	}
	if payload.Model == "" {
		payload.Model = "grok-2-image"
	}

	ctx, cancel := withTimeout(ctx, c.Timeout)
	if cancel != nil {
		defer cancel()
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/images/generations"
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
	defer resp.Body.Close() // nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &driver.ProviderError{Provider: "xai", StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(respBody)), RawResponse: respBody}
	}

	var parsed imageGenerationResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	blocks := make([]content.ContentBlock, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		b64 := strings.TrimSpace(item.B64JSON)
		if b64 == "" {
			continue
		}
		// xAI may return either raw base64 or a data URL.
		if strings.HasPrefix(b64, "data:") {
			if idx := strings.Index(b64, ","); idx > 0 {
				b64 = b64[idx+1:]
			}
		}
		decoded, err := encode.DecodeBase64String(b64)
		if err != nil {
			return nil, fmt.Errorf("decode image base64: %w", err)
		}
		blocks = append(blocks, content.ContentBlock{Type: "image/jpeg", Data: decoded})
	}

	return &driver.ImageResponse{Created: parsed.Created, Images: blocks}, nil
}
