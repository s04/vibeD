package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/vibed-project/vibeD/internal/config"
	"github.com/vibed-project/vibeD/internal/events"
)

const defaultTimeout = 10 * time.Second

var defaultRetrySchedule = []time.Duration{1 * time.Second, 5 * time.Second, 25 * time.Second}

type webhookTarget struct {
	url     string
	events  []string
	secret  string
	timeout time.Duration
}

// Payload is the outbound webhook request body.
type Payload struct {
	Event        string            `json:"event"`
	ArtifactID   string            `json:"artifact_id"`
	ArtifactName string            `json:"artifact_name,omitempty"`
	Target       string            `json:"target,omitempty"`
	URL          string            `json:"url,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Dispatcher fans out lifecycle events from the in-memory EventBus to configured HTTP endpoints.
type Dispatcher struct {
	bus           *events.EventBus
	client        *http.Client
	logger        *slog.Logger
	targets       []webhookTarget
	retrySchedule []time.Duration
}

func NewDispatcher(cfgs []config.WebhookConfig, bus *events.EventBus, logger *slog.Logger) (*Dispatcher, error) {
	targets := make([]webhookTarget, 0, len(cfgs))
	for i, cfg := range cfgs {
		timeout := defaultTimeout
		if cfg.Timeout != "" {
			parsed, err := time.ParseDuration(cfg.Timeout)
			if err != nil {
				return nil, fmt.Errorf("parse webhooks[%d].timeout: %w", i, err)
			}
			timeout = parsed
		}

		secret, err := config.ResolveSecret(cfg.Secret)
		if err != nil {
			return nil, fmt.Errorf("resolve webhooks[%d].secret: %w", i, err)
		}

		targets = append(targets, webhookTarget{
			url:     cfg.URL,
			events:  append([]string(nil), cfg.Events...),
			secret:  secret,
			timeout: timeout,
		})
	}

	return &Dispatcher{
		bus:           bus,
		client:        &http.Client{},
		logger:        logger,
		targets:       targets,
		retrySchedule: append([]time.Duration(nil), defaultRetrySchedule...),
	}, nil
}

func (d *Dispatcher) Start(ctx context.Context) {
	if d == nil || d.bus == nil || len(d.targets) == 0 {
		return
	}

	ch, unsub := d.bus.Subscribe(ctx)
	go func() {
		defer unsub()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-ch:
				if !ok {
					return
				}
				payload, ok := projectEvent(event)
				if !ok {
					continue
				}
				for _, target := range d.targets {
					if !matchesEvent(target.events, payload.Event) {
						continue
					}
					go d.deliverWithRetry(ctx, target, payload)
				}
			}
		}
	}()
}

func projectEvent(event events.Event) (Payload, bool) {
	payload := Payload{
		ArtifactID:   event.ArtifactID,
		ArtifactName: event.ArtifactName,
		Target:       event.Target,
		URL:          event.URL,
		Timestamp:    event.Timestamp,
		Metadata:     map[string]string{"raw_event": string(event.Type)},
	}

	if event.OwnerID != "" {
		payload.Metadata["owner_id"] = event.OwnerID
	}
	if event.Status != "" {
		payload.Metadata["status"] = event.Status
	}
	if event.Error != "" {
		payload.Metadata["error"] = event.Error
	}

	switch event.Type {
	case events.ArtifactStatusChanged:
		switch event.Status {
		case "running":
			payload.Event = "deploy.completed"
		case "failed":
			payload.Event = "deploy.failed"
		default:
			if event.Status == "" {
				return Payload{}, false
			}
			payload.Event = "deploy." + event.Status
		}
	case events.ArtifactDeleted:
		payload.Event = "artifact.deleted"
	default:
		return Payload{}, false
	}

	return payload, true
}

func matchesEvent(filters []string, event string) bool {
	for _, filter := range filters {
		if filter == "*" || filter == event {
			return true
		}
	}
	return false
}

func (d *Dispatcher) deliverWithRetry(ctx context.Context, target webhookTarget, payload Payload) {
	deliveryID := newDeliveryID()
	for attempt := 0; ; attempt++ {
		err := d.deliver(ctx, target, payload, deliveryID)
		if err == nil {
			return
		}

		if attempt >= len(d.retrySchedule) {
			d.logger.Warn("webhook delivery failed", "url", target.url, "event", payload.Event, "delivery_id", deliveryID, "error", err)
			return
		}

		delay := d.retrySchedule[attempt]
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (d *Dispatcher) deliver(ctx context.Context, target webhookTarget, payload Payload, deliveryID string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, target.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, target.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-VibeD-Event", payload.Event)
	req.Header.Set("X-VibeD-Delivery", deliveryID)
	if target.secret != "" {
		req.Header.Set("X-VibeD-Signature", signature(body, target.secret))
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

func signature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func newDeliveryID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("delivery-%d", time.Now().UnixNano())
	}
	return strings.ToLower(hex.EncodeToString(buf[:]))
}
