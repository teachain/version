# GitHub Release Version Monitor — Design Spec

Date: 2026-10-08
Topic: teachain-version service (GitHub release → version record monitor)

## 1. Purpose

`teachain-version` is a Go backend service that monitors registered GitHub repositories for new releases and persists each new release as a row in a `version` table. The list of repositories to watch is managed through an `application` table exposed via HTTP API.

Goals:
- Strict separation of concerns across `handler / service / repository / model`.
- Pluggable persistence and external API clients through interfaces.
- Robust background polling with deduplication and graceful error handling.
- Testable at every layer through dependency injection.

Non-goals:
- Triggering builds/CI on release detection (out of scope).
- Webhook-driven ingestion (we poll; webhook support is deferred).
- Authentication / authorization of HTTP endpoints (assume trusted).

## 2. Directory Structure and Responsibilities

```
teachain-version/
├── cmd/server/main.go               # Composition: config → clients → repos → handler → poller → http
├── config/config.go                 # Configuration loading (env / file)
├── docs/superpowers/specs/...      # Design + planning artifacts
├── go.mod
├── README.md
│
├── pkg/                              # Reusable across teachain services
│   └── github/
│       ├── release.go                # ReleaseRepository interface (FetchReleases)
│       ├── release_api.go            # go-github implementation
│       └── parser.go                  # repo_url → (owner, repo)
│
└── internal/                         # Strictly internal to this service
    ├── apperror/                     # Domain error types (NotFound/Conflict/BadRequest/Internal)
    │   └── apperror.go
    ├── handler/
    │   ├── application.go            # Application HTTP handlers
    │   └── version.go                # Version HTTP handlers
    ├── service/
    │   ├── application.go            # Application business rules
    │   └── version.go                # Version query rules
    ├── repository/
    │   ├── application.go            # ApplicationRepository interface
    │   ├── application_xorm.go       # XORM/MySQL implementation
    │   ├── version.go                # VersionRepository interface
    │   └── version_xorm.go           # XORM/MySQL implementation
    ├── poller/
    │   └── poller.go                 # Background ticker loop
    └── model/
        ├── application.go            # Application entity
        └── version.go                # Version entity
```

### Layer responsibilities

- **handler**: parse HTTP requests, validate DTO shape, translate `apperror` to status codes, call service.
- **service**: enforce business rules, orchestrate repositories, define transaction boundaries. Depends on interfaces only.
- **repository**: data access via XORM engine. Implementations are private to the package; only interfaces are exposed.
- **model**: entity structs and XORM tags only. No methods, no DTOs.
- **poller**: long-running goroutine that periodically calls `applicationRepo.ListEnabled` → `githubClient.FetchReleases` → `versionRepo.Save` with dedup.
- **pkg/github**: external-API client (interface + go-github implementation + URL parser). Lives in `pkg/` so sibling teachain services may reuse.
- **config**: load env / file config; no business logic.
- **cmd/server**: dependency wiring — composition root.

## 3. Dependency Direction (Strict)

```
handler → service → repository → model
                       ↑
                   (interfaces)
                  /             \
              poller  ←────  pkg/github
```

Rules:
- `handler` must not import repository or poller.
- `service` must not import handler or poller.
- `repository` must not import service, handler, or poller.
- `model` must not import any other internal package.
- `poller` depends on `repository` interfaces + `pkg/github.ReleaseRepository` interface only.
- `pkg/github` must not import anything from `internal/`.
- All wiring happens in `cmd/server/main.go` through interfaces.

## 4. Data Model

### Application

```go
type Application struct {
    ID          uint       `xorm:"pk autoincr 'id'"`
    Name        string     `xorm:"varchar(128) notnull unique 'name'"`
    RepoURL     string     `xorm:"varchar(256) notnull 'repo_url'"`
    Enabled     bool       `xorm:"notnull default true 'enabled'"`
    LastCheckAt *time.Time `xorm:"last_check_at"`
    CreatedAt   time.Time  `xorm:"created"`
    UpdatedAt   time.Time  `xorm:"updated"`
}
```

