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

func TestNewUnknownDriver(t *testing.T) {
	t.Setenv("GOAPI_DB_DRIVER", "sqlite")
	t.Setenv("DATABASE_URL", "")

	if _, err := New(context.Background()); err == nil {
		t.Fatal("expected error for unknown GOAPI_DB_DRIVER")
	}
}
