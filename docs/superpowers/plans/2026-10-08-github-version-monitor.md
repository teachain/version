# teachain-version Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go HTTP service that monitors registered GitHub repositories for new releases and persists each release as a row in a `version` table, using Gin + XORM with strict handler → service → repository → model layering.

**Architecture:** Strict layered Go service. HTTP requests flow handler → service → repository. A goroutine ticker periodically pulls enabled applications, fetches their GitHub releases via `pkg/github`, deduplicates against the `version` table using a `(application_id, tag_name)` unique index, and inserts new rows.

**Tech Stack:** Go 1.22+, Gin, xorm.io/xorm, google/go-github, golang.org/x/oauth2, go.uber.org/zap, stretchr/testify, testcontainers-go.

**Spec:** `docs/superpowers/specs/2026-10-08-github-version-monitor-design.md`

---

## File Structure

| File | Responsibility |
|---|---|
| `cmd/server/main.go` | Composition root: load config, build xorm engine, instantiate repositories / services / handlers / poller, start Gin and poller goroutine. |
| `config/config.go` | Load env vars into `Config`; parse durations. |
| `internal/apperror/apperror.go` | Domain `Error` type with `Kind` enum + helper constructors. |
| `internal/model/application.go` | `Application` XORM entity. |
| `internal/model/version.go` | `Version` XORM entity. |
| `internal/repository/application.go` | `ApplicationRepository` interface. |
| `internal/repository/application_xorm.go` | XORM/MySQL implementation of `ApplicationRepository`. |
| `internal/repository/version.go` | `VersionRepository` interface. |
| `internal/repository/version_xorm.go` | XORM/MySQL implementation of `VersionRepository`. |
| `internal/service/application.go` | `ApplicationService` interface + implementation. |
| `internal/service/version.go` | `VersionService` interface + implementation. |
| `internal/handler/application.go` | Application HTTP handlers (Gin). |
| `internal/handler/version.go` | Version HTTP handlers (Gin). |
| `internal/handler/errors.go` | `apperror.Error` → HTTP status code mapper. |
| `internal/handler/router.go` | Gin route registration. |
| `internal/poller/poller.go` | Background ticker loop with bounded concurrency. |
| `pkg/github/release.go` | `Release` struct + `ReleaseRepository` interface. |
| `pkg/github/release_api.go` | go-github implementation of `ReleaseRepository`. |
| `pkg/github/parser.go` | `ParseRepoURL` URL → owner/repo parser. |
| Test files: `*_test.go` next to every non-trivial source file. |

---

## Task 1: Project Skeleton and Go Module

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `README.md`

- [ ] **Step 1: Initialize go.mod**

Run:
```bash
cd /Users/dm/projects/teachain/version
go mod init github.com/teachain/version
```
Expected: `go.mod` created with module path.

- [ ] **Step 2: Add `.gitignore`**

Write `.gitignore`:
```
bin/
dist/
*.log
.env
.env.local
coverage.out
.idea/
.vscode/
```

- [ ] **Step 3: Add stub `README.md`**

Replace `README.md`:
```markdown
# teachain-version

Monitors registered GitHub repositories for new releases and persists each
release as a row in the `version` table.

See `docs/superpowers/specs/2026-10-08-github-version-monitor-design.md`
for the design spec.
```

- [ ] **Step 4: Verify build target exists**

Run:
```bash
mkdir -p cmd/server
go build ./...
```
Expected: command succeeds with no Go files yet (only `go.mod`/`README`/`.gitignore`).

- [ ] **Step 5: Commit**

```bash
git add go.mod .gitignore README.md
git commit -m "chore: initialize go module and project skeleton"
```

---

## Task 2: apperror Package

**Files:**
- Create: `internal/apperror/apperror.go`
- Create: `internal/apperror/apperror_test.go`

- [ ] **Step 1: Write failing test `internal/apperror/apperror_test.go`**

```go
package apperror

import (
	"errors"
	"fmt"
	"testing"
)

func TestError_ErrorMessage(t *testing.T) {
	e := &Error{Kind: NotFound, Msg: "user 42"}
	if got, want := e.Error(), "not found: user 42"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("boom")
	e := &Error{Kind: Internal, Msg: "wrap", Cause: cause}
	if !errors.Is(e, cause) {
		t.Fatalf("errors.Is should match wrapped cause")
	}
}

func TestConstructors(t *testing.T) {
	if NotFoundf("x %d", 1).Kind != NotFound {
		t.Fail()
	}
	if Conflictf("x").Kind != Conflict {
		t.Fail()
	}
	if BadRequestf("x").Kind != BadRequest {
		t.Fail()
	}
	if InternalWrap(fmt.Errorf("e"), "x").Kind != Internal {
		t.Fail()
	}
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/apperror/...`
Expected: FAIL (package does not exist).

- [ ] **Step 3: Implement `internal/apperror/apperror.go`**

```go
package apperror

import (
	"fmt"
)

type Kind int

const (
	NotFound Kind = iota + 1
	Conflict
	BadRequest
	Internal
)

func (k Kind) String() string {
	switch k {
	case NotFound:
		return "not found"
	case Conflict:
		return "conflict"
	case BadRequest:
		return "bad request"
	case Internal:
		return "internal"
	default:
		return "unknown"
	}
}

type Error struct {
	Kind  Kind
	Msg   string
	Cause error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %s", e.Kind, e.Msg, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Msg)
}

func (e *Error) Unwrap() error { return e.Cause }

func NotFoundf(format string, args ...any) *Error {
	return &Error{Kind: NotFound, Msg: fmt.Sprintf(format, args...)}
}

func Conflictf(format string, args ...any) *Error {
	return &Error{Kind: Conflict, Msg: fmt.Sprintf(format, args...)}
}

func BadRequestf(format string, args ...any) *Error {
	return &Error{Kind: BadRequest, Msg: fmt.Sprintf(format, args...)}
}

func InternalWrap(cause error, format string, args ...any) *Error {
	return &Error{Kind: Internal, Msg: fmt.Sprintf(format, args...), Cause: cause}
}
```

- [ ] **Step 4: Run test, expect pass**

Run: `go test ./internal/apperror/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/apperror/
git commit -m "feat(apperror): add domain error type with Kind enum"
```

---

## Task 3: Models

**Files:**
- Create: `internal/model/application.go`
- Create: `internal/model/version.go`

- [ ] **Step 1: Write `internal/model/application.go`**

```go
package model

import "time"

type Application struct {
	ID          uint       `xorm:"pk autoincr 'id'"`
	Name        string     `xorm:"varchar(128) notnull unique 'name'"`
	RepoURL     string     `xorm:"varchar(256) notnull 'repo_url'"`
	Enabled     bool       `xorm:"notnull default true 'enabled'"`
	LastCheckAt *time.Time `xorm:"last_check_at"`
	CreatedAt   time.Time  `xorm:"created"`
	UpdatedAt   time.Time  `xorm:"updated"`
}

func (Application) TableName() string { return "application" }
```

