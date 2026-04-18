package config

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestResolveLogLevelDefaultsToDebug(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("APP_ENV", "")

	level := ResolveLogLevel()
	if level != log.DebugLevel {
		t.Fatalf("expected debug level, got %s", level.String())
	}
}

func TestResolveLogLevelUsesInfoForHigherEnvironment(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("APP_ENV", "production")

	level := ResolveLogLevel()
	if level != log.InfoLevel {
		t.Fatalf("expected info level, got %s", level.String())
	}
}

func TestResolveLogLevelPrefersLogLevelOverride(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_LEVEL", "error")

	level := ResolveLogLevel()
	if level != log.ErrorLevel {
		t.Fatalf("expected error level, got %s", level.String())
	}
}

func TestResolveLogLevelIgnoresInvalidOverride(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_LEVEL", "not-a-level")

	level := ResolveLogLevel()
	if level != log.InfoLevel {
		t.Fatalf("expected info level, got %s", level.String())
	}
}

func TestResolveLogLevelTrimSpaces(t *testing.T) {
	_ = os.Setenv("APP_ENV", "  dev  ")
	_ = os.Setenv("LOG_LEVEL", "")
	t.Cleanup(func() {
		_ = os.Unsetenv("APP_ENV")
		_ = os.Unsetenv("LOG_LEVEL")
	})

	level := ResolveLogLevel()
	if level != log.DebugLevel {
		t.Fatalf("expected debug level, got %s", level.String())
	}
}

func TestConfigureLogOutputMissingDirectoryHandledGracefully(t *testing.T) {
	t.Setenv("LOG_FILE_PATH", "")
	ConfigureLogOutput()
}
