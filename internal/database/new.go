package database

import (
	"context"
	"fmt"
)

// New constructs a Repository from GOAPI_DB_DRIVER (default: postgres).
func New(ctx context.Context) (Repository, error) {
	switch ResolveDriver() {
	case DriverMock:
		return &mockRepo{}, nil
	case DriverPostgres:
		dsn, err := DSNFromEnv()
		if err != nil {
			return nil, err
		}
		return newPostgresRepo(ctx, dsn)
	default:
		return nil, fmt.Errorf("unknown GOAPI_DB_DRIVER %q", ResolveDriver())
	}
}
