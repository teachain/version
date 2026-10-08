package config

import (
	"testing"
	"time"
)

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("DB_DSN", "xorm:mysql://u:p@tcp(h:3306)/d")
	t.Setenv("GITHUB_TOKEN", "ghp_x")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("POLL_INTERVAL", "5m")
	t.Setenv("POLL_CONCURRENCY", "7")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DBDSN != "xorm:mysql://u:p@tcp(h:3306)/d" {
		t.Fail()
	}
	if cfg.GitHubToken != "ghp_x" {
		t.Fail()
	}
	if cfg.HTTPPort != 9090 {
		t.Fail()
	}
	if cfg.PollInterval != 5*time.Minute {
		t.Fail()
	}
	if cfg.PollConcurrency != 7 {
		t.Fail()
	}
	if cfg.LogLevel != "debug" {
		t.Fail()
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DB_DSN", "x")
	for _, k := range []string{"HTTP_PORT", "POLL_INTERVAL", "POLL_CONCURRENCY", "LOG_LEVEL"} {
		t.Setenv(k, "")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPPort != 8080 {
		t.Fail()
	}
	if cfg.PollInterval != 10*time.Minute {
		t.Fail()
	}
	if cfg.PollConcurrency != 10 {
		t.Fail()
	}
	if cfg.LogLevel != "info" {
		t.Fail()
	}
}

func TestLoad_MissingDBDSN(t *testing.T) {
	t.Setenv("DB_DSN", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DB_DSN missing")
	}
}