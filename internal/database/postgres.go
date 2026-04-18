package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	otrace "go.opentelemetry.io/otel/trace"
)

const (
	dbOpTimeout = 8 * time.Second
)

type postgresRepo struct {
	pool *pgxpool.Pool

	peerHost string
	peerPort int
}

var pgTracer = otel.Tracer("goapi/postgres")

func (p *postgresRepo) startSQLSpan(ctx context.Context, operationName, stmt string) (context.Context, otrace.Span) {
	attrs := []attribute.KeyValue{
		semconv.DBSystemPostgreSQL,
		attribute.String("db.statement", stmt),
		semconv.PeerService("postgresql"),
		semconv.ServerAddress(p.peerHost),
	}
	if p.peerPort > 0 {
		attrs = append(attrs, semconv.ServerPort(p.peerPort))
	}

	return pgTracer.Start(ctx, operationName,
		otrace.WithSpanKind(otrace.SpanKindClient),
		otrace.WithAttributes(attrs...),
	)
}

func recordSpanErr(span otrace.Span, err error) {
	switch {
	case err == nil:
	case errors.Is(err, ErrUserNotFound):
	case errors.Is(err, pgx.ErrNoRows):
	default:
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

func newPostgresRepo(ctx context.Context, dsn string) (*postgresRepo, error) {
	waitCtx, cancelWait := context.WithTimeout(ctx, 60*time.Second)
	defer cancelWait()
	if err := waitForPostgresSQL(waitCtx, dsn); err != nil {
		return nil, fmt.Errorf("wait postgres: %w", err)
	}

	if err := migrateUp(ctx, dsn); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx pool config: %w", err)
	}

	peerHost := cfg.ConnConfig.Host
	if peerHost == "" {
		peerHost = "localhost"
	}
	peerPort := int(cfg.ConnConfig.Port)
	if peerPort == 0 {
		peerPort = 5432
	}

	cfg.MaxConns = 25
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgx ping: %w", err)
	}

	return &postgresRepo{pool: pool, peerHost: peerHost, peerPort: peerPort}, nil
}

// Close releases the pool.
func (p *postgresRepo) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

func (p *postgresRepo) Setup(context.Context) error {
	return nil
}

func (p *postgresRepo) GetLoginDetails(ctx context.Context, username string) (*LoginDetails, error) {
	ctx, cancel := context.WithTimeout(ctx, dbOpTimeout)
	defer cancel()

	const stmt = `SELECT username, auth_token FROM users WHERE username = $1`
	ctx, span := p.startSQLSpan(ctx, "postgres.query users SELECT", stmt)
	defer span.End()

	row := p.pool.QueryRow(ctx, stmt, username)

	var ld LoginDetails
	if err := row.Scan(&ld.Username, &ld.AuthToken); err != nil {
		recordSpanErr(span, err)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get login: %w", err)
	}
	return &ld, nil
}

func (p *postgresRepo) GetCoinDetails(ctx context.Context, username string) (*CoinDetails, error) {
	ctx, cancel := context.WithTimeout(ctx, dbOpTimeout)
	defer cancel()

	const stmt = `SELECT username, balance FROM coin_balances WHERE username = $1`
	ctx, span := p.startSQLSpan(ctx, "postgres.query coin_balances SELECT", stmt)
	defer span.End()

	row := p.pool.QueryRow(ctx, stmt, username)

	var cd CoinDetails
	if err := row.Scan(&cd.Username, &cd.Coins); err != nil {
		recordSpanErr(span, err)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get coins: %w", err)
	}
	return &cd, nil
}

func (p *postgresRepo) UpdateCoinDetails(ctx context.Context, username string, balance int64) (*CoinDetails, error) {
	if balance < 0 {
		return nil, fmt.Errorf("balance cannot be negative")
	}

	ctx, cancel := context.WithTimeout(ctx, dbOpTimeout)
	defer cancel()

	const stmt = `UPDATE coin_balances SET balance = $1 WHERE username = $2 RETURNING username, balance`
	ctx, span := p.startSQLSpan(ctx, "postgres.query coin_balances UPDATE", stmt)
	defer span.End()

	row := p.pool.QueryRow(ctx, stmt, balance, username)

	var cd CoinDetails
	if err := row.Scan(&cd.Username, &cd.Coins); err != nil {
		recordSpanErr(span, err)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("update coins: %w", err)
	}
	return &cd, nil
}

func (p *postgresRepo) SetUserWebhookURL(ctx context.Context, username string, webhookURL string) error {
	ctx, cancel := context.WithTimeout(ctx, dbOpTimeout)
	defer cancel()

	const stmt = `INSERT INTO user_webhooks (username, webhook_url)
VALUES ($1, $2)
ON CONFLICT (username) DO UPDATE SET webhook_url = EXCLUDED.webhook_url, updated_at = NOW()`
	ctx, span := p.startSQLSpan(ctx, "postgres.query user_webhooks UPSERT", stmt)
	defer span.End()

	tag, err := p.pool.Exec(ctx, stmt, username, webhookURL)
	if err != nil {
		recordSpanErr(span, err)
		return fmt.Errorf("set webhook url: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (p *postgresRepo) GetUserWebhookURL(ctx context.Context, username string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, dbOpTimeout)
	defer cancel()

	const stmt = `SELECT webhook_url FROM user_webhooks WHERE username = $1`
	ctx, span := p.startSQLSpan(ctx, "postgres.query user_webhooks SELECT", stmt)
	defer span.End()

	var webhookURL string
	if err := p.pool.QueryRow(ctx, stmt, username).Scan(&webhookURL); err != nil {
		recordSpanErr(span, err)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrUserNotFound
		}
		return "", fmt.Errorf("get webhook url: %w", err)
	}
	return webhookURL, nil
}
