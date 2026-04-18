package tools

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestMockDBGetLoginDetails(t *testing.T) {
	db := &mockDB{}

	login, err := db.GetLoginDetails("alex")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if login.AuthToken != "123AL100" {
		t.Fatalf("expected token 123AL100, got %s", login.AuthToken)
	}
}

func TestMockDBGetCoinDetails(t *testing.T) {
	db := &mockDB{}

	coin, err := db.GetCoinDetails("max")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if coin.Coins != 130 {
		t.Fatalf("expected 130 coins, got %d", coin.Coins)
	}
}

func TestMockDBUpdateCoinDetails(t *testing.T) {
	db := &mockDB{}

	original, err := db.GetCoinDetails("alex")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, err := db.UpdateCoinDetails("alex", 999)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Coins != 999 {
		t.Fatalf("expected 999 coins, got %d", updated.Coins)
	}

	_, err = db.UpdateCoinDetails("alex", original.Coins)
	if err != nil {
		t.Fatalf("failed to restore original balance: %v", err)
	}
}

func TestMockDBUpdateCoinDetailsValidation(t *testing.T) {
	db := &mockDB{}

	if _, err := db.UpdateCoinDetails("alex", -1); err == nil {
		t.Fatal("expected error for negative balance")
	}
}

func TestNewDatabase(t *testing.T) {
	db, err := NewDatabase()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if db == nil {
		t.Fatal("expected database instance, got nil")
	}
}

func TestConnectDatabase(t *testing.T) {
	previousLevel := log.GetLevel()
	log.SetLevel(log.InfoLevel)
	t.Cleanup(func() { log.SetLevel(previousLevel) })

	ConnectDatabase()
}

func TestMockDBGetLoginDetailsUnknownUser(t *testing.T) {
	db := &mockDB{}
	if _, err := db.GetLoginDetails("unknown"); err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestMockDBGetCoinDetailsUnknownUser(t *testing.T) {
	db := &mockDB{}
	if _, err := db.GetCoinDetails("unknown"); err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestMockDBUpdateCoinDetailsUnknownUser(t *testing.T) {
	db := &mockDB{}
	if _, err := db.UpdateCoinDetails("unknown", 10); err == nil {
		t.Fatal("expected error for unknown user")
	}
}
