package handlers

import (
	"context"

	"goapi/internal/messaging/events"
)

type EventPublisher interface {
	PublishCoinBalanceChanged(context.Context, events.CoinBalanceChanged) error
}

type HealthChecker interface {
	HealthCheck(context.Context) error
}

type Deps struct {
	Publisher   EventPublisher
	KafkaHealth HealthChecker
}