Fields:
- `ID`: auto-increment primary key.
- `Name`: human label, unique.
- `RepoURL`: full GitHub repo URL (e.g. `https://github.com/owner/repo`).
- `Enabled`: soft toggle for polling; defaults true.
- `LastCheckAt`: updated by poller after each cycle for that app; nullable until first check.

### Version

```go
type Version struct {
    ID            uint      `xorm:"pk autoincr 'id'"`
    ApplicationID uint      `xorm:"notnull unique(uk_app_tag) 'application_id'"`
    // The matching `unique(uk_app_tag)` on TagName completes the composite
    // unique index (application_id, tag_name) — the dedup backstop.
    TagName       string    `xorm:"varchar(64) notnull unique(uk_app_tag) 'tag_name'"`
    Name          string    `xorm:"varchar(256) 'name'"`
    Body          string    `xorm:"text 'body'"`
    URL           string    `xorm:"varchar(512) 'url'"`
    PublishedAt   time.Time `xorm:"published_at"`
    CreatedAt     time.Time `xorm:"created"`
}
```

Fields:
- `ApplicationID`: FK reference (logical; not enforced via FK constraint).
- `TagName`: GitHub release tag (e.g. `v1.2.3`).
- `Name`: release title.
- `Body`: release notes text.
- `URL`: release page URL on github.com.
- `PublishedAt`: release publication time from GitHub.
- `CreatedAt`: row insertion time (local clock).
- Composite unique index `(application_id, tag_name)` is the dedup backstop.

## 5. Repository Interfaces

```go
// repository/application.go
type ApplicationRepository interface {
    Save(ctx context.Context, a *model.Application) error
    Update(ctx context.Context, a *model.Application) error
    Delete(ctx context.Context, id uint) error
    Get(ctx context.Context, id uint) (*model.Application, error)
    GetByName(ctx context.Context, name string) (*model.Application, error)
    List(ctx context.Context, page, size int) ([]model.Application, int64, error)
    ListEnabled(ctx context.Context) ([]model.Application, error)
    UpdateLastCheckAt(ctx context.Context, id uint, t time.Time) error
}
```

```go
// repository/version.go
type VersionRepository interface {
    Save(ctx context.Context, v *model.Version) error
    Get(ctx context.Context, id uint) (*model.Version, error)
    PageByApp(ctx context.Context, appID uint, offset, limit int) ([]model.Version, int64, error)
    ListTagNamesByApp(ctx context.Context, appID uint) ([]string, error)
}
```

Implementations live in `application_xorm.go` / `version_xorm.go` using `*xorm.Engine`.

## 6. pkg/github Interfaces

```go
// pkg/github/release.go
type Release struct {
    TagName     string
    Name        string
    Body        string
    URL         string
    PublishedAt time.Time
}

type ReleaseRepository interface {
    FetchReleases(ctx context.Context, repoURL string) ([]Release, error)
}

// pkg/github/parser.go
func ParseRepoURL(raw string) (owner, repo string, err error)
```

`release_api.go` implements `ReleaseRepository` via `google/go-github`. `ParseRepoURL` accepts canonical `https://github.com/<owner>/<repo>` URLs (with optional trailing `.git`).

## 7. Service Interfaces

```go
// service/application.go
type ApplicationService interface {
    Create(ctx context.Context, name, repoURL string) (*model.Application, error)
    Update(ctx context.Context, id uint, name, repoURL *string, enabled *bool) (*model.Application, error)
    Delete(ctx context.Context, id uint) error
    Get(ctx context.Context, id uint) (*model.Application, error)
    List(ctx context.Context, page, size int) ([]model.Application, int64, error)
}
```

```go
// service/version.go
type VersionService interface {
    ListByApp(ctx context.Context, appID uint, page, size int) ([]model.Version, int64, error)
}
```

Business rules:
- `ApplicationService.Create` parses `repoURL` via `pkg/github.ParseRepoURL`; rejects unparseable URLs with `apperror.BadRequest`. Calls `applicationRepo.GetByName`; returns `apperror.Conflict` if duplicate.
- `ApplicationService.Update` parses `repoURL` if provided; ignores unchanged fields.
- `VersionService.ListByApp` validates page/size bounds; returns `apperror.BadRequest` on invalid input.

## 8. Handler Surface

