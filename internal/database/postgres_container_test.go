//go:build testcontainers

package database

import (
	"context"
	"errors"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresRepositoryAgainstContainer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("goapi"),
		postgres.WithUsername("goapi"),
		postgres.WithPassword("goapi"),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	t.Setenv("GOAPI_DB_DRIVER", DriverPostgres)
	t.Setenv("DATABASE_URL", connStr)

	repo, err := New(ctx)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ld, err := repo.GetLoginDetails(ctx, "alex")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if ld.AuthToken != "123AL100" {
		t.Fatalf("unexpected token")
	}

	cd, err := repo.GetCoinDetails(ctx, "alex")
	if err != nil || cd.Coins != 100 {
		t.Fatalf("coins: %+v err=%v", cd, err)
	}

	upd, err := repo.UpdateCoinDetails(ctx, "alex", 42)
	if err != nil || upd.Coins != 42 {
		t.Fatalf("update: %+v err=%v", upd, err)
	}

	if _, err := repo.UpdateCoinDetails(ctx, "alex", 100); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.GetLoginDetails(ctx, "ghost"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound for login, got %v", err)
	}
	if _, err := repo.GetCoinDetails(ctx, "ghost"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound for coins, got %v", err)
	}
	if _, err := repo.UpdateCoinDetails(ctx, "ghost", 1); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound for update unknown user, got %v", err)
	}
	if _, err := repo.UpdateCoinDetails(ctx, "alex", -1); err == nil {
		t.Fatal("expected error for negative balance")
	}

	if pr, ok := repo.(*postgresRepo); ok {
		if err := pr.Setup(ctx); err != nil {
			t.Fatalf("Setup: %v", err)
		}
		pr.Close()
	}
}
