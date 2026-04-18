package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func ResolveLogLevel() log.Level {
	if value := strings.TrimSpace(os.Getenv("LOG_LEVEL")); value != "" {
		level, err := log.ParseLevel(strings.ToLower(value))
		if err == nil {
			return level
		}
	}

	environment := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if environment == "" || environment == "development" || environment == "dev" || environment == "local" {
		return log.DebugLevel
	}

	return log.InfoLevel
}

func ConfigureLogOutput() {
	logFilePath := strings.TrimSpace(os.Getenv("LOG_FILE_PATH"))
	if logFilePath == "" {
		return
	}

	if err := os.MkdirAll(filepath.Dir(logFilePath), 0o755); err != nil {
		log.Warnf("failed to create log directory: %v", err)
		return
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Warnf("failed to open log file: %v", err)
		return
	}

	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
}
