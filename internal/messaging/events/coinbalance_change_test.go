package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCoinBalanceChangedTypeConstant(t *testing.T) {
	if CoinBalanceChangedType != "coinbalance_change" {
		t.Fatalf("got %q", CoinBalanceChangedType)
	}
}

func TestCoinBalanceChangedJSONRoundTrip(t *testing.T) {
	at := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	orig := CoinBalanceChanged{
		SchemaVersion: 1,
		EventID:       "evt-1",
		EventType:     CoinBalanceChangedType,
		Username:      "alex",
		Previous:      10,
		Current:       20,
		Delta:         10,
		OccurredAt:    at,
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out CoinBalanceChanged
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out != orig {
		t.Fatalf("round trip mismatch: %+v vs %+v", out, orig)
	}
}
