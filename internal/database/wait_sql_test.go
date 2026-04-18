package database

import (
	"context"
	"testing"
)

func TestWaitForPostgresSQL_ContextCancelledBeforeDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitForPostgresSQL(ctx, "postgres://u:p@127.0.0.1:59999/db?sslmode=disable")
	if err == nil {
		t.Fatal("expected error when context already cancelled")
	}
}
