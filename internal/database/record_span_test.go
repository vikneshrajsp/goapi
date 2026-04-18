package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
)

func TestRecordSpanErrBranches(t *testing.T) {
	tr := otel.GetTracerProvider().Tracer("test")
	_, span := tr.Start(context.Background(), "record_span_test")
	defer span.End()

	recordSpanErr(span, nil)
	recordSpanErr(span, ErrUserNotFound)
	recordSpanErr(span, pgx.ErrNoRows)
	recordSpanErr(span, errors.New("simulated scan failure"))
}
