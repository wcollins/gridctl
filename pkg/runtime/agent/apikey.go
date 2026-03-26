package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicClient implements LLMClient using the Anthropic Messages API.
type AnthropicClient struct {
	client anthropic.Client
	model  string
}

// NewAnthropicClient creates a new Anthropic API client for the given model.
func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	return &AnthropicClient{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

// Stream runs the full agentic loop against the Anthropic Messages API.
// It streams tokens as they arrive, executes tool calls via caller, and loops
// until the model returns stop_reason "end_turn" (or another terminal reason).
// toolTurns contains intermediate assistant tool-use and tool-result messages
// for the caller to persist to session history for multi-turn continuity.
func (c *AnthropicClient) Stream(
	ctx context.Context,
	systemPrompt string,
	history []Message,
	tools []Tool,
	caller ToolCaller,
	events chan<- LLMEvent,
) (string, []Message, error) {
	anthropicTools := convertToolsForAnthropic(tools)
	messages := historyToAnthropicMessages(history)

	var totalInputTokens, totalOutputTokens int64
	var cacheReadTokens int64
	var toolTurns []Message

	for {
		var roundText strings.Builder

		params := anthropic.MessageNewParams{
			Model:     anthropic.Model(c.model),
			MaxTokens: 8192,
			Messages:  messages,
		}
		if systemPrompt != "" {
			params.System = []anthropic.TextBlockParam{{Text: systemPrompt}}
		}
		if len(anthropicTools) > 0 {
			params.Tools = anthropicTools
		}

		stream := c.client.Messages.NewStreaming(ctx, params)
		var acc anthropic.Message

		for stream.Next() {
			event := stream.Current()
			if err := acc.Accumulate(event); err != nil {
				continue // non-fatal accumulation errors
			}
			switch e := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch d := e.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					if d.Text != "" {
						roundText.WriteString(d.Text)
						select {
						case events <- LLMEvent{Type: EventTypeToken, Data: TokenData{Text: d.Text}}:
						case <-ctx.Done():
							return roundText.String(), toolTurns, ctx.Err()
						}
					}
				}
			}
		}
		if err := stream.Err(); err != nil {
			if ctx.Err() != nil {
				return roundText.String(), toolTurns, nil // context cancelled — clean exit
			}
			select {
			case events <- LLMEvent{Type: EventTypeError, Data: ErrorData{Message: err.Error()}}:
			default:
			}
			return roundText.String(), toolTurns, fmt.Errorf("anthropic stream: %w", err)
		}

		totalInputTokens += int64(acc.Usage.InputTokens)
		totalOutputTokens += int64(acc.Usage.OutputTokens)
		cacheReadTokens += int64(acc.Usage.CacheReadInputTokens)

		if acc.StopReason != anthropic.StopReasonToolUse {
			// Final round — emit metrics and return
			formatSavingsPct := 0.0
			if totalInputTokens > 0 && cacheReadTokens > 0 {
				formatSavingsPct = float64(cacheReadTokens) / float64(totalInputTokens) * 100
			}
			select {
			case events <- LLMEvent{Type: EventTypeMetrics, Data: MetricsData{
				TokensIn:         int(totalInputTokens),
				TokensOut:        int(totalOutputTokens),
				FormatSavingsPct: formatSavingsPct,
			}}:
			default:
			}
			select {
			case events <- LLMEvent{Type: EventTypeDone}:
			default:
			}
			return roundText.String(), toolTurns, nil
		}

		// Persist the assistant tool-use turn. RawParam stores the full serialized
		// MessageParam (text + tool_use blocks) for accurate Anthropic reconstruction
		// without needing to rebuild SDK union types from scratch.
		assistantParam := acc.ToParam()
		var toolCallBlocks []ToolCallBlock
		for _, block := range acc.Content {
			if toolUse, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				toolCallBlocks = append(toolCallBlocks, ToolCallBlock{
					ID:        toolUse.ID,
					Name:      toolUse.Name,
					Arguments: string(toolUse.Input),
				})
			}
		}
		assistantTurn := Message{
			Role:      "assistant",
			Content:   roundText.String(),
			ToolCalls: toolCallBlocks,
		}
		if raw, err := json.Marshal(assistantParam); err == nil {
			assistantTurn.RawParam = raw
		}
		toolTurns = append(toolTurns, assistantTurn)

		// Execute each tool call and persist individual tool-result messages.
		// historyToAnthropicMessages groups consecutive "tool" messages back into
		// a single batched user message on the next request.
		var toolResults []anthropic.ContentBlockParamUnion
		for _, block := range acc.Content {
			toolUse, ok := block.AsAny().(anthropic.ToolUseBlock)
			if !ok {
				continue
			}

			var args map[string]any
			if len(toolUse.Input) > 0 {
				_ = json.Unmarshal(toolUse.Input, &args)
			}

			serverName := serverNameForTool(toolUse.Name, tools)
			select {
			case events <- LLMEvent{Type: EventTypeToolCallStart, Data: ToolCallStartData{
				ToolName:   toolUse.Name,
				ServerName: serverName,
				Input:      args,
			}}:
			case <-ctx.Done():
				return roundText.String(), toolTurns, ctx.Err()
			}

			start := time.Now()
			result, callErr := caller.CallTool(ctx, toolUse.Name, args)
			durationMs := time.Since(start).Milliseconds()

			output := result.Content
			isError := callErr != nil || result.IsError
			if callErr != nil {
				output = callErr.Error()
			}

			select {
			case events <- LLMEvent{Type: EventTypeToolCallEnd, Data: ToolCallEndData{
				ToolName:   toolUse.Name,
				Output:     output,
				DurationMs: durationMs,
			}}:
			case <-ctx.Done():
				return roundText.String(), toolTurns, ctx.Err()
			}

			toolResults = append(toolResults, anthropic.NewToolResultBlock(toolUse.ID, output, isError))
			toolTurns = append(toolTurns, Message{
				Role:       "tool",
				ToolCallID: toolUse.ID,
				Content:    output,
			})
		}

		// Add assistant message + batched tool results and loop.
		messages = append(messages, assistantParam)
		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}
}

