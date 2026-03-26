package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	openai "github.com/openai/openai-go"
	openaiopt "github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

// LocalLLMClient implements LLMClient for any OpenAI-compatible endpoint.
// It is used for Path C (Ollama / local inference) and can be pointed at any
// OpenAI-compatible API by setting a custom BaseURL.
type LocalLLMClient struct {
	client  *openai.Client
	model   string
	baseURL string
}

// NewLocalLLMClient creates a client for an OpenAI-compatible endpoint.
// baseURL defaults to http://localhost:11434/v1 (Ollama) when empty.
func NewLocalLLMClient(baseURL, model string) *LocalLLMClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	client := openai.NewClient(
		openaiopt.WithBaseURL(baseURL),
		openaiopt.WithAPIKey("ollama"), // non-empty key required by SDK; Ollama ignores it
	)
	return &LocalLLMClient{client: &client, model: model, baseURL: baseURL}
}

// NewLocalLLMClientWithKey creates a client for an OpenAI-compatible endpoint with an explicit API key.
// Used for hosted OpenAI-compatible APIs (e.g. api.openai.com) that require real authentication.
func NewLocalLLMClientWithKey(baseURL, model, apiKey string) *LocalLLMClient {
	client := openai.NewClient(
		openaiopt.WithBaseURL(baseURL),
		openaiopt.WithAPIKey(apiKey),
	)
	return &LocalLLMClient{client: &client, model: model, baseURL: baseURL}
}

// Stream runs the agentic loop against the configured OpenAI-compatible endpoint.
func (c *LocalLLMClient) Stream(
	ctx context.Context,
	systemPrompt string,
	history []Message,
	tools []Tool,
	caller ToolCaller,
	events chan<- LLMEvent,
) (string, error) {
	messages := historyToOpenAIMessages(systemPrompt, history)
	oaiTools := convertToolsForOpenAI(tools)

	var finalResponse strings.Builder

	for {
		params := openai.ChatCompletionNewParams{
			Model:    c.model,
			Messages: messages,
		}
		if len(oaiTools) > 0 {
			params.Tools = oaiTools
		}

		stream := c.client.Chat.Completions.NewStreaming(ctx, params)

		// Accumulated tool call state: index → {id, name, argsBuilder}
		pending := map[int]*localToolCall{}
		var finishReason string

		for stream.Next() {
			chunk := stream.Current()
			if len(chunk.Choices) == 0 {
				continue
			}
			choice := chunk.Choices[0]
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
			delta := choice.Delta

			// Stream text tokens
			if delta.Content != "" {
				finalResponse.WriteString(delta.Content)
				select {
				case events <- LLMEvent{Type: EventTypeToken, Data: TokenData{Text: delta.Content}}:
				case <-ctx.Done():
					return finalResponse.String(), ctx.Err()
				}
			}

			// Accumulate tool call fragments
			for _, tc := range delta.ToolCalls {
				idx := int(tc.Index)
				if _, ok := pending[idx]; !ok {
					pending[idx] = &localToolCall{}
				}
				p := pending[idx]
				if tc.ID != "" {
					p.id = tc.ID
				}
				if tc.Function.Name != "" {
					p.name = tc.Function.Name
				}
				p.args.WriteString(tc.Function.Arguments)
			}
		}
		if err := stream.Err(); err != nil {
			if ctx.Err() != nil {
				return finalResponse.String(), nil
			}
			select {
			case events <- LLMEvent{Type: EventTypeError, Data: ErrorData{Message: err.Error()}}:
			default:
			}
			return finalResponse.String(), fmt.Errorf("local LLM stream: %w", err)
		}

		if finishReason != "tool_calls" || len(pending) == 0 {
			break
		}

		// Build assistant message with tool calls for history
		assistantMsg := buildAssistantMessageWithToolCalls(finalResponse.String(), pending)
		messages = append(messages, assistantMsg)

		// Execute tool calls and collect results
		for i := 0; i < len(pending); i++ {
			p, ok := pending[i]
			if !ok {
				continue
			}
			var args map[string]any
			if p.args.Len() > 0 {
				_ = json.Unmarshal([]byte(p.args.String()), &args)
			}

			serverName := serverNameForTool(p.name, tools)
			select {
			case events <- LLMEvent{Type: EventTypeToolCallStart, Data: ToolCallStartData{
				ToolName:   p.name,
				ServerName: serverName,
				Input:      args,
			}}:
			case <-ctx.Done():
				return finalResponse.String(), ctx.Err()
			}

			start := time.Now()
			result, callErr := caller.CallTool(ctx, p.name, args)
			durationMs := time.Since(start).Milliseconds()

			output := result.Content
			if callErr != nil {
				output = callErr.Error()
			}

			select {
			case events <- LLMEvent{Type: EventTypeToolCallEnd, Data: ToolCallEndData{
				ToolName:   p.name,
				Output:     output,
				DurationMs: durationMs,
			}}:
			case <-ctx.Done():
				return finalResponse.String(), ctx.Err()
			}

			messages = append(messages, openai.ToolMessage(output, p.id))
		}
	}

	select {
	case events <- LLMEvent{Type: EventTypeMetrics, Data: MetricsData{
		TokensIn:  0, // OpenAI-compatible streaming doesn't always include usage
		TokensOut: 0,
	}}:
	default:
	}
	select {
	case events <- LLMEvent{Type: EventTypeDone}:
	default:
	}

	return finalResponse.String(), nil
}

