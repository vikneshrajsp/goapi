package database

import (
	"strings"
	"testing"
)

func TestDSNFromEnvDATABASE_URL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x:y@localhost:5432/db?sslmode=disable")
	t.Setenv("POSTGRES_HOST", "ignored")

	dsn, err := DSNFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dsn, "postgres://x:y@localhost:5432/db") {
		t.Fatalf("unexpected dsn: %s", dsn)
	}
}

func TestResolveDriverMockTrimmed(t *testing.T) {
	t.Setenv("GOAPI_DB_DRIVER", " MOCK ")
	if got := ResolveDriver(); got != DriverMock {
		t.Fatalf("expected mock driver, got %q", got)
	}
}

func TestResolveDriverPostgresCaseInsensitive(t *testing.T) {
	t.Setenv("GOAPI_DB_DRIVER", "PoStGrEs")
	if got := ResolveDriver(); got != DriverPostgres {
		t.Fatalf("expected postgres driver, got %q", got)
	}
}

func TestDSNFromEnvDiscrete(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("POSTGRES_USER", "u")
	t.Setenv("POSTGRES_PASSWORD", "p@ss")
	t.Setenv("POSTGRES_HOST", "h")
	t.Setenv("POSTGRES_PORT", "55432")
	t.Setenv("POSTGRES_DB", "d")

	dsn, err := DSNFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(dsn, "postgres://") {
		t.Fatalf("unexpected dsn prefix: %s", dsn)
	}
	if !strings.Contains(dsn, "@h:55432/") || !strings.Contains(dsn, "sslmode=") {
		t.Fatalf("unexpected dsn: %s", dsn)
	}
}
