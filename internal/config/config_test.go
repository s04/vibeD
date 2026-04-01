package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadRejectsInvalidWebhookTimeout(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "vibed.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`
registry:
  enabled: true
  url: registry.local
webhooks:
  - url: https://hooks.example.com/vibed
    events: ["deploy.completed"]
    timeout: nope
`), 0o600))

	_, err := Load(cfgPath)
	require.Error(t, err)
	require.ErrorContains(t, err, `webhooks[0].timeout`)
}
