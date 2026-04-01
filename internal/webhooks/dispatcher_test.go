package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vibed-project/vibeD/internal/config"
	"github.com/vibed-project/vibeD/internal/events"
)

func TestDispatcherDeliversMatchingEvent(t *testing.T) {
	bus := events.NewEventBus()
	requests := make(chan struct {
		header http.Header
		body   Payload
	}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload Payload
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		requests <- struct {
			header http.Header
			body   Payload
		}{
			header: r.Header.Clone(),
			body:   payload,
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	dispatcher, err := NewDispatcher([]config.WebhookConfig{{
		URL:    server.URL,
		Events: []string{"deploy.completed"},
		Secret: "super-secret",
	}}, bus, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dispatcher.Start(ctx)

	bus.Publish(events.Event{
		Type:         events.ArtifactStatusChanged,
		ArtifactID:   "art-1",
		ArtifactName: "demo",
		OwnerID:      "alice",
		Target:       "knative",
		URL:          "https://demo.example.com",
		Status:       "running",
		Timestamp:    time.Now().UTC(),
	})

	select {
	case req := <-requests:
		assert.Equal(t, "deploy.completed", req.body.Event)
		assert.Equal(t, "art-1", req.body.ArtifactID)
		assert.Equal(t, "demo", req.body.ArtifactName)
		assert.Equal(t, "knative", req.body.Target)
		assert.Equal(t, "https://demo.example.com", req.body.URL)
		assert.Equal(t, "artifact.status_changed", req.body.Metadata["raw_event"])
		assert.Equal(t, "alice", req.body.Metadata["owner_id"])
		assert.Equal(t, "running", req.body.Metadata["status"])

		assert.Equal(t, "deploy.completed", req.header.Get("X-VibeD-Event"))
		assert.NotEmpty(t, req.header.Get("X-VibeD-Delivery"))
		assert.True(t, verifySignature(req.header.Get("X-VibeD-Signature"), req.body, "super-secret"))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook delivery")
	}
}

func TestDispatcherRetriesFailedDelivery(t *testing.T) {
	bus := events.NewEventBus()
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 4 {
			http.Error(w, "try again", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	dispatcher, err := NewDispatcher([]config.WebhookConfig{{
		URL:    server.URL,
		Events: []string{"deploy.failed"},
	}}, bus, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	dispatcher.retrySchedule = []time.Duration{10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dispatcher.Start(ctx)

	bus.Publish(events.Event{
		Type:       events.ArtifactStatusChanged,
		ArtifactID: "art-2",
		Status:     "failed",
		Timestamp:  time.Now().UTC(),
	})

	require.Eventually(t, func() bool { return attempts == 4 }, 2*time.Second, 20*time.Millisecond)
}

func TestDispatcherFiltersUnmatchedEvents(t *testing.T) {
	bus := events.NewEventBus()
	called := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called <- struct{}{}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	dispatcher, err := NewDispatcher([]config.WebhookConfig{{
		URL:    server.URL,
		Events: []string{"deploy.completed"},
	}}, bus, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dispatcher.Start(ctx)

	bus.Publish(events.Event{
		Type:       events.ArtifactStatusChanged,
		ArtifactID: "art-3",
		Status:     "building",
		Timestamp:  time.Now().UTC(),
	})

	select {
	case <-called:
		t.Fatal("received unexpected webhook delivery")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestProjectEventDeletion(t *testing.T) {
	payload, ok := projectEvent(events.Event{
		Type:         events.ArtifactDeleted,
		ArtifactID:   "art-4",
		ArtifactName: "demo",
		Target:       "kubernetes",
		URL:          "http://demo.local",
		Timestamp:    time.Now().UTC(),
	})
	require.True(t, ok)
	assert.Equal(t, "artifact.deleted", payload.Event)
	assert.Equal(t, "demo", payload.ArtifactName)
	assert.Equal(t, "kubernetes", payload.Target)
}

func verifySignature(header string, payload Payload, secret string) bool {
	body, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(header), []byte(expected))
}
