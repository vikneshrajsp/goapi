package tools

import (
	"errors"
	"sync"
)

type mockDB struct{}

var mockCoinDetailsMu sync.RWMutex

var mockLoginDetails = map[string]LoginDetails{
	"alex": {
		Username:  "alex",
		AuthToken: "123AL100",
	},
	"kevin": {
		Username:  "kevin",
		AuthToken: "456KV100",
	},
	"max": {
		Username:  "max",
		AuthToken: "789MX100",
	},
}

var mockCoinDetails = map[string]CoinDetails{
	"alex": {
		Username: "alex",
		Coins:    100,
	},
	"kevin": {
		Username: "kevin",
		Coins:    120,
	},
	"max": {
		Username: "max",
		Coins:    130,
	},
}

func (m *mockDB) SetupDatabase() error {
	return nil
}

func (m *mockDB) GetLoginDetails(username string) (*LoginDetails, error) {
	if _, ok := mockLoginDetails[username]; !ok {
		return &LoginDetails{}, errors.New("user not found")
	}
	d := mockLoginDetails[username]
	return &d, nil
}

func (m *mockDB) GetCoinDetails(username string) (*CoinDetails, error) {
	mockCoinDetailsMu.RLock()
	defer mockCoinDetailsMu.RUnlock()

	if _, ok := mockCoinDetails[username]; !ok {
		return &CoinDetails{}, errors.New("user not found")
	}
	d := mockCoinDetails[username]
	return &d, nil
}

func (m *mockDB) UpdateCoinDetails(username string, balance int64) (*CoinDetails, error) {
	if balance < 0 {
		return &CoinDetails{}, errors.New("balance cannot be negative")
	}

	mockCoinDetailsMu.Lock()
	defer mockCoinDetailsMu.Unlock()

	if _, ok := mockCoinDetails[username]; !ok {
		return &CoinDetails{}, errors.New("user not found")
	}

	mockCoinDetails[username] = CoinDetails{
		Username: username,
		Coins:    balance,
	}

	d := mockCoinDetails[username]
	return &d, nil
}
