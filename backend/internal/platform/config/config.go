package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

var (
	errInvalidPort           = errors.New("config: invalid PORT number")
	errConcurrencyOutOfRange = errors.New("config: LINK_CHECK_CONCURRENCY must be 1-100")
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port                 string
	LogLevel             string
	LinkCheckConcurrency int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (Config, error) {
	cfg := Config{
		Port:                 getEnv("PORT", "8080"),
		LogLevel:             getEnv("LOG_LEVEL", "ERROR"),
		LinkCheckConcurrency: getEnvAsInt("LINK_CHECK_CONCURRENCY", 10),
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
