package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/segmentio/kafka-go"
	"goapi/internal/database"
	"goapi/internal/messaging/events"
)

func setupRepo(t *testing.T) database.Repository {
	t.Helper()
	t.Setenv("GOAPI_DB_DRIVER", database.DriverMock)
	repo, err := database.New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func eventBytes(t *testing.T) []byte {
	t.Helper()
	event := events.CoinBalanceChanged{
		SchemaVersion: 1,
		EventID:       "e-1",
		EventType:     events.CoinBalanceChangedType,
		Username:      "alex",
		Previous:      100,
		Current:       120,
		Delta:         20,
	}
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestHandleSendsWebhook(t *testing.T) {
	repo := setupRepo(t)
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := repo.SetUserWebhookURL(context.Background(), "alex", srv.URL); err != nil {
		t.Fatal(err)
	}

	worker := New(repo)
	if err := worker.Handle(context.Background(), kafka.Message{Value: eventBytes(t)}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if !hit {
		t.Fatal("expected webhook hit")
	}
}

func TestHandleInvalidPayload(t *testing.T) {
	worker := New(setupRepo(t))
	if err := worker.Handle(context.Background(), kafka.Message{Value: []byte("{")}); err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestHandleNoWebhookConfigured(t *testing.T) {
	worker := New(setupRepo(t))
	event := events.CoinBalanceChanged{
		SchemaVersion: 1,
		EventID:       "e-2",
		EventType:     events.CoinBalanceChangedType,
		Username:      "kevin",
		Previous:      100,
		Current:       105,
		Delta:         5,
	}
	body, _ := json.Marshal(event)
	if err := worker.Handle(context.Background(), kafka.Message{Value: body}); err != nil {
		t.Fatalf("expected skipped webhook without error, got %v", err)
	}
}

func TestHandleWebhookNon2xx(t *testing.T) {
	repo := setupRepo(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	if err := repo.SetUserWebhookURL(context.Background(), "alex", srv.URL); err != nil {
		t.Fatal(err)
	}
	worker := New(repo)
	if err := worker.Handle(context.Background(), kafka.Message{Value: eventBytes(t)}); err != nil {
		t.Fatalf("expected non-fatal non-2xx webhook handling, got %v", err)
	}
}
