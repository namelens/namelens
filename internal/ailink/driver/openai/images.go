package openai

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
	Model        string `json:"model,omitempty"`
	Prompt       string `json:"prompt"`
	N            int    `json:"n,omitempty"`
	Size         string `json:"size,omitempty"`
	Quality      string `json:"quality,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
	Background   string `json:"background,omitempty"`
	// response_format is used for DALL·E models; GPT image models always return base64.
	ResponseFormat string `json:"response_format,omitempty"`
}

type imageGenerationResponse struct {
	Created      int64  `json:"created"`
	OutputFormat string `json:"output_format,omitempty"`
	Size         string `json:"size,omitempty"`
	Quality      string `json:"quality,omitempty"`
	Data         []struct {
		B64JSON string `json:"b64_json,omitempty"`
		URL     string `json:"url,omitempty"`
	} `json:"data"`
}

func (c *Client) GenerateImage(ctx context.Context, req *driver.ImageRequest) (*driver.ImageResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("openai client not configured")
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
		Model:   strings.TrimSpace(req.Model),
		Prompt:  req.Prompt,
		N:       count,
		Size:    strings.TrimSpace(req.Size),
		Quality: strings.TrimSpace(req.Quality),
	}
	if payload.Model == "" {
		payload.Model = "dall-e-2"
	}

	// DALL·E models require response_format and do not support output_format/background.
	// DALL·E 3 expects quality standard|hd; default "auto" coerces to standard.
	// GPT image models accept output_format/background and always return base64.
	if strings.HasPrefix(payload.Model, "dall-e") {
		payload.ResponseFormat = "b64_json"
		q := strings.ToLower(strings.TrimSpace(payload.Quality))
		if q == "" || q == "auto" {
			payload.Quality = "standard"
		}
	} else {
		payload.OutputFormat = strings.TrimSpace(req.OutputFormat)
		payload.Background = strings.TrimSpace(req.Background)
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
	defer resp.Body.Close() // nolint:errcheck // best-effort cleanup

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &driver.ProviderError{Provider: "openai", StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(respBody)), RawResponse: respBody}
	}

	var parsed imageGenerationResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	blocks := make([]content.ContentBlock, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		if strings.TrimSpace(item.B64JSON) != "" {
			decoded, err := encode.DecodeBase64String(item.B64JSON)
			if err != nil {
				return nil, fmt.Errorf("decode image base64: %w", err)
			}
			mime := "image/png"
			if parsed.OutputFormat != "" {
				mime = "image/" + parsed.OutputFormat
			} else if req.OutputFormat != "" {
				mime = "image/" + strings.ToLower(strings.TrimSpace(req.OutputFormat))
			}
			blocks = append(blocks, content.ContentBlock{Type: content.ContentType(mime), Data: decoded})
			continue
		}
		if strings.TrimSpace(item.URL) != "" {
			blocks = append(blocks, content.ContentBlock{Type: "text/plain", Text: item.URL})
		}
	}

	return &driver.ImageResponse{
		Created:      parsed.Created,
		OutputFormat: parsed.OutputFormat,
		Size:         parsed.Size,
		Quality:      parsed.Quality,
		Images:       blocks,
	}, nil
}
