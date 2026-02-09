package anthropic

import (
	"fmt"
	"strings"

	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
)

// messagesRequest is the request body for the /v1/messages endpoint.
type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
	System    string    `json:"system,omitempty"`
}

// message represents a conversation message in Anthropic format.
type message struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

// contentBlock represents a content block in Anthropic format.
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// buildMessagesRequest converts a driver.Request to an Anthropic messagesRequest.
func buildMessagesRequest(req *driver.Request) (*messagesRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages are required")
	}

	// Extract system message and convert other messages
	var systemText string
	messages := make([]message, 0, len(req.Messages))

	for _, msg := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))

		// Anthropic uses a top-level system field, not a system message
		if role == "system" {
			chunk := strings.TrimSpace(extractTextContent(msg.Content))
			if chunk != "" {
				if systemText != "" {
					systemText += "\n" + chunk
				} else {
					systemText = chunk
				}
			}
			continue
		}

		// Convert role names (Anthropic uses "user" and "assistant")
		if role != "user" && role != "assistant" {
			// Map any other roles to user (e.g., "human" -> "user")
			role = "user"
		}

		converted, err := convertContent(msg.Content)
		if err != nil {
			return nil, err
		}

		messages = append(messages, message{
			Role:    role,
			Content: converted,
		})
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("at least one non-system message is required")
	}

	// Determine max tokens
	maxTokens := defaultMaxTokens
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		maxTokens = *req.MaxTokens
	}

	payload := &messagesRequest{
		Model:     req.Model,
		MaxTokens: maxTokens,
		Messages:  messages,
		System:    systemText,
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

// convertContent converts driver content blocks to Anthropic format.
func convertContent(blocks []content.ContentBlock) ([]contentBlock, error) {
	if len(blocks) == 0 {
		return []contentBlock{{Type: "text", Text: ""}}, nil
	}

	converted := make([]contentBlock, 0, len(blocks))
	for _, block := range blocks {
		if block.Type != content.ContentTypeText {
			return nil, fmt.Errorf("unsupported content type: %s", block.Type)
		}
		converted = append(converted, contentBlock{
			Type: "text",
			Text: block.Text,
		})
	}

	return converted, nil
}
