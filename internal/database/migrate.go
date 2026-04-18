package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

func waitForPostgresSQL(ctx context.Context, databaseURL string) error {
	const tick = 200 * time.Millisecond
	tkr := time.NewTicker(tick)
	defer tkr.Stop()

	var lastErr error
	for {
		db, err := sql.Open("pgx", databaseURL)
		if err != nil {
			lastErr = err
		} else {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err = db.PingContext(pingCtx)
			cancel()
			closeErr := db.Close()
			if err == nil && closeErr == nil {
				return nil
			}
			if err != nil {
				lastErr = err
			} else {
				lastErr = closeErr
			}
		}

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("%w (last error: %v)", ctx.Err(), lastErr)
			}
			return ctx.Err()
		case <-tkr.C:
		}
	}
}

func migrateUp(_ context.Context, databaseURL string) error {
	sourceDriver, err := iofs.New(embeddedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("migrate embedded source: %w", err)
	}

	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("sql open for migrate: %w", err)
	}

	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{})
	if err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("migrate postgres driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("migrate new: %w", err)
	}

	upErr := m.Up()
	if upErr != nil && !errors.Is(upErr, migrate.ErrNoChange) {
		_, _ = m.Close()
		_ = sqlDB.Close()
		return fmt.Errorf("migrate up: %w", upErr)
	}

	srcErr, dbErr := m.Close()
	sqlCloseErr := sqlDB.Close()
	return errors.Join(srcErr, dbErr, sqlCloseErr)
}
