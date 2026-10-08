package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DBDSN           string        `yaml:"db_dsn"`
	GitHubToken     string        `yaml:"github_token"`
	HTTPPort        int           `yaml:"http_port"`
	PollInterval    time.Duration `yaml:"poll_interval"`
	PollConcurrency int           `yaml:"poll_concurrency"`
	LogLevel        string        `yaml:"log_level"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	applyDefaults(&cfg)

	if cfg.DBDSN == "" {
		return nil, fmt.Errorf("db_dsn is required")
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = 8080
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 10 * time.Minute
	}
	if cfg.PollConcurrency == 0 {
		cfg.PollConcurrency = 10
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
}