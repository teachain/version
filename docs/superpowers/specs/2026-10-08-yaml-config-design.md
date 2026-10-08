# YAML Configuration — Design Spec

Date: 2026-10-08
Topic: Replace env-based config with YAML file for teachain-version

## 1. Purpose

Switch the `teachain-version` service's configuration source from environment variables (env) to a YAML config file. The configuration surface, struct, and downstream consumers stay unchanged — only the loading mechanism changes.

Goals:
- Single file holds all config values, easy to read/diff/track in git.
- Path is configurable via `--config` flag or `CONFIG_FILE` env var, with `./config.yaml` fallback.
- Defaults applied for non-required fields (HTTP port, concurrency, log level, etc.).
- Required fields (DB_DSN) missing → clear error.

Non-goals:
- Hot reload (file watching) — out of scope.
- Remote config sources (etcd, consul) — out of scope.
- Env var overrides on top of YAML file — we are removing env-based config.

## 2. Configuration schema

Flat YAML structure (one level, no groups):

```yaml
db_dsn: "root:root@tcp(127.0.0.1:3306)/teachain?charset=utf8mb4"
github_token: ""
http_port: 8080
poll_interval: 10m
poll_concurrency: 10
log_level: info
```

Field semantics match the existing `Config` struct:

| Field | Type | Default | Required |
|---|---|---|---|
| `db_dsn` | string | (none) | Yes — missing/empty returns error |
| `github_token` | string | `""` (unauthenticated) | No |
| `http_port` | int | `8080` | No |
| `poll_interval` | duration | `10m` | No |
| `poll_concurrency` | int | `10` | No |
| `log_level` | string | `"info"` | No |

Comments in YAML (using `#`) are allowed but ignored. Unknown fields are ignored (forwards-compatibility). Invalid types (e.g. `http_port: "abc"`) return a parse error.

## 3. Path resolution

Order of precedence (highest first):
1. `--config` / `-c` CLI flag value (if non-empty)
2. `CONFIG_FILE` environment variable (if non-empty)
3. `./config.yaml` (relative to working directory)

If the resolved path does not exist, `config.Load` returns a clear error: `"config file not found: <path>"`.

Absolute paths are accepted as-is. Relative paths are resolved against the current process's working directory.

## 4. Architecture

Single file change in package boundary:

```
config/config.go              # YAML loading replaces env loading
config/config_test.go         # Update + add tests
cmd/server/main.go            # Add flag parsing; pass path to config.Load
config.example.yaml           # New example file at repo root
README.md                     # Update local-dev section
go.mod / go.sum               # Add gopkg.in/yaml.v3
```

No downstream changes: `Config` struct fields and types stay identical; the rest of the code (`service`, `handler`, `poller`) is untouched.

## 5. Config struct

```go
package config

type Config struct {
    DBDSN           string        `yaml:"db_dsn"`
    GitHubToken     string        `yaml:"github_token"`
    HTTPPort        int           `yaml:"http_port"`
    PollInterval    time.Duration `yaml:"poll_interval"`
    PollConcurrency int           `yaml:"poll_concurrency"`
    LogLevel        string        `yaml:"log_level"`
}

func Load(path string) (*Config, error)
```

`Load(path)` behavior:
1. Read file at `path` (use `os.ReadFile`).
2. On `os.IsNotExist`: return error `"config file not found: <path>"`.
3. Parse YAML via `yaml.Unmarshal` (gopkg.in/yaml.v3).
4. On YAML parse error: return wrapped error with file path context.
5. Apply defaults for empty/zero fields:
   - `HTTPPort == 0` → `8080`
   - `PollInterval == 0` → `10 * time.Minute`
   - `PollConcurrency == 0` → `10`
   - `LogLevel == ""` → `"info"`
6. `DBDSN` still empty after defaults → return error `"db_dsn is required"`.
7. Return the populated `*Config`.

`Unknown fields` in the YAML document do NOT cause errors (yaml.v3 `KnownFields(false)` is the default). Strict mode is intentionally off for flexibility.

## 6. main.go changes

```go
func main() {
    var configPath string
    flag.StringVar(&configPath, "config", "", "path to YAML config file")
    flag.StringVar(&configPath, "c", "", "path to YAML config file (shorthand)")
    flag.Parse()

    if configPath == "" {
        configPath = os.Getenv("CONFIG_FILE")
    }
    if configPath == "" {
        configPath = "./config.yaml"
    }

    cfg, err := config.Load(configPath)
    if err != nil {
        log.Fatalf("config: %v", err)
    }
    // ... rest unchanged
}
```

`flag.Parse()` is called before `config.Load` and before logger construction so flag parse failures (e.g., unknown flag) exit cleanly.

The rest of `main.go` (`newLogger`, engine setup, route registration, poller goroutine, graceful shutdown) is unchanged.

## 7. Testing approach

| Test | Covers |
|---|---|
| `TestLoad_Defaults` | Load minimal YAML (only db_dsn) → defaults applied |
| `TestLoad_AllFieldsSet` | Load full YAML → all values reflected |
| `TestLoad_MissingDBDSN` | YAML without db_dsn → error |
| `TestLoad_FileNotFound` | Non-existent path → clear "not found" error |
| `TestLoad_MalformedYAML` | Invalid YAML syntax → wrapped parse error |
| `TestLoad_InvalidPort` | `http_port: "abc"` → unmarshal error |

All tests use `t.TempDir()` to create a temporary file path. No env var manipulation required.

## 8. Dependencies

- Add: `gopkg.in/yaml.v3` (de-facto standard; clean error type with `yaml.TypeError`).

## 9. Out of scope

- Hot reload / file watching.
- Remote config sources.
- Env-var overrides on top of YAML.
- Schema validation beyond type checking.
- Multiple profile / environment-based configs (e.g., `config.dev.yaml`).