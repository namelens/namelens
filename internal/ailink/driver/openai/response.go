package openai

import (
	"fmt"

	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
)

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
