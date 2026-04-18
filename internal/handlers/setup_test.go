package handlers

import (
	"context"
	"testing"

	"goapi/internal/database"
	"goapi/internal/messaging/kafka"
)

func mockRepo(t *testing.T) database.Repository {
	t.Helper()
	t.Setenv("GOAPI_DB_DRIVER", database.DriverMock)
	repo, err := database.New(context.Background())
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	return repo
}

func testDeps() Deps {
	return Deps{
		Publisher:   kafka.NewNoopProducer(),
		KafkaHealth: kafka.NewNoopProducer(),
	}
}
