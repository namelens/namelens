package xai

import (
	"fmt"
	"strings"

	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
)

// chatCompletionResponse is for the legacy /v1/chat/completions endpoint.
type chatCompletionResponse struct {
	Choices []choice `json:"choices"`
	Usage   *usage   `json:"usage,omitempty"`
}

type choice struct {
	Message      chatResponseMessage `json:"message"`
	FinishReason string              `json:"finish_reason"`
}

type chatResponseMessage struct {
	Content   string     `json:"content"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

type toolCall struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Function *toolInvoke `json:"function,omitempty"`
}

type toolInvoke struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// responsesAPIResponse is for the new /v1/responses endpoint.
type responsesAPIResponse struct {
	ID     string          `json:"id"`
	Output []outputItem    `json:"output"`
	Usage  *responsesUsage `json:"usage,omitempty"`
}

type outputItem struct {
	Type    string          `json:"type"`
	Role    string          `json:"role,omitempty"`
	Content []outputContent `json:"content,omitempty"` // For message type
	Text    string          `json:"text,omitempty"`    // Alternative text location
}

type outputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type responsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func toDriverResponse(resp *chatCompletionResponse) (*driver.Response, error) {
	if resp == nil || len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response choices")
	}

	choice := resp.Choices[0]
	contentBlock := content.ContentBlock{Type: content.ContentTypeText, Text: choice.Message.Content}
	response := &driver.Response{
		Content:      []content.ContentBlock{contentBlock},
		FinishReason: choice.FinishReason,
	}

	if resp.Usage != nil {
		response.Usage = &driver.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	if len(choice.Message.ToolCalls) > 0 {
		calls := make([]driver.ToolCall, 0, len(choice.Message.ToolCalls))
		for _, call := range choice.Message.ToolCalls {
			toolCall := driver.ToolCall{ID: call.ID, Type: call.Type}
			if call.Function != nil {
				toolCall.Name = call.Function.Name
				toolCall.Input = map[string]any{"arguments": call.Function.Arguments}
			}
			calls = append(calls, toolCall)
		}
		response.ToolCalls = calls
	}

	return response, nil
}

// toDriverResponseFromResponses converts a /v1/responses API response to driver.Response.
func toDriverResponseFromResponses(resp *responsesAPIResponse) (*driver.Response, error) {
	if resp == nil || len(resp.Output) == 0 {
		return nil, fmt.Errorf("empty response output")
	}

	// Extract text content from output items
	var textParts []string
	for _, item := range resp.Output {
		// The responses API returns different output types
		// "message" type contains the assistant's text response with nested content array
		// "web_search_call", "x_search_call" etc. are tool invocations (skip these)
		switch item.Type {
		case "message":
			// Extract text from nested content array
			for _, c := range item.Content {
				if c.Type == "output_text" && c.Text != "" {
					textParts = append(textParts, c.Text)
				}
			}
			// Fallback to direct text field
			if item.Text != "" {
				textParts = append(textParts, item.Text)
			}
		case "text":
			if item.Text != "" {
				textParts = append(textParts, item.Text)
			}
		}
	}

	text := strings.Join(textParts, "\n")
	contentBlock := content.ContentBlock{Type: content.ContentTypeText, Text: text}
	response := &driver.Response{
		Content:      []content.ContentBlock{contentBlock},
		FinishReason: "stop",
	}

	if resp.Usage != nil {
		response.Usage = &driver.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return response, nil
}
