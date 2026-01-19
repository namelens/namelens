package openai

import (
	"fmt"
	"strings"

	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
)

type chatCompletionRequest struct {
	Model          string           `json:"model"`
	Messages       []chatMessage    `json:"messages"`
	Tools          []map[string]any `json:"tools,omitempty"`
	ResponseFormat *responseFormat  `json:"response_format,omitempty"`
	Temperature    *float64         `json:"temperature,omitempty"`
	MaxTokens      *int             `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type responseFormat struct {
	Type       string            `json:"type"`
	JSONSchema *responseJSONSpec `json:"json_schema,omitempty"`
}

type responseJSONSpec struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func buildChatRequest(req *driver.Request) (*chatCompletionRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	if req.SearchParameters != nil {
		// SearchParameters are an xAI-specific extension. They should not be sent to OpenAI.
		return nil, fmt.Errorf("search_parameters are not supported by openai driver")
	}

	messages, err := convertMessages(req.Messages)
	if err != nil {
		return nil, err
	}

	payload := &chatCompletionRequest{
		Model:          req.Model,
		Messages:       messages,
		Tools:          flattenTools(req.Tools),
		Temperature:    req.Temperature,
		MaxTokens:      req.MaxTokens,
		ResponseFormat: nil,
	}
	if req.ResponseFormat != nil {
		payload.ResponseFormat = &responseFormat{Type: req.ResponseFormat.Type}
		if req.ResponseFormat.JSONSchema != nil {
			payload.ResponseFormat.JSONSchema = &responseJSONSpec{
				Name:   req.ResponseFormat.JSONSchema.Name,
				Strict: req.ResponseFormat.JSONSchema.Strict,
				Schema: req.ResponseFormat.JSONSchema.Schema,
			}
		}
	}

	return payload, nil
}

func convertMessages(messages []content.Message) ([]chatMessage, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages are required")
	}
	result := make([]chatMessage, 0, len(messages))
	for _, msg := range messages {
		contentValue, err := convertContent(msg.Content)
		if err != nil {
			return nil, err
		}
		result = append(result, chatMessage{Role: msg.Role, Content: contentValue})
	}
	return result, nil
}

func flattenTools(tools []driver.Tool) []map[string]any {
	if len(tools) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		flat := map[string]any{"type": t.Type}
		for k, v := range t.Config {
			flat[k] = v
		}
		result = append(result, flat)
	}
	return result
}

func convertContent(blocks []content.ContentBlock) (interface{}, error) {
	if len(blocks) == 0 {
		return "", nil
	}
	if len(blocks) == 1 && blocks[0].Type == content.ContentTypeText {
		return blocks[0].Text, nil
	}

	converted := make([]contentBlock, 0, len(blocks))
	for _, block := range blocks {
		if block.Type != content.ContentTypeText {
			return nil, fmt.Errorf("unsupported content type: %s", block.Type)
		}
		converted = append(converted, contentBlock{Type: "text", Text: block.Text})
	}
	return converted, nil
}