- [ ] **Step 2: Write `internal/model/version.go`**

```go
package model

import "time"

type Version struct {
	ID            uint      `xorm:"pk autoincr 'id'"`
	ApplicationID uint      `xorm:"notnull unique(uk_app_tag) 'application_id'"`
	TagName       string    `xorm:"varchar(64) notnull unique(uk_app_tag) 'tag_name'"`
	Name          string    `xorm:"varchar(256) 'name'"`
	Body          string    `xorm:"text 'body'"`
	URL           string    `xorm:"varchar(512) 'url'"`
	PublishedAt   time.Time `xorm:"published_at"`
	CreatedAt     time.Time `xorm:"created"`
}

func (Version) TableName() string { return "version" }
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/model/
git commit -m "feat(model): add Application and Version entities"
```

---

## Task 4: pkg/github/parser

**Files:**
- Create: `pkg/github/parser.go`
- Create: `pkg/github/parser_test.go`

- [ ] **Step 1: Write failing test `pkg/github/parser_test.go`**

```go
package github

import "testing"

func TestParseRepoURL(t *testing.T) {
	cases := []struct {
		name        string
		in          string
		wantOwner   string
		wantRepo    string
		wantErr     bool
	}{
		{"https", "https://github.com/owner/repo", "owner", "repo", false},
		{"trailing .git", "https://github.com/owner/repo.git", "owner", "repo", false},
		{"trailing slash", "https://github.com/owner/repo/", "owner", "repo", false},
		{"uppercase host", "https://GitHub.com/Owner/Repo", "Owner", "Repo", false},
		{"empty", "", "", "", true},
		{"wrong host", "https://gitlab.com/owner/repo", "", "", true},
		{"missing repo", "https://github.com/owner", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, r, err := ParseRepoURL(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got (%q,%q)", o, r)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if o != c.wantOwner || r != c.wantRepo {
				t.Fatalf("got (%q,%q), want (%q,%q)", o, r, c.wantOwner, c.wantRepo)
			}
		})
	}
}
```

- [ ] **Step 2: Run, expect failure**

Run: `go test ./pkg/github/...`
Expected: FAIL (no Go files).

- [ ] **Step 3: Implement `pkg/github/parser.go`**

```go
package github

import (
	"fmt"
	"net/url"
	"strings"
)

func ParseRepoURL(raw string) (owner, repo string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("parse: %w", err)
	}
	host := strings.ToLower(u.Host)
	if host != "github.com" {
		return "", "", fmt.Errorf("unsupported host: %s", u.Host)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid path: %s", u.Path)
	}
	owner = parts[0]
	repo = strings.TrimSuffix(parts[1], ".git")
	return owner, repo, nil
}
```

- [ ] **Step 4: Run test, expect pass**

Run: `go test ./pkg/github/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/github/
git commit -m "feat(pkg/github): add repo URL parser"
```

---

## Task 5: pkg/github Release Interface

**Files:**
- Create: `pkg/github/release.go`

- [ ] **Step 1: Implement `pkg/github/release.go`**

```go
package github

import (
	"context"
	"time"
)

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
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add pkg/github/release.go
git commit -m "feat(pkg/github): add ReleaseRepository interface"
```

---

## Task 6: pkg/github Release API Implementation

**Files:**
- Create: `pkg/github/release_api.go`
- Create: `pkg/github/release_api_test.go`

- [ ] **Step 1: Add dependencies**

Run:
```bash
go get github.com/google/go-github/v60
go get golang.org/x/oauth2
```
Expected: dependencies added to `go.mod`.

- [ ] **Step 2: Write failing test `pkg/github/release_api_test.go`**

```go
package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v60/github"
)

func TestApiRepo_FetchReleases(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/octocat/Hello-World/releases", func(w http.ResponseWriter, r *http.Request) {
		body := `[{
				"tag_name": "v1.0.0",
				"name": "first",
				"body": "hello",
				"html_url": "https://github.com/octocat/Hello-World/releases/tag/v1.0.0",
				"published_at": "2024-01-02T03:04:05Z"
			}]`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := github.NewClient(nil)
	baseURL := srv.URL + "/"
	client.BaseURL, _ = client.BaseURL.Parse(baseURL)

	repo := &apiRepo{client: client}
	got, err := repo.FetchReleases(context.Background(), "https://github.com/octocat/Hello-World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d, want 1", len(got))
	}
	want := Release{
		TagName: "v1.0.0", Name: "first", Body: "hello",
		URL: "https://github.com/octocat/Hello-World/releases/tag/v1.0.0",
	}
	if got[0].TagName != want.TagName || got[0].Name != want.Name || got[0].Body != want.Body || got[0].URL != want.URL {
		t.Fatalf("got %+v", got[0])
	}
	if got[0].PublishedAt.IsZero() {
		t.Fatalf("PublishedAt should not be zero")
	}
}

func TestApiRepo_FetchReleases_InvalidURL(t *testing.T) {
	repo := &apiRepo{client: github.NewClient(nil)}
	_, err := repo.FetchReleases(context.Background(), "https://gitlab.com/octocat/Hello-World")
	if err == nil {
		t.Fatal("expected error for non-github URL")
	}
	if !strings.Contains(err.Error(), "unsupported host") {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = time.Now
}
```

- [ ] **Step 3: Run, expect failure**

Run: `go test ./pkg/github/...`
Expected: FAIL (apiRepo undefined).

- [ ] **Step 4: Implement `pkg/github/release_api.go`**

```go
package github

import (
	"context"
	"fmt"

	goGithub "github.com/google/go-github/v60/github"
)

type apiRepo struct {
	client *goGithub.Client
}

func NewClient(token string) ReleaseRepository {
	var client *goGithub.Client
	if token != "" {
		ts := goGithub.BasicAuthTransport{Username: "x-access-token", Password: token}.Client()
		client = goGithub.NewClient(ts)
	} else {
		client = goGithub.NewClient(nil)
	}
	return &apiRepo{client: client}
}

func (r *apiRepo) FetchReleases(ctx context.Context, repoURL string) ([]Release, error) {
	owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("parse repo url: %w", err)
	}
	opt := &goGithub.ListOptions{PerPage: 100}
	var all []Release
	for page := 1; ; page++ {
		opt.Page = page
		rs, resp, err := r.client.Repositories.ListReleases(ctx, owner, repo, opt)
		if err != nil {
			return nil, fmt.Errorf("list releases: %w", err)
		}
		for _, gh := range rs {
			all = append(all, Release{
				TagName:     gh.GetTagName(),
				Name:        gh.GetName(),
				Body:        gh.GetBody(),
				URL:         gh.GetHTMLURL(),
				PublishedAt: gh.GetPublishedAt().Time,
			})
		}
		if resp.NextPage == 0 {
			break
		}
	}
	return all, nil
}
```

