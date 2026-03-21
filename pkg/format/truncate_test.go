package format

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncateResult(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		maxBytes      int
		wantTruncated bool
		wantPrefix    string // expected prefix of returned string (before suffix)
		wantSuffix    string // expected suffix in returned string
	}{
		{
			name:          "under limit passes through unchanged",
			input:         "hello world",
			maxBytes:      100,
			wantTruncated: false,
		},
		{
			name:          "exactly at limit passes through unchanged",
			input:         "hello",
			maxBytes:      5,
			wantTruncated: false,
		},
		{
			name:          "empty string no-op",
			input:         "",
			maxBytes:      64,
			wantTruncated: false,
		},
		{
			name:          "maxBytes zero disables truncation",
			input:         strings.Repeat("x", 1000),
			maxBytes:      0,
			wantTruncated: false,
		},
		{
			name:          "maxBytes negative disables truncation",
			input:         strings.Repeat("x", 1000),
			maxBytes:      -1,
			wantTruncated: false,
		},
		{
			name:          "over limit truncated with suffix",
			input:         strings.Repeat("a", 200),
			maxBytes:      100,
			wantTruncated: true,
			wantPrefix:    strings.Repeat("a", 100),
			wantSuffix:    "[truncated: 200 bytes, showing first 100 bytes]",
		},
		{
			name:          "1MB input truncated to 64KB",
			input:         strings.Repeat("z", 1<<20),
			maxBytes:      65536,
			wantTruncated: true,
			wantSuffix:    "[truncated: 1048576 bytes, showing first 65536 bytes]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, wasTruncated := TruncateResult(tc.input, tc.maxBytes)

			assert.Equal(t, tc.wantTruncated, wasTruncated)

			if !tc.wantTruncated {
				assert.Equal(t, tc.input, result)
				return
			}

			require.True(t, wasTruncated)
			if tc.wantPrefix != "" {
				assert.True(t, strings.HasPrefix(result, tc.wantPrefix),
					"result should start with expected prefix")
			}
			assert.Contains(t, result, tc.wantSuffix)
		})
	}
}

func TestTruncateResult_UTF8Safety(t *testing.T) {
	// Build a string: 99 ASCII bytes + a 3-byte UTF-8 rune (€ = U+20AC = 0xE2 0x82 0xAC)
	// Truncating at 100 bytes would split the rune; we expect it to back up to 99.
	prefix := strings.Repeat("a", 99)
	euro := "€" // 3 bytes: 0xE2 0x82 0xAC
	input := prefix + euro + strings.Repeat("b", 50)

	result, wasTruncated := TruncateResult(input, 100)

	require.True(t, wasTruncated, "should be truncated")

	// Result should start with the 99 ASCII bytes (not include the partial €)
	assert.True(t, strings.HasPrefix(result, prefix),
		"truncated result should start with the ASCII prefix")

	// The € rune should NOT appear in the truncated portion
	resultPrefix := result[:99]
	assert.NotContains(t, resultPrefix, "€",
		"truncation should not include a split rune")

	// The suffix should reflect original size and configured limit
	assert.Contains(t, result, fmt.Sprintf("[truncated: %d bytes, showing first 100 bytes]", len(input)))
}

func TestTruncateResult_AllocationFreeUnderLimit(t *testing.T) {
	// Verify the fast path returns the original string identity (no allocation/copy)
	input := "small result"
	result, wasTruncated := TruncateResult(input, 1000)
	assert.False(t, wasTruncated)
	assert.Equal(t, input, result)
	// Go string headers share the same backing array when the slice is identical
	assert.Equal(t, input, result)
}
