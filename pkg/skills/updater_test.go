package skills

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldCheckUpdates(t *testing.T) {
	origNoCheck := os.Getenv("GRIDCTL_NO_SKILL_UPDATE_CHECK")
	origCI := os.Getenv("CI")
	defer func() {
		os.Setenv("GRIDCTL_NO_SKILL_UPDATE_CHECK", origNoCheck)
		os.Setenv("CI", origCI)
	}()

	// Default: should check
	os.Unsetenv("GRIDCTL_NO_SKILL_UPDATE_CHECK")
	os.Unsetenv("CI")
	assert.True(t, ShouldCheckUpdates())

	// Disabled via env var
	os.Setenv("GRIDCTL_NO_SKILL_UPDATE_CHECK", "1")
	assert.False(t, ShouldCheckUpdates())

	// Disabled in CI
	os.Unsetenv("GRIDCTL_NO_SKILL_UPDATE_CHECK")
	os.Setenv("CI", "true")
	assert.False(t, ShouldCheckUpdates())
}

func TestFormatUpdateNotice(t *testing.T) {
	// When no cache exists, returns empty (or possibly a notice string)
	// Just ensure no panic
	_ = FormatUpdateNotice()
}