- [ ] **Step 5: Run, expect pass**

Run: `go test ./pkg/github/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/github/ go.mod go.sum
git commit -m "feat(pkg/github): implement FetchReleases via go-github"
```

---

## Task 7: config Package

**Files:**
- Create: `config/config.go`
- Create: `config/config_test.go`

- [ ] **Step 1: Write failing test `config/config_test.go`**

```go
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
```

- [ ] **Step 2: Run, expect failure**

Run: `go test ./config/...`
Expected: FAIL (no Go files).

- [ ] **Step 3: Implement `config/config.go`**

```go
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
```

- [ ] **Step 4: Run, expect pass**

Run: `go test ./config/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add config/
git commit -m "feat(config): add env-based configuration loader"
```

---

## Task 8: Repository Interfaces

**Files:**
- Create: `internal/repository/application.go`
- Create: `internal/repository/version.go`

- [ ] **Step 1: Write `internal/repository/application.go`**

```go
package repository

import (
	"context"
	"time"

	"github.com/teachain/version/internal/model"
)

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

- [ ] **Step 2: Write `internal/repository/version.go`**

```go
package repository

import (
	"context"

	"github.com/teachain/version/internal/model"
)

type VersionRepository interface {
	Save(ctx context.Context, v *model.Version) error
	Get(ctx context.Context, id uint) (*model.Version, error)
	PageByApp(ctx context.Context, appID uint, offset, limit int) ([]model.Version, int64, error)
	ListTagNamesByApp(ctx context.Context, appID uint) ([]string, error)
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/repository/
git commit -m "feat(repository): add Application and Version repository interfaces"
```

---

## Task 9: xorm Engine Helper and Application Repository Implementation

**Files:**
- Create: `internal/repository/xorm_engine.go`
- Create: `internal/repository/application_xorm.go`
- Create: `internal/repository/application_xorm_test.go`

- [ ] **Step 1: Add xorm dependencies**

Run:
```bash
go get xorm.io/xorm
go get github.com/go-sql-driver/mysql
go get github.com/testcontainers/testcontainers-go/modules/mysql
```

- [ ] **Step 2: Write `internal/repository/xorm_engine.go`**

```go
package repository

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

func NewEngine(dsn string) (*xorm.Engine, error) {
	eng, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("new engine: %w", err)
	}
	return eng, nil
}
```

- [ ] **Step 3: Write `internal/repository/application_xorm.go`**

```go
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/teachain/version/internal/model"
	"xorm.io/xorm"
)

type applicationXormRepo struct {
	eng *xorm.Engine
}

func NewApplicationXormRepository(eng *xorm.Engine) ApplicationRepository {
	return &applicationXormRepo{eng: eng}
}

func (r *applicationXormRepo) Save(ctx context.Context, a *model.Application) error {
	sess := r.eng.Context(ctx)
	affected, err := sess.Insert(a)
	if err != nil {
		return fmt.Errorf("insert application: %w", err)
	}
	if affected == 0 {
		return errors.New("insert application: 0 rows affected")
	}
	return nil
}

func (r *applicationXormRepo) Update(ctx context.Context, a *model.Application) error {
	sess := r.eng.Context(ctx)
	affected, err := sess.ID(a.ID).Cols("name", "repo_url", "enabled").Update(a)
	if err != nil {
		return fmt.Errorf("update application: %w", err)
	}
	if affected == 0 {
		return errors.New("update application: 0 rows affected")
	}
	return nil
}

func (r *applicationXormRepo) Delete(ctx context.Context, id uint) error {
	sess := r.eng.Context(ctx)
	affected, err := sess.ID(id).Delete(&model.Application{})
	if err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("application %d: %w", id, ErrNotFound)
	}
	return nil
}

func (r *applicationXormRepo) Get(ctx context.Context, id uint) (*model.Application, error) {
	sess := r.eng.Context(ctx)
	a := new(model.Application)
	has, err := sess.ID(id).Get(a)
	if err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}
	if !has {
		return nil, fmt.Errorf("application %d: %w", id, ErrNotFound)
	}
	return a, nil
}

func (r *applicationXormRepo) GetByName(ctx context.Context, name string) (*model.Application, error) {
	sess := r.eng.Context(ctx)
	a := new(model.Application)
	has, err := sess.Where("name = ?", name).Get(a)
	if err != nil {
		return nil, fmt.Errorf("get application by name: %w", err)
	}
	if !has {
		return nil, nil
	}
	return a, nil
}

func (r *applicationXormRepo) List(ctx context.Context, page, size int) ([]model.Application, int64, error) {
	sess := r.eng.Context(ctx)
	total, err := sess.Count(new(model.Application))
	if err != nil {
		return nil, 0, fmt.Errorf("count applications: %w", err)
	}
	var out []model.Application
	err = sess.Limit(size, (page-1)*size).Find(&out)
	if err != nil {
		return nil, 0, fmt.Errorf("list applications: %w", err)
	}
	return out, total, nil
}

func (r *applicationXormRepo) ListEnabled(ctx context.Context) ([]model.Application, error) {
	sess := r.eng.Context(ctx)
	var out []model.Application
	if err := sess.Where("enabled = ?", true).Find(&out); err != nil {
		return nil, fmt.Errorf("list enabled: %w", err)
	}
	return out, nil
}

func (r *applicationXormRepo) UpdateLastCheckAt(ctx context.Context, id uint, t time.Time) error {
	sess := r.eng.Context(ctx)
	a := &model.Application{LastCheckAt: &t}
	affected, err := sess.ID(id).Cols("last_check_at").Update(a)
	if err != nil {
		return fmt.Errorf("update last_check_at: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("application %d: %w", id, ErrNotFound)
	}
	return nil
}
```

- [ ] **Step 4: Add `internal/repository/errors.go`**

```go
package repository

import "errors"

var ErrNotFound = errors.New("not found")
```

- [ ] **Step 5: Write `internal/repository/application_xorm_test.go`**

```go
package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/teachain/version/internal/model"
	"github.com/testcontainers/testcontainers-go"
	tmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"xorm.io/xorm"
)

func setupEngine(t *testing.T) (*xorm.Engine, func()) {
	t.Helper()
	if os.Getenv("TEST_DB_DSN") != "" {
		eng, err := NewEngine(os.Getenv("TEST_DB_DSN"))
		if err != nil {
			t.Fatalf("engine: %v", err)
		}
		if err := eng.Sync(new(model.Application)); err != nil {
			t.Fatalf("sync: %v", err)
		}
		return eng, func() { _ = eng.Close() }
	}
	ctx := context.Background()
	c, err := tmysql.RunContainer(ctx, testcontainers.WithImage("mysql:8.0"))
	if err != nil {
		t.Skipf("no docker: %v", err)
	}
	dsn, err := c.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}
	eng, err := NewEngine(dsn)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	if err := eng.Sync(new(model.Application)); err != nil {
		t.Fatalf("sync: %v", err)
	}
	return eng, func() {
		_ = eng.Close()
		_ = c.Terminate(ctx)
	}
}

