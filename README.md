# teachain-version

Monitors registered GitHub repositories for new releases and persists each
release as a row in the `version` table.

See `docs/superpowers/specs/2026-10-08-github-version-monitor-design.md`
for the design spec.

## Local development

```bash
docker run --rm -d --name teachain-mysql \
  -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=teachain \
  -p 3306:3306 mysql:8.0
DB_DSN='root:root@tcp(127.0.0.1:3306)/teachain?charset=utf8mb4' \
POLL_INTERVAL=10s \
go run ./cmd/server
```