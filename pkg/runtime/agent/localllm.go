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
// toolTurns contains intermediate assistant tool-use and tool-result messages
// for the caller to persist to session history for multi-turn continuity.
func (c *LocalLLMClient) Stream(
	ctx context.Context,
	systemPrompt string,
	history []Message,
	tools []Tool,
	caller ToolCaller,
	events chan<- LLMEvent,
) (string, []Message, error) {
	messages := historyToOpenAIMessages(systemPrompt, history)
	oaiTools := convertToolsForOpenAI(tools)

	var totalInputTokens, totalOutputTokens int
	var toolTurns []Message

	for {
		var roundText strings.Builder

		params := openai.ChatCompletionNewParams{
			Model:    c.model,
			Messages: messages,
			StreamOptions: openai.ChatCompletionStreamOptionsParam{
				IncludeUsage: openai.Bool(true),
			},
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

			// Capture usage from the final streaming chunk (requires IncludeUsage).
			if chunk.Usage.TotalTokens > 0 {
				totalInputTokens = int(chunk.Usage.PromptTokens)
				totalOutputTokens = int(chunk.Usage.CompletionTokens)
			}

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
				roundText.WriteString(delta.Content)
				select {
				case events <- LLMEvent{Type: EventTypeToken, Data: TokenData{Text: delta.Content}}:
				case <-ctx.Done():
					return roundText.String(), toolTurns, ctx.Err()
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
				return roundText.String(), toolTurns, nil
			}
			select {
			case events <- LLMEvent{Type: EventTypeError, Data: ErrorData{Message: err.Error()}}:
			default:
			}
			return roundText.String(), toolTurns, fmt.Errorf("local LLM stream: %w", err)
		}

		if finishReason != "tool_calls" || len(pending) == 0 {
			// Final round — emit metrics and return
			select {
			case events <- LLMEvent{Type: EventTypeMetrics, Data: MetricsData{
				TokensIn:  totalInputTokens,
				TokensOut: totalOutputTokens,
			}}:
			default:
			}
			select {
			case events <- LLMEvent{Type: EventTypeDone}:
			default:
			}
			return roundText.String(), toolTurns, nil
		}

		// Persist the assistant tool-use turn using ToolCalls for provider-agnostic storage.
		// OpenAI tool results are individual messages (no batching needed).
		toolCallBlocks := make([]ToolCallBlock, 0, len(pending))
		for i := 0; i < len(pending); i++ {
			if p, ok := pending[i]; ok {
				toolCallBlocks = append(toolCallBlocks, ToolCallBlock{
					ID:        p.id,
					Name:      p.name,
					Arguments: p.args.String(),
				})
			}
		}
		toolTurns = append(toolTurns, Message{
			Role:      "assistant",
			Content:   roundText.String(),
			ToolCalls: toolCallBlocks,
		})

		// Build assistant message with tool calls for the in-flight messages slice
		assistantMsg := buildAssistantMessageWithToolCalls(roundText.String(), pending)
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
				return roundText.String(), toolTurns, ctx.Err()
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
				return roundText.String(), toolTurns, ctx.Err()
			}

			messages = append(messages, openai.ToolMessage(output, p.id))
			toolTurns = append(toolTurns, Message{
				Role:       "tool",
				ToolCallID: p.id,
				Content:    output,
			})
		}
	}
}

// Close releases any held resources.
func (c *LocalLLMClient) Close() error { return nil }

// historyToOpenAIMessages converts history to OpenAI message params with optional system prompt.
// Tool-use assistant turns are reconstructed from ToolCalls. Tool-result messages map to
// the OpenAI "tool" role (one message per result — no batching required).
func historyToOpenAIMessages(systemPrompt string, history []Message) []openai.ChatCompletionMessageParamUnion {
	var msgs []openai.ChatCompletionMessageParamUnion
	if systemPrompt != "" {
		msgs = append(msgs, openai.SystemMessage(systemPrompt))
	}
	for _, m := range history {
		switch {
		case m.Role == "assistant" && len(m.ToolCalls) > 0:
			toolCalls := make([]openai.ChatCompletionMessageToolCallParam, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				toolCalls[i] = openai.ChatCompletionMessageToolCallParam{
					ID: tc.ID,
					Function: openai.ChatCompletionMessageToolCallFunctionParam{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				}
			}
			var param openai.ChatCompletionAssistantMessageParam
			if m.Content != "" {
				param.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(m.Content),
				}
			}
			param.ToolCalls = toolCalls
			msgs = append(msgs, openai.ChatCompletionMessageParamUnion{OfAssistant: &param})
		case m.Role == "tool":
			msgs = append(msgs, openai.ToolMessage(m.Content, m.ToolCallID))
		case m.Role == "user":
			msgs = append(msgs, openai.UserMessage(m.Content))
		case m.Role == "assistant":
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