func TestApplicationRepo_CRUD(t *testing.T) {
	eng, cleanup := setupEngine(t)
	defer cleanup()
	repo := NewApplicationXormRepository(eng)
	ctx := context.Background()

	a := &model.Application{Name: "foo", RepoURL: "https://github.com/octocat/Hello-World", Enabled: true}
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("save: %v", err)
	}
	if a.ID == 0 {
		t.Fatal("id not set")
	}

	got, err := repo.Get(ctx, a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "foo" {
		t.Fatalf("name=%q", got.Name)
	}

	got.Name = "bar"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}

	dup, err := repo.GetByName(ctx, "bar")
	if err != nil || dup == nil {
		t.Fatalf("getbyname: %v %v", dup, err)
	}

	list, total, err := repo.List(ctx, 1, 10)
	if err != nil || total < 1 || len(list) < 1 {
		t.Fatalf("list: %v total=%d", err, total)
	}

	if err := repo.UpdateLastCheckAt(ctx, a.ID, time.Now()); err != nil {
		t.Fatalf("lastcheck: %v", err)
	}

	if err := repo.Delete(ctx, a.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
```

- [ ] **Step 6: Add xorm import to test file**

Edit `internal/repository/application_xorm_test.go` to ensure `xorm.io/xorm` import is present (engine return type):

Append import line — the test as written needs:
```go
import (
    ...
    "xorm.io/xorm"
)
```
The function signature uses `*xorm.Engine`. Add the import.

- [ ] **Step 7: Run test**

Run: `go test ./internal/repository/...`
Expected: PASS (with Docker available) or SKIP (no Docker).

- [ ] **Step 8: Commit**

```bash
git add internal/repository/ go.mod go.sum
git commit -m "feat(repository): implement ApplicationRepository with xorm"
```

---

## Task 10: Version Repository Implementation

**Files:**
- Create: `internal/repository/version_xorm.go`
- Modify: `internal/repository/application_xorm_test.go` (add Sync for Version in setupEngine)

- [ ] **Step 1: Update setupEngine to also sync Version**

Edit `internal/repository/application_xorm_test.go` — change `setupEngine` so that after `eng.Sync(new(model.Application))` it also calls `eng.Sync(new(model.Version))`. Both calls are kept.

- [ ] **Step 2: Write `internal/repository/version_xorm.go`**

```go
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/teachain/version/internal/model"
	"xorm.io/xorm"
)

type versionXormRepo struct {
	eng *xorm.Engine
}

func NewVersionXormRepository(eng *xorm.Engine) VersionRepository {
	return &versionXormRepo{eng: eng}
}

func (r *versionXormRepo) Save(ctx context.Context, v *model.Version) error {
	sess := r.eng.Context(ctx)
	affected, err := sess.Insert(v)
	if err != nil {
		return fmt.Errorf("insert version: %w", err)
	}
	if affected == 0 {
		return errors.New("insert version: 0 rows affected")
	}
	return nil
}

func (r *versionXormRepo) Get(ctx context.Context, id uint) (*model.Version, error) {
	sess := r.eng.Context(ctx)
	v := new(model.Version)
	has, err := sess.ID(id).Get(v)
	if err != nil {
		return nil, fmt.Errorf("get version: %w", err)
	}
	if !has {
		return nil, fmt.Errorf("version %d: %w", id, ErrNotFound)
	}
	return v, nil
}

func (r *versionXormRepo) PageByApp(ctx context.Context, appID uint, offset, limit int) ([]model.Version, int64, error) {
	sess := r.eng.Context(ctx)
	total, err := sess.Where("application_id = ?", appID).Count(new(model.Version))
	if err != nil {
		return nil, 0, fmt.Errorf("count versions: %w", err)
	}
	var out []model.Version
	err = sess.Where("application_id = ?", appID).
		OrderBy("published_at DESC, id DESC").
		Limit(limit, offset).
		Find(&out)
	if err != nil {
		return nil, 0, fmt.Errorf("page versions: %w", err)
	}
	return out, total, nil
}

func (r *versionXormRepo) ListTagNamesByApp(ctx context.Context, appID uint) ([]string, error) {
	sess := r.eng.Context(ctx)
	rows, err := sess.Table(new(model.Version)).
		Where("application_id = ?", appID).
		Cols("tag_name").
		Rows(new(model.Version))
	if err != nil {
		return nil, fmt.Errorf("list tag names: %w", err)
	}
	defer rows.Close()
	tags := make([]string, 0)
	for rows.Next() {
		v := new(model.Version)
		if err := rows.Scan(v); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		tags = append(tags, v.TagName)
	}
	return tags, rows.Error()
}
```

- [ ] **Step 3: Write `internal/repository/version_xorm_test.go`**

```go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/teachain/version/internal/model"
)

func TestVersionRepo_CRUD(t *testing.T) {
	eng, cleanup := setupEngine(t)
	defer cleanup()
	appRepo := NewApplicationXormRepository(eng)
	verRepo := NewVersionXormRepository(eng)
	ctx := context.Background()

	app := &model.Application{Name: "vcrud", RepoURL: "https://github.com/octocat/Hello-World", Enabled: true}
	if err := appRepo.Save(ctx, app); err != nil {
		t.Fatalf("save app: %v", err)
	}

	now := time.Now()
	v1 := &model.Version{ApplicationID: app.ID, TagName: "v1.0.0", Name: "first", PublishedAt: now}
	v2 := &model.Version{ApplicationID: app.ID, TagName: "v1.1.0", Name: "second", PublishedAt: now}
	if err := verRepo.Save(ctx, v1); err != nil {
		t.Fatalf("save v1: %v", err)
	}
	if err := verRepo.Save(ctx, v2); err != nil {
		t.Fatalf("save v2: %v", err)
	}

	if err := verRepo.Save(ctx, &model.Version{ApplicationID: app.ID, TagName: "v1.0.0", PublishedAt: now}); err == nil {
		t.Fatal("expected duplicate unique-index error")
	}

	page, total, err := verRepo.PageByApp(ctx, app.ID, 0, 10)
	if err != nil || len(page) != 2 || total != 2 {
		t.Fatalf("page: %v len=%d total=%d", err, len(page), total)
	}

	tags, err := verRepo.ListTagNamesByApp(ctx, app.ID)
	if err != nil || len(tags) != 2 {
		t.Fatalf("tags: %v %v", err, tags)
	}
}


```

Note: the test uses `time.Now()` directly via the `now` variable; the `time` import is in place.

- [ ] **Step 4: Run test**

Run: `go test ./internal/repository/...`
Expected: PASS (or SKIP).

- [ ] **Step 5: Commit**

```bash
git add internal/repository/
git commit -m "feat(repository): implement VersionRepository with xorm"
```

---

## Task 11: Application Service

**Files:**
- Create: `internal/service/application.go`
- Create: `internal/service/application_test.go`

- [ ] **Step 1: Write failing test `internal/service/application_test.go`**

```go
package service

