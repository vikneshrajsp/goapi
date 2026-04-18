package middleware

import (
	"context"
	"testing"

	"goapi/internal/database"
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