// Close releases any held resources.
func (c *LocalLLMClient) Close() error { return nil }

// historyToOpenAIMessages converts history to OpenAI message params with optional system prompt.
func historyToOpenAIMessages(systemPrompt string, history []Message) []openai.ChatCompletionMessageParamUnion {
	var msgs []openai.ChatCompletionMessageParamUnion
	if systemPrompt != "" {
		msgs = append(msgs, openai.SystemMessage(systemPrompt))
	}
	for _, m := range history {
		switch m.Role {
		case "user":
			msgs = append(msgs, openai.UserMessage(m.Content))
		case "assistant":
			msgs = append(msgs, openai.AssistantMessage(m.Content))
		}
	}
	return msgs
}

// convertToolsForOpenAI converts agent.Tool slice to OpenAI ChatCompletionToolParam slice.
func convertToolsForOpenAI(tools []Tool) []openai.ChatCompletionToolParam {
	result := make([]openai.ChatCompletionToolParam, 0, len(tools))
	for _, t := range tools {
		var params shared.FunctionParameters
		if err := json.Unmarshal(t.InputSchema, &params); err != nil {
			params = shared.FunctionParameters{"type": "object"}
		}
		result = append(result, openai.ChatCompletionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:        t.Name,
				Description: openai.String(t.Description),
				Parameters:  params,
			},
		})
	}
	return result
}

// localToolCall accumulates a streaming tool call fragment.
type localToolCall struct {
	id   string
	name string
	args strings.Builder
}

// buildAssistantMessageWithToolCalls creates an assistant message param with tool_call blocks.
func buildAssistantMessageWithToolCalls(content string, calls map[int]*localToolCall) openai.ChatCompletionMessageParamUnion {
	toolCalls := make([]openai.ChatCompletionMessageToolCallParam, 0, len(calls))
	for i := 0; i < len(calls); i++ {
		p, ok := calls[i]
		if !ok {
			continue
		}
		toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
			ID: p.id,
			Function: openai.ChatCompletionMessageToolCallFunctionParam{
				Name:      p.name,
				Arguments: p.args.String(),
			},
		})
	}
	var param openai.ChatCompletionAssistantMessageParam
	if content != "" {
		param.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
			OfString: openai.String(content),
		}
	}
	param.ToolCalls = toolCalls
	return openai.ChatCompletionMessageParamUnion{OfAssistant: &param}
}
