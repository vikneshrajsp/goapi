package events

import "time"

const CoinBalanceChangedType = "coinbalance_change"

type CoinBalanceChanged struct {
	SchemaVersion int       `json:"schema_version"`
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	Username      string    `json:"username"`
	Previous      int64     `json:"previous_balance"`
	Current       int64     `json:"current_balance"`
	Delta         int64     `json:"delta"`
	OccurredAt    time.Time `json:"occurred_at"`
}
