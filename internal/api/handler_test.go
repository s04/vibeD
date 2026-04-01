package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vibed-project/vibeD/internal/config"
	"github.com/vibed-project/vibeD/internal/deployer"
	"github.com/vibed-project/vibeD/internal/metrics"
	"github.com/vibed-project/vibeD/internal/operations"
	"github.com/vibed-project/vibeD/internal/orchestrator"
	"github.com/vibed-project/vibeD/internal/storage"
	"github.com/vibed-project/vibeD/internal/store"
)

var (
	testMetricsOnce sync.Once
	testMetricsInst *metrics.Metrics
)

func testMetrics() *metrics.Metrics {
	testMetricsOnce.Do(func() {
		testMetricsInst = metrics.New()
	})
	return testMetricsInst
}

func TestAPI_DeployArtifact_ValidationError(t *testing.T) {
	handler := testHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/artifacts", bytes.NewBufferString(`{"name":"demo","files":{}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "at least one file is required")
}

func TestAPI_DeployArtifact_FileLimitError(t *testing.T) {
	handler := testHandlerWithLimits(t, config.LimitsConfig{
		MaxFileCount:     2,
		MaxTotalFileSize: 1024,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/artifacts", bytes.NewBufferString(`{"name":"demo","files":{"a.txt":"1","b.txt":"2","c.txt":"3"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `invalid input for "files"`)
	assert.Contains(t, rec.Body.String(), "too many files")
}

func TestAPI_UpdateArtifact_NotFound(t *testing.T) {
	handler := testHandler(t)
	body, err := json.Marshal(operations.UpdateArtifactRequest{
		Files: map[string]string{"index.html": "<h1>hi</h1>"},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/artifacts/missing-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "not found")
}

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	return testHandlerWithLimits(t, config.LimitsConfig{
		MaxFileCount:     500,
		MaxTotalFileSize: 50 * 1024 * 1024,
	})
}

func testHandlerWithLimits(t *testing.T, limits config.LimitsConfig) http.Handler {
	t.Helper()

	tmpDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	localStorage, err := storage.NewLocalStorage(tmpDir)
	require.NoError(t, err)

	cfg := &config.Config{
		Deployment: config.DeploymentConfig{
			PreferredTarget: "kubernetes",
			Namespace:       "default",
		},
		Storage: config.StorageConfig{
			Backend: "local",
			Local: config.LocalStorageConfig{
				BasePath: tmpDir,
			},
		},
		Registry: config.RegistryConfig{
			Enabled: false,
		},
		Limits: limits,
	}

	m := testMetrics()

	orch := orchestrator.NewOrchestrator(
		cfg,
		nil,
		nil,
		deployer.NewFactory(),
		localStorage,
		store.NewMemoryStore(),
		m,
		nil,
		nil,
		nil,
		logger,
	)

	return NewHandler(orch, cfg, nil, m, nil)
}
