package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samber/oops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func TestReadAppliesDefaultsAndValidates(t *testing.T) {
	// Empty file → defaults fill everything → validation passes.
	path := writeConfig(t, "")
	cfg, err := config.Read(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, ":8080", cfg.Address)
	assert.Equal(t, "nats://localhost:4222", cfg.Nats.URL)
}

func TestReadRejectsInvalidConfigWithAllViolations(t *testing.T) {
	// Override NATS URL to an invalid value and zero out LockExpiry.
	// The validator must report BOTH violations (multi-error).
	path := writeConfig(t, `
nats:
  url: not-a-url
redis:
  lock_expiry: 0s
`)
	_, err := config.Read(path)
	require.Error(t, err)

	oe, ok := oops.AsOops(err)
	require.True(t, ok)
	assert.Equal(t, config.ErrCodeConfigValidateFailed, oe.Code())

	// Both violations must be present (multi-error, not fail-fast).
	ctx := oe.Context()
	violations, ok := ctx["violations"].([]string)
	require.True(t, ok, "violations context must be []string")
	require.GreaterOrEqual(t, len(violations), 2, "expected at least 2 violations (multi-error), got %d: %v", len(violations), violations)

	joined := strings.Join(violations, " | ")
	assert.Contains(t, joined, "Nats.URL")
	assert.Contains(t, joined, "Redis.LockExpiry")
}