| Method | Path | Purpose |
|--------|------|---------|
| POST   | /applications                          | Create application |
| GET    | /applications                          | List applications (paginated) |
| GET    | /applications/:id                      | Get one application |
| PATCH  | /applications/:id                      | Update application |
| DELETE | /applications/:id                      | Delete application |
| GET    | /applications/:id/versions              | List versions for application |

All endpoints return:
- `apperror.NotFound` → 404
- `apperror.Conflict` → 409
- `apperror.BadRequest` → 400
- other → 500

Success payloads are JSON; pagination responses use shape `{ "total": int, "items": [...] }`.

## 9. Polling Loop

`internal/poller/poller.go`:

```
type Poller struct {
    interval     time.Duration
    concurrency  int
    apps         repository.ApplicationRepository
    versions     repository.VersionRepository
    releases     github.ReleaseRepository
    log          *zap.Logger
}

func (p *Poller) Run(ctx context.Context)
```

`Run` semantics:
1. Block on `ticker := time.NewTicker(p.interval)`.
2. On each tick, fetch `apps := p.apps.ListEnabled()`.
3. Process apps with bounded concurrency (semaphore channel of size `p.concurrency`).
4. For each app:
   - `releases, err := p.releases.FetchReleases(ctx, app.RepoURL)` — log on error, continue.
   - `tags := p.versions.ListTagNamesByApp(ctx, app.ID)`.
   - For each release whose `TagName` is not in `tags`: `p.versions.Save(ctx, version)`. On unique-constraint error, log and continue.
   - Always call `p.apps.UpdateLastCheckAt(ctx, app.ID, time.Now())` regardless of fetch success.
5. Shut down cleanly when `ctx` is cancelled.

Configuration:
- `POLL_INTERVAL` (default `10m`).
- `POLL_CONCURRENCY` (default `10`).

## 10. Error Type

```go
// internal/apperror/apperror.go
type Kind int

const (
    NotFound Kind = iota + 1
    Conflict
    BadRequest
    Internal
)

type Error struct {
    Kind    Kind
    Cause  error
}

func (e *Error) Error() string { ... }
func (e *Error) Unwrap() error { return e.Cause }
```

Helper constructors: `NotFoundf`, `Conflictf`, `BadRequestf`, `InternalWrap`.

## 11. Configuration

Loaded by `config/config.go` from env:

```
DB_DSN              xorm:mysql://user:pass@tcp(host:3306)/teachain_version?charset=utf8mb4
GITHUB_TOKEN        (optional) Personal access token
HTTP_PORT           8080
POLL_INTERVAL       10m
POLL_CONCURRENCY    10
LOG_LEVEL           info
```

When `GITHUB_TOKEN` is empty, the GitHub client uses unauthenticated requests (rate limit 60/hr); when present, the token is supplied via go-github's `BasicAuthTransport` with username "x-access-token".

## 12. Testing Strategy

| Layer | Approach |
|-------|----------|
| handler | `httptest.NewRecorder` + Gin test engine + mock service |
| service | mock repository, unit tests over business rules |
| repository | real MySQL via testcontainers-go (single shared container per test binary) |
| pkg/github | interface mock + table-driven cases for `ParseRepoURL` |
| poller | all dependencies mocked; fake clock for ticker control |

Coverage targets:
- service + handler: ≥80%
- repository: ≥70% (covers CRUD paths)
- pkg/github parser: ≥90%

## 13. Dependencies (go.mod)

- `github.com/gin-gonic/gin`
- `xorm.io/xorm`
- `github.com/go-sql-driver/mysql` (xorm driver)
- `github.com/google/go-github/v60` (latest at implementation time)
- `go.uber.org/zap` (logging)
- `github.com/stretchr/testify` (test only)
- `github.com/testcontainers/testcontainers-go` (test only)

## 14. Out of Scope (Deferred)

- Authentication / authorization of HTTP endpoints.
- Webhook-driven release ingestion from GitHub.
- Notifications (email, Slack) on new release detection.
- Multi-tenant separation.
- Soft-delete for applications.
- Release diffing between consecutive tags.
- Internationalisation of release notes.

## 15. Open Questions

None at design freeze time.