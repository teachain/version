package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBDSN           string
	GitHubToken     string
	HTTPPort        int
	PollInterval    time.Duration
	PollConcurrency int
	LogLevel        string
}

func Load() (*Config, error) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("DB_DSN is required")
	}
	port, err := atoiOr("HTTP_PORT", 8080)
	if err != nil {
		return nil, err
	}
	dur, err := durationOr("POLL_INTERVAL", 10*time.Minute)
	if err != nil {
		return nil, err
	}
	conc, err := atoiOr("POLL_CONCURRENCY", 10)
	if err != nil {
		return nil, err
	}
	lvl := os.Getenv("LOG_LEVEL")
	if lvl == "" {
		lvl = "info"
	}
	return &Config{
		DBDSN:           dsn,
		GitHubToken:     os.Getenv("GITHUB_TOKEN"),
		HTTPPort:        port,
		PollInterval:    dur,
		PollConcurrency: conc,
		LogLevel:        lvl,
	}, nil
}

func atoiOr(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s invalid: %w", key, err)
	}
	return n, nil
}

func durationOr(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s invalid: %w", key, err)
	}
	return d, nil
}