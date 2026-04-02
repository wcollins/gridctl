// Package token provides token counting for MCP tool call content.
//
// The default implementation uses cl100k_base (the BPE vocabulary used by
// GPT-4 and related models). Claude 3+ uses a different, unpublished vocabulary,
// so cl100k_base counts are an approximation for Claude models — typically within
// 10-15% for English and code content. This is intentional: the interface exists
// to allow swapping implementations without changing consumers, and cl100k_base
// is a meaningful improvement over the 4-bytes/token heuristic it replaces.
//
// Users who need exact Claude token counts can enable gateway.tokenizer: api
// in their stack.yaml, which routes counting through Anthropic's count_tokens
// endpoint (Anthropic-specific, requires network access).
package token

import (
	"encoding/json"
	"fmt"

	"github.com/tiktoken-go/tokenizer"
)

// Counter estimates token counts for text content.
// Implementations may vary in accuracy — the interface allows swapping
// a heuristic counter for a tiktoken-based one without changing consumers.
type Counter interface {
	// Count returns the estimated number of tokens in the given text.
	Count(text string) int
}

// HeuristicCounter estimates tokens using a simple bytes-per-token ratio.
// This is fast and zero-dependency but approximate. Suitable for visibility
// purposes where exact counts are not required.
type HeuristicCounter struct {
	bytesPerToken int
}

// NewHeuristicCounter creates a counter that estimates tokens at the given
// bytes-per-token ratio. A ratio of 4 is a reasonable default for English text.
func NewHeuristicCounter(bytesPerToken int) *HeuristicCounter {
	if bytesPerToken <= 0 {
		bytesPerToken = 4
	}
	return &HeuristicCounter{bytesPerToken: bytesPerToken}
}

// Count returns the estimated token count for the given text.
func (c *HeuristicCounter) Count(text string) int {
	n := len(text)
	if n == 0 {
		return 0
	}
	// Round up to avoid underestimating
	return (n + c.bytesPerToken - 1) / c.bytesPerToken
}

// CountJSON estimates the token count for a value by marshaling it to JSON first.
// This is useful for counting tool call arguments (map[string]any).
func CountJSON(c Counter, v any) int {
	data, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	return c.Count(string(data))
}

// TiktokenCounter counts tokens using the cl100k_base BPE encoding.
// This is the vocabulary used by GPT-4 and related models and is a close
// approximation for Claude models (whose vocabulary is unpublished).
type TiktokenCounter struct {
	enc tokenizer.Codec
}

// NewTiktokenCounter creates a counter using the cl100k_base encoding.
// The vocabulary is loaded eagerly at construction time to surface any
// initialization failure at startup rather than during request handling.
func NewTiktokenCounter() (*TiktokenCounter, error) {
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return nil, fmt.Errorf("token: failed to load cl100k_base vocabulary: %w", err)
	}
	// Force eager loading: encode a test string and discard the result so
	// any lazy initialization happens now, not on the first tool call.
	if _, _, err := enc.Encode("hello"); err != nil {
		return nil, fmt.Errorf("token: cl100k_base warm-up failed: %w", err)
	}
	return &TiktokenCounter{enc: enc}, nil
}

// Count returns the token count for the given text using cl100k_base encoding.
func (c *TiktokenCounter) Count(text string) int {
	if text == "" {
		return 0
	}
	ids, _, err := c.enc.Encode(text)
	if err != nil {
		// Encoding errors are unexpected; fall back to a rough estimate.
		return (len(text) + 3) / 4
	}
	return len(ids)
}
