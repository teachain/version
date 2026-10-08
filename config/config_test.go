package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoad_AllFieldsSet(t *testing.T) {
	path := writeYAML(t, `
db_dsn: "root:root@tcp(host:3306)/teachain"
github_token: "ghp_x"
http_port: 9090
poll_interval: 5m
poll_concurrency: 7
log_level: debug
`)
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.DBDSN != "root:root@tcp(host:3306)/teachain" {
		t.Errorf("DBDSN = %q", got.DBDSN)
	}
	if got.GitHubToken != "ghp_x" {
		t.Errorf("GitHubToken = %q", got.GitHubToken)
	}
	if got.HTTPPort != 9090 {
		t.Errorf("HTTPPort = %d", got.HTTPPort)
	}
	if got.PollInterval != 5*time.Minute {
		t.Errorf("PollInterval = %v", got.PollInterval)
	}
	if got.PollConcurrency != 7 {
		t.Errorf("PollConcurrency = %d", got.PollConcurrency)
	}
	if got.LogLevel != "debug" {
		t.Errorf("LogLevel = %q", got.LogLevel)
	}
}

func TestLoad_Defaults(t *testing.T) {
	path := writeYAML(t, `db_dsn: "x"`)
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.HTTPPort != 8080 {
		t.Errorf("HTTPPort default = %d", got.HTTPPort)
	}
	if got.PollInterval != 10*time.Minute {
		t.Errorf("PollInterval default = %v", got.PollInterval)
	}
	if got.PollConcurrency != 10 {
		t.Errorf("PollConcurrency default = %d", got.PollConcurrency)
	}
	if got.LogLevel != "info" {
		t.Errorf("LogLevel default = %q", got.LogLevel)
	}
	if got.GitHubToken != "" {
		t.Errorf("GitHubToken default = %q", got.GitHubToken)
	}
}

func TestLoad_MissingDBDSN(t *testing.T) {
	path := writeYAML(t, `http_port: 8080`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when db_dsn missing")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	path := writeYAML(t, `db_dsn: "x
http_port: 8080`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoad_InvalidPortType(t *testing.T) {
	path := writeYAML(t, `
db_dsn: "x"
http_port: "not-a-number"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	path := writeYAML(t, "")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when db_dsn missing from empty file")
	}
}