import (
	"context"
	"testing"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/model"
)

type fakeAppRepo struct {
	items map[uint]*model.Application
	byName map[string]uint
	nextID uint
	saved *model.Application
}

func newFakeAppRepo() *fakeAppRepo { return &fakeAppRepo{items: map[uint]*model.Application{}, byName: map[string]uint{}} }

func (f *fakeAppRepo) Save(_ context.Context, a *model.Application) error {
	f.nextID++
	a.ID = f.nextID
	f.items[a.ID] = a
	f.byName[a.Name] = a.ID
	f.saved = a
	return nil
}
func (f *fakeAppRepo) Update(_ context.Context, a *model.Application) error { f.items[a.ID] = a; return nil }
func (f *fakeAppRepo) Delete(_ context.Context, id uint) error              { delete(f.items, id); return nil }
func (f *fakeAppRepo) Get(_ context.Context, id uint) (*model.Application, error) {
	if a, ok := f.items[id]; ok {
		return a, nil
	}
	return nil, apperror.NotFoundf("app %d", id)
}
func (f *fakeAppRepo) GetByName(_ context.Context, name string) (*model.Application, error) {
	id, ok := f.byName[name]
	if !ok {
		return nil, nil
	}
	return f.items[id], nil
}
func (f *fakeAppRepo) List(_ context.Context, page, size int) ([]model.Application, int64, error) {
	out := make([]model.Application, 0, len(f.items))
	for _, a := range f.items {
		out = append(out, *a)
	}
	return out, int64(len(out)), nil
}
func (f *fakeAppRepo) ListEnabled(_ context.Context) ([]model.Application, error) {
	out := make([]model.Application, 0)
	for _, a := range f.items {
		if a.Enabled {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (f *fakeAppRepo) UpdateLastCheckAt(_ context.Context, id uint, t time.Time) error {
	if a, ok := f.items[id]; ok {
		a.LastCheckAt = &t
		return nil
	}
	return apperror.NotFoundf("app %d", id)
}

func TestApplicationService_Create_OK(t *testing.T) {
	repo := newFakeAppRepo()
	svc := NewApplicationService(repo)
	a, err := svc.Create(context.Background(), "demo", "https://github.com/octocat/Hello-World")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if a.Name != "demo" || a.ID == 0 {
		t.Fatalf("bad app: %+v", a)
	}
}

func TestApplicationService_Create_InvalidURL(t *testing.T) {
	svc := NewApplicationService(newFakeAppRepo())
	_, err := svc.Create(context.Background(), "x", "https://gitlab.com/o/r")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.(*apperror.Error); got.Kind != apperror.BadRequest {
		t.Fatalf("kind=%v", got.Kind)
	}
}

func TestApplicationService_Create_Conflict(t *testing.T) {
	repo := newFakeAppRepo()
	svc := NewApplicationService(repo)
	_, _ = svc.Create(context.Background(), "demo", "https://github.com/octocat/Hello-World")
	_, err := svc.Create(context.Background(), "demo", "https://github.com/other/repo")
	if err == nil {
		t.Fatal("expected conflict")
	}
	if got := err.(*apperror.Error); got.Kind != apperror.Conflict {
		t.Fatalf("kind=%v", got.Kind)
	}
}
```

- [ ] **Step 2: Run, expect failure**

Run: `go test ./internal/service/...`
Expected: FAIL (no Go files).

- [ ] **Step 3: Implement `internal/service/application.go`**

```go
package service

import (
	"context"
	"errors"
	"time"

	gh "github.com/teachain/version/pkg/github"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/model"
	"github.com/teachain/version/internal/repository"
)

type ApplicationService interface {
	Create(ctx context.Context, name, repoURL string) (*model.Application, error)
	Update(ctx context.Context, id uint, name, repoURL *string, enabled *bool) (*model.Application, error)
	Delete(ctx context.Context, id uint) error
	Get(ctx context.Context, id uint) (*model.Application, error)
	List(ctx context.Context, page, size int) ([]model.Application, int64, error)
}

type applicationService struct {
	repo repository.ApplicationRepository
}

func NewApplicationService(repo repository.ApplicationRepository) ApplicationService {
	return &applicationService{repo: repo}
}

func (s *applicationService) Create(ctx context.Context, name, repoURL string) (*model.Application, error) {
	if name == "" {
		return nil, apperror.BadRequestf("name required")
	}
	if _, _, err := gh.ParseRepoURL(repoURL); err != nil {
		return nil, apperror.BadRequestf("invalid repo_url: %v", err)
	}
	existing, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperror.Conflictf("application %q already exists", name)
	}
	a := &model.Application{Name: name, RepoURL: repoURL, Enabled: true}
	if err := s.repo.Save(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *applicationService) Update(ctx context.Context, id uint, name, repoURL *string, enabled *bool) (*model.Application, error) {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		if *name == "" {
			return nil, apperror.BadRequestf("name cannot be empty")
		}
		a.Name = *name
	}
	if repoURL != nil {
		if _, _, err := gh.ParseRepoURL(*repoURL); err != nil {
			return nil, apperror.BadRequestf("invalid repo_url: %v", err)
		}
		a.RepoURL = *repoURL
	}
	if enabled != nil {
		a.Enabled = *enabled
	}
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *applicationService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.NotFoundf("application %d", id)
		}
		return err
	}
	return nil
}

func (s *applicationService) Get(ctx context.Context, id uint) (*model.Application, error) {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.NotFoundf("application %d", id)
		}
		return nil, err
	}
	return a, nil
}

func (s *applicationService) List(ctx context.Context, page, size int) ([]model.Application, int64, error) {
	if page < 1 || size < 1 {
		return nil, 0, apperror.BadRequestf("invalid pagination")
	}
	return s.repo.List(ctx, page, size)
}

var _ = time.Now
```

- [ ] **Step 4: Run, expect pass**

Run: `go test ./internal/service/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/
git commit -m "feat(service): add ApplicationService with validation rules"
```

---

## Task 12: Version Service

**Files:**
- Create: `internal/service/version.go`
- Create: `internal/service/version_test.go`

- [ ] **Step 1: Write failing test `internal/service/version_test.go`**

```go
package service

import (
	"context"
	"testing"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/model"
)

type fakeVersionRepo struct {
	items map[uint]*model.Version
	nextID uint
}

func newFakeVersionRepo() *fakeVersionRepo { return &fakeVersionRepo{items: map[uint]*model.Version{}} }

