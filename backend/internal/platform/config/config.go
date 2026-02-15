package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

var errInvalidPort = errors.New("invalid port number")

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port     string
	LogLevel string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (Config, error) {
	cfg := Config{
		Port:     getEnv("PORT", "8080"),
		LogLevel: getEnv("LOG_LEVEL", "ERROR"),
	}

	return cfg, cfg.validate()
}

func (c Config) validate() error {
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%w: %q", errInvalidPort, c.Port)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
