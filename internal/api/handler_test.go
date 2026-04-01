package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vibed-project/vibeD/internal/config"
	"github.com/vibed-project/vibeD/internal/deployer"
	"github.com/vibed-project/vibeD/internal/metrics"
	"github.com/vibed-project/vibeD/internal/operations"
	"github.com/vibed-project/vibeD/internal/orchestrator"
	"github.com/vibed-project/vibeD/internal/storage"
	"github.com/vibed-project/vibeD/internal/store"
	pkgapi "github.com/vibed-project/vibeD/pkg/api"
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

func TestAPI_ListArtifacts_Pagination(t *testing.T) {
	handler := testHandlerWithSQLiteArtifacts(t, []*pkgapi.Artifact{
		newTestAPIArtifact("a1", "app-1"),
		newTestAPIArtifact("a2", "app-2"),
		newTestAPIArtifact("a3", "app-3"),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/artifacts?offset=1&limit=1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp operations.ListArtifactsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Artifacts, 1)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 1, resp.Offset)
	assert.Equal(t, 1, resp.Limit)
	assert.Equal(t, "a2", resp.Artifacts[0].ID)
}

func TestAPI_ListArtifacts_DefaultPagination(t *testing.T) {
	artifacts := make([]*pkgapi.Artifact, 0, 60)
	for i := 1; i <= 60; i++ {
		artifacts = append(artifacts, newTestAPIArtifact(
			"a"+strconv.Itoa(i),
			"app-"+strconv.Itoa(i),
		))
	}

	handler := testHandlerWithSQLiteArtifacts(t, artifacts)
	req := httptest.NewRequest(http.MethodGet, "/api/artifacts", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp operations.ListArtifactsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Artifacts, 50)
	assert.Equal(t, 60, resp.Total)
	assert.Equal(t, 0, resp.Offset)
	assert.Equal(t, 50, resp.Limit)
}

func TestAPI_ListArtifacts_ClampsLimit(t *testing.T) {
	artifacts := make([]*pkgapi.Artifact, 0, 220)
	for i := 1; i <= 220; i++ {
		artifacts = append(artifacts, newTestAPIArtifact(
			"a"+strconv.Itoa(i),
			"app-"+strconv.Itoa(i),
		))
	}

	handler := testHandlerWithSQLiteArtifacts(t, artifacts)
	req := httptest.NewRequest(http.MethodGet, "/api/artifacts?limit=500", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp operations.ListArtifactsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Artifacts, 200)
	assert.Equal(t, 220, resp.Total)
	assert.Equal(t, 200, resp.Limit)
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
	return testHandlerWithStore(t, limits, store.NewMemoryStore())
}

func testHandlerWithArtifacts(t *testing.T, artifacts []*pkgapi.Artifact) http.Handler {
	t.Helper()
	memStore := store.NewMemoryStore()
	ctx := context.Background()
	for _, artifact := range artifacts {
		require.NoError(t, memStore.Create(ctx, artifact))
	}

	return testHandlerWithStore(t, config.LimitsConfig{
		MaxFileCount:     500,
		MaxTotalFileSize: 50 * 1024 * 1024,
	}, memStore)
}

func testHandlerWithSQLiteArtifacts(t *testing.T, artifacts []*pkgapi.Artifact) http.Handler {
	t.Helper()
	sqliteStore, err := store.NewSQLiteStore(filepath.Join(t.TempDir(), "artifacts.db"))
	require.NoError(t, err)
	t.Cleanup(func() { sqliteStore.Close() })

	ctx := context.Background()
	for i, artifact := range artifacts {
		artifact.CreatedAt = artifact.CreatedAt.Add(time.Duration(i) * time.Second)
		artifact.UpdatedAt = artifact.CreatedAt
		require.NoError(t, sqliteStore.Create(ctx, artifact))
	}

	return testHandlerWithStore(t, config.LimitsConfig{
		MaxFileCount:     500,
		MaxTotalFileSize: 50 * 1024 * 1024,
	}, sqliteStore)
}

func testHandlerWithStore(t *testing.T, limits config.LimitsConfig, artifactStore store.ArtifactStore) http.Handler {
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
		artifactStore,
		m,
		nil,
		nil,
		nil,
		logger,
	)

	return NewHandler(orch, cfg, nil, m, nil)
}

func newTestAPIArtifact(id, name string) *pkgapi.Artifact {
	ts := time.Unix(int64(len(name)), 0).UTC()
	return &pkgapi.Artifact{
		ID:        id,
		Name:      name,
		Status:    pkgapi.StatusRunning,
		Target:    pkgapi.TargetKubernetes,
		ImageRef:  "example.com/test:latest",
		URL:       "https://example.com/" + name,
		CreatedAt: ts,
		UpdatedAt: ts,
		Version:   1,
	}
}