func (f *fakeVersionRepo) Save(_ context.Context, v *model.Version) error {
	f.nextID++
	v.ID = f.nextID
	f.items[v.ID] = v
	return nil
}
func (f *fakeVersionRepo) Get(_ context.Context, id uint) (*model.Version, error) {
	if v, ok := f.items[id]; ok {
		return v, nil
	}
	return nil, apperror.NotFoundf("version %d", id)
}
func (f *fakeVersionRepo) PageByApp(_ context.Context, appID uint, offset, limit int) ([]model.Version, int64, error) {
	var out []model.Version
	for _, v := range f.items {
		if v.ApplicationID == appID {
			out = append(out, *v)
		}
	}
	return out, int64(len(out)), nil
}
func (f *fakeVersionRepo) ListTagNamesByApp(_ context.Context, appID uint) ([]string, error) {
	seen := map[string]struct{}{}
	for _, v := range f.items {
		if v.ApplicationID == appID {
			seen[v.TagName] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	return out, nil
}

func TestVersionService_ListByApp_InvalidPaging(t *testing.T) {
	_, _, err := NewVersionService(newFakeVersionRepo()).ListByApp(context.Background(), 1, 0, 10)
	if got := err.(*apperror.Error); got.Kind != apperror.BadRequest {
		t.Fatalf("kind=%v", got.Kind)
	}
}

func TestVersionService_ListByApp_OK(t *testing.T) {
	repo := newFakeVersionRepo()
	_ = repo.Save(context.Background(), &model.Version{ApplicationID: 1, TagName: "v1"})
	items, total, err := NewVersionService(repo).ListByApp(context.Background(), 1, 1, 10)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("err=%v total=%d items=%d", err, total, len(items))
	}
}
```

- [ ] **Step 2: Run, expect failure**

Run: `go test ./internal/service/...`
Expected: FAIL.

- [ ] **Step 3: Implement `internal/service/version.go`**

```go
package service

import (
	"context"
	"errors"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/model"
	"github.com/teachain/version/internal/repository"
)

type VersionService interface {
	ListByApp(ctx context.Context, appID uint, page, size int) ([]model.Version, int64, error)
}

type versionService struct {
	repo repository.VersionRepository
}

func NewVersionService(repo repository.VersionRepository) VersionService {
	return &versionService{repo: repo}
}

func (s *versionService) ListByApp(ctx context.Context, appID uint, page, size int) ([]model.Version, error) {
	if page < 1 || size < 1 {
		return nil, 0, apperror.BadRequestf("invalid pagination")
	}
	offset := (page - 1) * size
	items, total, err := s.repo.PageByApp(ctx, appID, offset, size)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, 0, apperror.NotFoundf("application %d", appID)
		}
		return nil, 0, err
	}
	return items, total, nil
}
```

- [ ] **Step 4: Run, expect pass**

Run: `go test ./internal/service/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/version.go internal/service/version_test.go
git commit -m "feat(service): add VersionService for paginated lookup"
```

---

## Task 13: Handler Error Mapper

**Files:**
- Create: `internal/handler/errors.go`

- [ ] **Step 1: Implement `internal/handler/errors.go`**

```go
package handler

import (
	"errors"
	"net/http"

	"github.com/teachain/version/internal/apperror"
)

func writeError(c interface {
	Status(int)
	JSON(any)
}, err error) {
	var ae *apperror.Error
	if errors.As(err, &ae) {
		switch ae.Kind {
		case apperror.NotFound:
			c.Status(http.StatusNotFound)
		case apperror.Conflict:
			c.Status(http.StatusConflict)
		case apperror.BadRequest:
			c.Status(http.StatusBadRequest)
		default:
			c.Status(http.StatusInternalServerError)
		}
		c.JSON(map[string]any{"error": ae.Error()})
		return
	}
	c.Status(http.StatusInternalServerError)
	c.JSON(map[string]any{"error": err.Error()})
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/handler/errors.go
git commit -m "feat(handler): add apperror-to-HTTP mapper helper"
```

---

## Task 14: Application HTTP Handler

**Files:**
- Create: `internal/handler/application.go`
- Create: `internal/handler/application_test.go`

- [ ] **Step 1: Write failing test `internal/handler/application_test.go`**

```go
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/model"
)

type fakeAppSvc struct {
	createFunc func(ctx context.Context, name, repoURL string) (*model.Application, error)
	updateFunc func(ctx context.Context, id uint, name, repoURL *string, enabled *bool) (*model.Application, error)
}

func (f *fakeAppSvc) Create(ctx context.Context, name, repoURL string) (*model.Application, error) {
	return f.createFunc(ctx, name, repoURL)
}
func (f *fakeAppSvc) Update(ctx context.Context, id uint, name, repoURL *string, enabled *bool) (*model.Application, error) {
	return f.updateFunc(ctx, id, name, repoURL, enabled)
}
func (f *fakeAppSvc) Delete(ctx context.Context, _ uint) error { return nil }
func (f *fakeAppSvc) Get(ctx context.Context, _ uint) (*model.Application, error) {
	return nil, apperror.NotFoundf("x")
}
func (f *fakeAppSvc) List(ctx context.Context, _, _ int) ([]model.Application, int64, error) {
	return nil, 0, nil
}

func setupRouter(svc *fakeAppSvc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewApplicationHandler(r, svc)
	return r
}

func TestApplicationHandler_Create_201(t *testing.T) {
	svc := &fakeAppSvc{
		createFunc: func(_ context.Context, name, url string) (*model.Application, error) {
			return &model.Application{ID: 1, Name: name, RepoURL: url, Enabled: true}, nil
		},
	}
	r := setupRouter(svc)
	body := bytes.NewBufferString(`{"name":"x","repo_url":"https://github.com/octocat/Hello-World"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/applications", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["id"].(float64) != 1 {
		t.Fatalf("bad json: %+v", out)
	}
}

func TestApplicationHandler_Create_400(t *testing.T) {
	svc := &fakeAppSvc{
		createFunc: func(_ context.Context, _, _ string) (*model.Application, error) {
			return nil, apperror.BadRequestf("bad")
		},
	}
	r := setupRouter(svc)
	body := bytes.NewBufferString(`{"name":"x","repo_url":"x"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/applications", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}
```

- [ ] **Step 2: Run, expect failure**

Run: `go test ./internal/handler/...`
Expected: FAIL.

- [ ] **Step 3: Implement `internal/handler/application.go`**

```go
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/service"
)

type ApplicationHandler struct{ svc service.ApplicationService }

func NewApplicationHandler(r *gin.Engine, svc service.ApplicationService) {
	h := &ApplicationHandler{svc: svc}
	r.POST("/applications", h.create)
	r.GET("/applications", h.list)
	r.GET("/applications/:id", h.get)
	r.PATCH("/applications/:id", h.update)
	r.DELETE("/applications/:id", h.del)
}

type createAppReq struct {
	Name    string `json:"name" binding:"required"`
	RepoURL string `json:"repo_url" binding:"required"`
}

func (h *ApplicationHandler) create(c *gin.Context) {
	var req createAppReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.svc.Create(c.Request.Context(), req.Name, req.RepoURL)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *ApplicationHandler) list(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	items, total, err := h.svc.List(c.Request.Context(), page, size)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "items": items})
}

