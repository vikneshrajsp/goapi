package database

import (
	"context"
	"errors"
	"sync"
)

type mockRepo struct{}

var mockLoginDetails = map[string]LoginDetails{
	"alex": {Username: "alex", AuthToken: "123AL100"},
	"kevin": {Username: "kevin", AuthToken: "456KV100"},
	"max":   {Username: "max", AuthToken: "789MX100"},
}

var mockCoinDetailsMu sync.RWMutex

var mockCoinDetails = map[string]CoinDetails{
	"alex": {Username: "alex", Coins: 100},
	"kevin": {Username: "kevin", Coins: 120},
	"max":   {Username: "max", Coins: 130},
}

func (m *mockRepo) Setup(context.Context) error {
	return nil
}

func (m *mockRepo) GetLoginDetails(_ context.Context, username string) (*LoginDetails, error) {
	if _, ok := mockLoginDetails[username]; !ok {
		return nil, ErrUserNotFound
	}
	d := mockLoginDetails[username]
	return &d, nil
}

func (m *mockRepo) GetCoinDetails(_ context.Context, username string) (*CoinDetails, error) {
	mockCoinDetailsMu.RLock()
	defer mockCoinDetailsMu.RUnlock()

	if _, ok := mockCoinDetails[username]; !ok {
		return nil, ErrUserNotFound
	}
	d := mockCoinDetails[username]
	return &d, nil
}

func (m *mockRepo) UpdateCoinDetails(_ context.Context, username string, balance int64) (*CoinDetails, error) {
	if balance < 0 {
		return nil, errors.New("balance cannot be negative")
	}

	mockCoinDetailsMu.Lock()
	defer mockCoinDetailsMu.Unlock()

	if _, ok := mockCoinDetails[username]; !ok {
		return nil, ErrUserNotFound
	}

	mockCoinDetails[username] = CoinDetails{Username: username, Coins: balance}
	d := mockCoinDetails[username]
	return &d, nil
}
