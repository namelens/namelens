package anthropic

import (
	"fmt"
	"strings"

	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
)

// messagesResponse is the response from the /v1/messages endpoint.
type messagesResponse struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Role         string                 `json:"role"`
	Content      []responseContentBlock `json:"content"`
	Model        string                 `json:"model"`
	StopReason   string                 `json:"stop_reason"`
	StopSequence *string                `json:"stop_sequence,omitempty"`
	Usage        *usage                 `json:"usage,omitempty"`
}

// responseContentBlock represents a content block in the response.
type responseContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// usage contains token usage statistics.
type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// toDriverResponse converts an Anthropic response to a driver.Response.
func toDriverResponse(resp *messagesResponse) (*driver.Response, error) {
	if resp == nil {
		return nil, fmt.Errorf("empty response")
	}

	// Extract text content from response blocks
	var textParts []string
	for _, block := range resp.Content {
		if block.Type == "text" && block.Text != "" {
			textParts = append(textParts, block.Text)
		}
	}

	// Combine all text blocks
	var text string
	if len(textParts) > 0 {
		text = textParts[0]
		for i := 1; i < len(textParts); i++ {
			text += "\n" + textParts[i]
		}
	}

	// Strip markdown fences if present (Anthropic sometimes wraps JSON in ```json ... ```)
	text = stripMarkdownFences(text)

	response := &driver.Response{
		Content: []content.ContentBlock{
			{Type: content.ContentTypeText, Text: text},
		},
		FinishReason: mapStopReason(resp.StopReason),
	}

	if resp.Usage != nil {
		response.Usage = &driver.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		}
	}

	return response, nil
}

// stripMarkdownFences removes markdown code fences (```json ... ``` or ``` ... ```)
// from the beginning and end of text. Anthropic sometimes wraps JSON responses
// in markdown fences despite being instructed not to.
func stripMarkdownFences(text string) string {
	text = strings.TrimSpace(text)

	// Match ```json\n ... \n``` or ``` ... ```
	if strings.HasPrefix(text, "```") {
		// Find first newline after opening fence
		firstNewline := strings.Index(text, "\n")
		if firstNewline != -1 {
			// Strip opening fence line
			text = text[firstNewline+1:]
		}
	}

	// Strip closing fence
	if strings.HasSuffix(text, "```") {
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	return text
}

// mapStopReason converts Anthropic stop reasons to standard finish reasons.
func mapStopReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	case "tool_use":
		return "tool_calls"
	default:
		return stopReason
	}
}