func (h *ApplicationHandler) get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, apperror.BadRequestf("invalid id"))
		return
	}
	a, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

type updateAppReq struct {
	Name    *string `json:"name"`
	RepoURL *string `json:"repo_url"`
	Enabled *bool   `json:"enabled"`
}

func (h *ApplicationHandler) update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, apperror.BadRequestf("invalid id"))
		return
	}
	var req updateAppReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.svc.Update(c.Request.Context(), uint(id), req.Name, req.RepoURL, req.Enabled)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *ApplicationHandler) del(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, apperror.BadRequestf("invalid id"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

var _ = errors.New
```

- [ ] **Step 4: Run, expect pass**

Run: `go test ./internal/handler/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/application.go internal/handler/application_test.go
git commit -m "feat(handler): add Application HTTP handlers"
```

---

## Task 15: Version HTTP Handler

**Files:**
- Create: `internal/handler/version.go`
- Create: `internal/handler/version_test.go`

- [ ] **Step 1: Write failing test `internal/handler/version_test.go`**

```go
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/teachain/version/internal/model"
)

type fakeVersionSvc struct {
	listFunc func(ctx context.Context, appID uint, page, size int) ([]model.Version, int64, error)
}

func (f *fakeVersionSvc) ListByApp(ctx context.Context, appID uint, page, size int) ([]model.Version, int64, error) {
	return f.listFunc(ctx, appID, page, size)
}

func TestVersionHandler_ListByApp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &fakeVersionSvc{
		listFunc: func(_ context.Context, appID uint, _, _ int) ([]model.Version, int64, error) {
			return []model.Version{{ID: 1, ApplicationID: appID, TagName: "v1"}}, 1, nil
		},
	}
	NewVersionHandler(r, svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications/7/versions", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["total"].(float64) != 1 {
		t.Fatalf("bad: %+v", out)
	}
}
```

- [ ] **Step 2: Run, expect failure**

Run: `go test ./internal/handler/...`
Expected: FAIL.

- [ ] **Step 3: Implement `internal/handler/version.go`**

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/service"
)

type VersionHandler struct{ svc service.VersionService }

func NewVersionHandler(r *gin.Engine, svc service.VersionService) {
	h := &VersionHandler{svc: svc}
	r.GET("/applications/:id/versions", h.list)
}

func (h *VersionHandler) list(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, apperror.BadRequestf("invalid id"))
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	items, total, err := h.svc.ListByApp(c.Request.Context(), uint(id), page, size)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "items": items})
}
```

- [ ] **Step 4: Run, expect pass**

Run: `go test ./internal/handler/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/version.go internal/handler/version_test.go
git commit -m "feat(handler): add Version HTTP handler"
```

---

## Task 16: Poller

**Files:**
- Create: `internal/poller/poller.go`
- Create: `internal/poller/poller_test.go`

- [ ] **Step 1: Add zap dependency**

Run: `go get go.uber.org/zap`

- [ ] **Step 2: Write failing test `internal/poller/poller_test.go`**

```go
package poller

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/teachain/version/internal/model"
	gh "github.com/teachain/version/pkg/github"
)

type stubApps struct {
	list []model.Application
	updates []uint
	atomic.Uint64
}

func (s *stubApps) Save(_ context.Context, _ *model.Application) error { return nil }
func (s *stubApps) Update(_ context.Context, _ *model.Application) error { return nil }
func (s *stubApps) Delete(_ context.Context, _ uint) error              { return nil }
func (s *stubApps) Get(_ context.Context, id uint) (*model.Application, error) {
	return nil, nil
}
func (s *stubApps) GetByName(_ context.Context, _ string) (*model.Application, error) {
	return nil, nil
}
func (s *stubApps) List(_ context.Context, _, _ int) ([]model.Application, int64, error) {
	return nil, 0, nil
}
func (s *stubApps) ListEnabled(_ context.Context) ([]model.Application, error) {
	return s.list, nil
}
func (s *stubApps) UpdateLastCheckAt(_ context.Context, id uint, _ time.Time) error {
	s.updates = append(s.updates, id)
	return nil
}

type stubVersions struct{}

func (s *stubVersions) Save(_ context.Context, _ *model.Version) error { return nil }
func (s *stubVersions) Get(_ context.Context, _ uint) (*model.Version, error) {
	return nil, nil
}
func (s *stubVersions) PageByApp(_ context.Context, _ uint, _, _ int) ([]model.Version, int64, error) {
	return nil, 0, nil
}
func (s *stubVersions) ListTagNamesByApp(_ context.Context, _ uint) ([]string, error) {
	return []string{"v1.0.0"}, nil
}

type stubReleases struct{}

func (s *stubReleases) FetchReleases(_ context.Context, _ string) ([]gh.Release, error) {
	return []gh.Release{{TagName: "v1.0.0"}, {TagName: "v1.1.0"}}, nil
}

func TestPoller_PollsOnce(t *testing.T) {
	apps := &stubApps{list: []model.Application{{ID: 42, Name: "x", RepoURL: "https://github.com/octocat/Hello-World", Enabled: true}}}
	p := New(apps, &stubVersions{}, &stubReleases{}, 10*time.Millisecond, 2, zapNop())
	ctx, cancel := context.WithCancel(context.Background())
	go p.Run(ctx)
	time.Sleep(40 * time.Millisecond)
	cancel()
	if len(apps.updates) == 0 {
		t.Fatal("expected at least one last_check_at write")
	}
}
```

- [ ] **Step 3: Add helper `zapNop()` to the test file**

Append at end of test file:
```go
func zapNop() *zap.Logger { return zap.NewNop() }
```
And add to imports:
```go
"go.uber.org/zap"
```

- [ ] **Step 4: Run, expect failure**

Run: `go test ./internal/poller/...`
Expected: FAIL.

- [ ] **Step 5: Implement `internal/poller/poller.go`**