// Close releases any held resources (the HTTP client is managed externally).
func (c *AnthropicClient) Close() error { return nil }

// convertToolsForAnthropic converts agent.Tool slice to Anthropic ToolUnionParam slice.
func convertToolsForAnthropic(tools []Tool) []anthropic.ToolUnionParam {
	result := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		var schema map[string]any
		if err := json.Unmarshal(t.InputSchema, &schema); err != nil {
			schema = map[string]any{"type": "object"}
		}

		props := schema["properties"]
		var required []string
		if reqRaw, ok := schema["required"].([]any); ok {
			for _, r := range reqRaw {
				if s, ok := r.(string); ok {
					required = append(required, s)
				}
			}
		}

		toolParam := anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: props,
				Required:   required,
			},
		}
		result = append(result, anthropic.ToolUnionParam{OfTool: &toolParam})
	}
	return result
}

// historyToAnthropicMessages converts conversation history to Anthropic MessageParam slice.
// Tool-use assistant turns use RawParam for accurate round-trip reconstruction.
// Consecutive "tool" role messages are grouped into a single batched user message.
func historyToAnthropicMessages(history []Message) []anthropic.MessageParam {
	msgs := make([]anthropic.MessageParam, 0, len(history))
	i := 0
	for i < len(history) {
		m := history[i]

		switch {
		case len(m.RawParam) > 0:
			// Provider-specific raw param: used for Anthropic tool-use assistant turns.
			var param anthropic.MessageParam
			if err := json.Unmarshal(m.RawParam, &param); err == nil {
				msgs = append(msgs, param)
			}
			i++

		case m.Role == "tool":
			// Group consecutive tool-result messages into one Anthropic user message.
			var toolResults []anthropic.ContentBlockParamUnion
			for i < len(history) && history[i].Role == "tool" {
				tr := history[i]
				toolResults = append(toolResults, anthropic.NewToolResultBlock(tr.ToolCallID, tr.Content, false))
				i++
			}
			if len(toolResults) > 0 {
				msgs = append(msgs, anthropic.NewUserMessage(toolResults...))
			}

		case m.Role == "user":
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
			i++

		case m.Role == "assistant":
			msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
			i++

		default:
			i++
		}
	}
	return msgs
}

// serverNameForTool finds the ServerName for a tool by its prefixed name.
func serverNameForTool(toolName string, tools []Tool) string {
	for _, t := range tools {
		if t.Name == toolName {
			return t.ServerName
		}
	}
	return ""
}
