package database

import (
	"context"
	"testing"
)

func TestMockGetLoginDetails(t *testing.T) {
	m := &mockRepo{}
	login, err := m.GetLoginDetails(context.Background(), "alex")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if login.AuthToken != "123AL100" {
		t.Fatalf("expected token, got %s", login.AuthToken)
	}
}

func TestMockGetCoinDetails(t *testing.T) {
	m := &mockRepo{}
	coin, err := m.GetCoinDetails(context.Background(), "max")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if coin.Coins != 130 {
		t.Fatalf("expected 130, got %d", coin.Coins)
	}
}

func TestMockUpdateAndUnknownUser(t *testing.T) {
	m := &mockRepo{}
	orig, err := m.GetCoinDetails(context.Background(), "alex")
	if err != nil {
		t.Fatal(err)
	}

	updated, err := m.UpdateCoinDetails(context.Background(), "alex", 999)
	if err != nil || updated.Coins != 999 {
		t.Fatalf("update failed: %v", err)
	}

	_, err = m.UpdateCoinDetails(context.Background(), "alex", orig.Coins)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.UpdateCoinDetails(context.Background(), "alex", -1); err == nil {
		t.Fatal("expected error for negative balance")
	}

	if _, err := m.GetLoginDetails(context.Background(), "nope"); err == nil {
		t.Fatal("expected error")
	}
}