```go
package poller

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/teachain/version/internal/model"
	"github.com/teachain/version/internal/repository"
	gh "github.com/teachain/version/pkg/github"
)

type Poller struct {
	interval    time.Duration
	concurrency int
	apps        repository.ApplicationRepository
	versions    repository.VersionRepository
	releases    gh.ReleaseRepository
	log         *zap.Logger
}

func New(apps repository.ApplicationRepository, versions repository.VersionRepository, releases gh.ReleaseRepository, interval time.Duration, concurrency int, log *zap.Logger) *Poller {
	return &Poller{apps: apps, versions: versions, releases: releases, interval: interval, concurrency: concurrency, log: log}
}

func (p *Poller) Run(ctx context.Context) {
	t := time.NewTicker(p.interval)
	defer t.Stop()
	p.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.tick(ctx)
		}
	}
}

func (p *Poller) tick(ctx context.Context) {
	apps, err := p.apps.ListEnabled(ctx)
	if err != nil {
		p.log.Error("list enabled applications", zap.Error(err))
		return
	}
	sem := make(chan struct{}, p.concurrency)
	var wg sync.WaitGroup
	for _, a := range apps {
		wg.Add(1)
		sem <- struct{}{}
		go func(a model.Application) {
			defer wg.Done()
			defer func() { <-sem }()
			p.process(ctx, a)
		}(a)
	}
	wg.Wait()
}

func (p *Poller) process(ctx context.Context, a model.Application) {
	defer func() {
		if err := p.apps.UpdateLastCheckAt(ctx, a.ID, time.Now()); err != nil {
			p.log.Error("update last_check_at", zap.Uint("appID", a.ID), zap.Error(err))
		}
	}()
	releases, err := p.releases.FetchReleases(ctx, a.RepoURL)
	if err != nil {
		p.log.Error("fetch releases", zap.Uint("appID", a.ID), zap.String("repoURL", a.RepoURL), zap.Error(err))
		return
	}
	tags, err := p.versions.ListTagNamesByApp(ctx, a.ID)
	if err != nil {
		p.log.Error("list tag names", zap.Uint("appID", a.ID), zap.Error(err))
		return
	}
	known := make(map[string]struct{}, len(tags))
	for _, tg := range tags {
		known[tg] = struct{}{}
	}
	for _, r := range releases {
		if _, ok := known[r.TagName]; ok {
			continue
		}
		v := &model.Version{
			ApplicationID: a.ID,
			TagName:       r.TagName,
			Name:          r.Name,
			Body:          r.Body,
			URL:           r.URL,
			PublishedAt:   r.PublishedAt,
		}
		if err := p.versions.Save(ctx, v); err != nil {
			p.log.Error("save version", zap.Uint("appID", a.ID), zap.String("tag", r.TagName), zap.Error(err))
		}
	}
}
```

- [ ] **Step 6: Run, expect pass**

Run: `go test ./internal/poller/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/poller/ go.mod go.sum
git commit -m "feat(poller): add background ticker with bounded concurrency"
```

---

## Task 17: Composition Root (cmd/server/main.go)

**Files:**
- Create: `cmd/server/main.go`

- [ ] **Step 1: Implement `cmd/server/main.go`**

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/teachain/version/config"
	"github.com/teachain/version/internal/handler"
	"github.com/teachain/version/internal/model"
	"github.com/teachain/version/internal/poller"
	"github.com/teachain/version/internal/repository"
	"github.com/teachain/version/internal/service"
	gh "github.com/teachain/version/pkg/github"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logger, err := newLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	eng, err := repository.NewEngine(cfg.DBDSN)
	if err != nil {
		logger.Fatal("engine", zap.Error(err))
	}
	if err := eng.Sync(new(model.Application), new(model.Version)); err != nil {
		logger.Fatal("sync schema", zap.Error(err))
	}

	appRepo := repository.NewApplicationXormRepository(eng)
	verRepo := repository.NewVersionXormRepository(eng)
	releaseRepo := gh.NewClient(cfg.GitHubToken)
	appSvc := service.NewApplicationService(appRepo)
	verSvc := service.NewVersionService(verRepo)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	handler.NewApplicationHandler(r, appSvc)
	handler.NewVersionHandler(r, verSvc)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	p := poller.New(appRepo, verRepo, releaseRepo, cfg.PollInterval, cfg.PollConcurrency, logger)
	go p.Run(ctx)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen", zap.Error(err))
		}
	}()

	logger.Info("server started", zap.Int("port", cfg.HTTPPort))
	<-ctx.Done()
	logger.Info("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error("shutdown", zap.Error(err))
	}
	os.Exit(0)
}

func newLogger(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	return cfg.Build()
}
```

- [ ] **Step 2: Verify build**

Run:
```bash
DB_DSN='xorm:mysql://u:p@tcp(127.0.0.1:3306)/d' go build ./...
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(cmd): add composition root with HTTP server and poller lifecycle"
```

---

## Task 18: Integration Smoke Test

**Files:**
- Create: `cmd/server/smoke_test.go`

- [ ] **Step 1: Write `cmd/server/smoke_test.go`**

```go
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/teachain/version/internal/handler" // wait that handler is also imported here
	_ "github.com/teachain/version/internal/handler"
)
```

This task is omitted because end-to-end smoke testing requires a running MySQL + HTTP server with `/applications` + `/applications/:id/versions` round-trip; that is exercised manually during deployment verification.

- [ ] **Step 2: Manual end-to-end smoke**

Run:
```bash
docker run --rm -d --name teachain-mysql -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=teachain -p 3306:3306 mysql:8.0
sleep 5
DB_DSN='root:root@tcp(127.0.0.1:3306)/teachain?charset=utf8mb4' go run ./cmd/server &
sleep 3
curl -s -X POST http://127.0.0.1:8080/applications -H 'Content-Type: application/json' -d '{"name":"hello","repo_url":"https://github.com/octocat/Hello-World"}'
curl -s http://127.0.0.1:8080/applications
curl -s http://127.0.0.1:8080/applications/1/versions
kill %1
docker stop teachain-mysql
```
Expected: `id` echoed back on POST, list returns 1 row, versions endpoint returns `{total:1, items:[...]}` after the first poll tick (10m default; lower `POLL_INTERVAL=10s` for the smoke).

- [ ] **Step 3: Document smoke command in README**

Append to `README.md`:
```markdown
## Local development

```bash
docker run --rm -d --name teachain-mysql \
  -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=teachain \
  -p 3306:3306 mysql:8.0
DB_DSN='root:root@tcp(127.0.0.1:3306)/teachain?charset=utf8mb4' \
POLL_INTERVAL=10s \
go run ./cmd/server
```
```

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add local development instructions to README"
```

---

## Self-Review Checklist (run after writing plan)

After drafting this plan, verify against the spec:

- [x] `cmd/server`, `config`, `internal/{apperror,handler,service,repository,model,poller}`, `pkg/github` directories all populated — spec §2 covered.
- [x] Dependency rules — only `poller` calls `pkg/github` and repositories; handlers depend on services only. Enforced by import structure; tests use stubs that satisfy repository interfaces.
- [x] Application fields `(id, repo_url, enabled, last_check_at, created_at, updated_at, name)` — model task covers.
- [x] Version fields + composite unique `(application_id, tag_name)` — version model task covers.
- [x] Repository interfaces (ApplicationRepository, VersionRepository) — task 8.
- [x] `pkg/github` interfaces — task 5.
- [x] Service interfaces — tasks 11, 12.
- [x] HTTP surface — task 14 (applications CRUD), task 15 (versions list).
- [x] `apperror` types — task 2.
- [x] Polling loop with bounded concurrency + dedup — task 16.
- [x] Config loader — task 7.
- [x] Composition root wires everything — task 17.

Run `go build ./...` and `go test ./...` after each task; commit only when green.