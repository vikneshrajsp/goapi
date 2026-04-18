package tools

import (
	log "github.com/sirupsen/logrus"
)

func ConnectDatabase() {
	log.Info("Connecting to database")
}

type LoginDetails struct {
	Username  string `json:"username"`
	AuthToken string `json:"auth_token"`
}

type CoinDetails struct {
	Username string `json:"username"`
	Coins    int64  `json:"balance"`
}

type DatabaseInterface interface {
	SetupDatabase() error
	GetLoginDetails(username string) (*LoginDetails, error)
	GetCoinDetails(username string) (*CoinDetails, error)
	UpdateCoinDetails(username string, balance int64) (*CoinDetails, error)
}

func NewDatabase() (DatabaseInterface, error) {
	var database DatabaseInterface = &mockDB{}

	error := database.SetupDatabase()
	if error != nil {
		log.Error(error)
		return nil, error
	}
	return database, nil
}
