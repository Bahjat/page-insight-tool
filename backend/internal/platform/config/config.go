package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

var (
	errInvalidPort           = errors.New("config: invalid PORT number")
	errConcurrencyOutOfRange = errors.New("config: LINK_CHECK_CONCURRENCY must be 1-100")
	errInvalidShutdown       = errors.New("config: SHUTDOWN_TIMEOUT_SECONDS must be greater than 0")
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port                 string
	LogLevel             string
	LinkCheckConcurrency int
	ShutdownTimeout      time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (Config, error) {
	cfg := Config{
		Port:                 getEnv("PORT", "8080"),
		LogLevel:             getEnv("LOG_LEVEL", "ERROR"),
		LinkCheckConcurrency: getEnvAsInt("LINK_CHECK_CONCURRENCY", 25),
		ShutdownTimeout:      time.Duration(getEnvAsInt("SHUTDOWN_TIMEOUT_SECONDS", 10)) * time.Second,
	}

	return cfg, cfg.validate()
}

func (c Config) validate() error {
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%w: %q", errInvalidPort, c.Port)
	}

	if c.LinkCheckConcurrency < 1 || c.LinkCheckConcurrency > 100 {
		return fmt.Errorf("%w: got %d", errConcurrencyOutOfRange, c.LinkCheckConcurrency)
	}

	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("%w: got %s", errInvalidShutdown, c.ShutdownTimeout)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	s := os.Getenv(key)
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}
