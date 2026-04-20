package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"goapi/internal/database"
	"goapi/internal/messaging/events"
)

type Worker struct {
	repo   database.Repository
	client *http.Client
}

func New(repo database.Repository) *Worker {
	return &Worker{
		repo: repo,
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (w *Worker) Handle(ctx context.Context, msg kafka.Message) error {
	var event events.CoinBalanceChanged
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}
	log.WithFields(log.Fields{
		"timestamp":        event.OccurredAt.Format(time.RFC3339Nano),
		"username":         event.Username,
		"previous_balance": event.Previous,
		"current_balance":  event.Current,
		"delta":            event.Delta,
		"event_id":         event.EventID,
	}).Info("coin balance changed event consumed")

	webhookURL, err := w.repo.GetUserWebhookURL(ctx, event.Username)
	if err != nil {
		log.WithFields(log.Fields{
			"username": event.Username,
			"event_id": event.EventID,
		}).Warnf("no webhook registered, skipping notify: %v", err)
		return nil
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"username": event.Username,
			"event_id": event.EventID,
		}).Warnf("webhook request failed: %v", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		log.WithFields(log.Fields{
			"username": event.Username,
			"event_id": event.EventID,
			"status":   resp.StatusCode,
		}).Warn("webhook endpoint returned non-success status")
		return nil
	}

	log.WithField("event_id", event.EventID).Info("webhook notified for coin balance change")

	return nil
}
