# teachain-version

Monitors registered GitHub repositories for new releases and persists each
release as a row in the `version` table.

See `docs/superpowers/specs/2026-10-08-github-version-monitor-design.md`
for the design spec.

## Configuration

Configuration is loaded from a YAML file. Path resolution precedence:

1. `--config` / `-c` CLI flag
2. `CONFIG_FILE` environment variable
3. `./config.yaml` (default)

See `config.example.yaml` for all supported keys. `db_dsn` is required; other fields have sensible defaults.

## Local development

```bash
docker run --rm -d --name teachain-mysql \
  -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=teachain \
  -p 3306:3306 mysql:8.0

cp config.example.yaml config.yaml
# edit config.yaml if your MySQL host/port differs

go run ./cmd/server
```

To use a custom config file path or env override:

```bash
go run ./cmd/server --config /etc/teachain/version.yaml
# or
CONFIG_FILE=/path/to/config.yaml go run ./cmd/server
```