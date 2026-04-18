package database

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Driver values for GOAPI_DB_DRIVER.
const (
	DriverPostgres = "postgres"
	DriverMock     = "mock"
)

// ResolveDriver returns GOAPI_DB_DRIVER; default is postgres.
func ResolveDriver() string {
	d := strings.ToLower(strings.TrimSpace(os.Getenv("GOAPI_DB_DRIVER")))
	if d == "" {
		return DriverPostgres
	}
	return d
}

// DSNFromEnv builds a postgres connection string.
// If DATABASE_URL is set, it is used as-is.
// Otherwise: POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_SSLMODE.
func DSNFromEnv() (string, error) {
	if u := strings.TrimSpace(os.Getenv("DATABASE_URL")); u != "" {
		return u, nil
	}

	user := firstNonEmpty(os.Getenv("POSTGRES_USER"), "goapi")
	pass := os.Getenv("POSTGRES_PASSWORD")
	host := firstNonEmpty(os.Getenv("POSTGRES_HOST"), "localhost")
	port := firstNonEmpty(os.Getenv("POSTGRES_PORT"), "5432")
	dbname := firstNonEmpty(os.Getenv("POSTGRES_DB"), "goapi")
	ssl := firstNonEmpty(os.Getenv("POSTGRES_SSLMODE"), "disable")

	// URL-encode password for special characters in DSN.
	pu := url.UserPassword(user, pass)
	if pass == "" {
		pu = url.User(user)
	}
	u := &url.URL{
		Scheme:   "postgres",
		User:     pu,
		Host:     fmt.Sprintf("%s:%s", host, port),
		Path:     "/" + dbname,
		RawQuery: "sslmode=" + url.QueryEscape(ssl),
	}
	return u.String(), nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
