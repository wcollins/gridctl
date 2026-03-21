package format

import (
	"fmt"
	"unicode/utf8"
)

// TruncateResult truncates result to at most maxBytes, trimming at a UTF-8 rune boundary.
// If result is at or under maxBytes, it is returned unchanged and wasTruncated is false.
// If maxBytes <= 0, the result is returned unchanged.
func TruncateResult(result string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(result) <= maxBytes {
		return result, false
	}

	// Back up from maxBytes to find a safe UTF-8 rune boundary.
	// Continuation bytes have the form 10xxxxxx; RuneStart returns false for them.
	i := maxBytes
	for i > 0 && !utf8.RuneStart(result[i]) {
		i--
	}

	suffix := fmt.Sprintf(" [truncated: %d bytes, showing first %d bytes]", len(result), maxBytes)
	return result[:i] + suffix, true
}
