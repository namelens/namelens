package xai

import (
	"fmt"
	"strings"

	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
)

// chatCompletionRequest is for the legacy /v1/chat/completions endpoint (no tools).
type chatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      *int            `json:"max_tokens,omitempty"`
}

// responsesAPIRequest is for the new /v1/responses endpoint (with tools).
type responsesAPIRequest struct {
	Model string          `json:"model"`
	Input []inputMessage  `json:"input"`
	Tools []responsesTool `json:"tools,omitempty"`
}

type inputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responsesTool struct {
	Type string `json:"type"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// useResponsesAPI returns true if the request should use the new /v1/responses endpoint.
func useResponsesAPI(req *driver.Request) bool {
	return req != nil && req.SearchParameters != nil && len(req.SearchParameters.Sources) > 0
}

// buildResponsesRequest builds a request for the /v1/responses endpoint.
func buildResponsesRequest(req *driver.Request) (*responsesAPIRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages are required")
	}

	// Convert messages to input format
	input := make([]inputMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		text := extractTextContent(msg.Content)
		input = append(input, inputMessage{Role: msg.Role, Content: text})
	}

	// Convert search sources to tools
	var tools []responsesTool
	if req.SearchParameters != nil {
		for _, src := range req.SearchParameters.Sources {
			toolType := src.Type
			// Map source type to tool type
			switch toolType {
			case "web":
				toolType = "web_search"
			case "x":
				toolType = "x_search"
			}
			tools = append(tools, responsesTool{Type: toolType})
		}
	}

	payload := &responsesAPIRequest{
		Model: req.Model,
		Input: input,
		Tools: tools,
	}

	return payload, nil
}

// buildChatRequest builds a request for the legacy /v1/chat/completions endpoint.
func buildChatRequest(req *driver.Request) (*chatCompletionRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	messages, err := convertMessages(req.Messages)
	if err != nil {
		return nil, err
	}

	payload := &chatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	if req.ResponseFormat != nil {
		payload.ResponseFormat = &responseFormat{Type: req.ResponseFormat.Type}
	}
	return payload, nil
}

// extractTextContent extracts plain text from content blocks.
func extractTextContent(blocks []content.ContentBlock) string {
	var parts []string
	for _, block := range blocks {
		if block.Type == content.ContentTypeText && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
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

func convertContent(blocks []content.ContentBlock) (any, error) {
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
