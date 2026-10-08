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