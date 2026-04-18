package database

import (
	"context"
	"errors"
)

var ErrUserNotFound = errors.New("user not found")

type LoginDetails struct {
	Username  string `json:"username"`
	AuthToken string `json:"auth_token"`
}

type CoinDetails struct {
	Username string `json:"username"`
	Coins    int64  `json:"balance"`
}

// Repository defines persistence for auth and coin balances.
type Repository interface {
	Setup(context.Context) error
	GetLoginDetails(context.Context, string) (*LoginDetails, error)
	GetCoinDetails(context.Context, string) (*CoinDetails, error)
	UpdateCoinDetails(context.Context, string, int64) (*CoinDetails, error)
}
