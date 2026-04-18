package database

import (
	"context"
	"testing"
)

func TestNewMockDriver(t *testing.T) {
	t.Setenv("GOAPI_DB_DRIVER", DriverMock)
	t.Setenv("DATABASE_URL", "")

	repo, err := New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.(*mockRepo); !ok {
		t.Fatalf("expected mock repo")
	}
}
