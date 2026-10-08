package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/model"
)

type fakeAppSvc struct {
	createFunc func(ctx context.Context, name, repoURL string) (*model.Application, error)
	updateFunc func(ctx context.Context, id uint, name, repoURL *string, enabled *bool) (*model.Application, error)
	getFunc    func(ctx context.Context, id uint) (*model.Application, error)
	listFunc   func(ctx context.Context, page, size int) ([]model.Application, int64, error)
	deleteFunc func(ctx context.Context, id uint) error
}

func (f *fakeAppSvc) Create(ctx context.Context, name, repoURL string) (*model.Application, error) {
	return f.createFunc(ctx, name, repoURL)
}
func (f *fakeAppSvc) Update(ctx context.Context, id uint, name, repoURL *string, enabled *bool) (*model.Application, error) {
	if f.updateFunc == nil {
		return nil, apperror.NotFoundf("update not configured")
	}
	return f.updateFunc(ctx, id, name, repoURL, enabled)
}
func (f *fakeAppSvc) Delete(ctx context.Context, id uint) error {
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, id)
	}
	return nil
}
func (f *fakeAppSvc) Get(ctx context.Context, id uint) (*model.Application, error) {
	if f.getFunc != nil {
		return f.getFunc(ctx, id)
	}
	return nil, apperror.NotFoundf("not configured")
}
func (f *fakeAppSvc) List(ctx context.Context, page, size int) ([]model.Application, int64, error) {
	if f.listFunc != nil {
		return f.listFunc(ctx, page, size)
	}
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

func TestApplicationHandler_Create_BadJSON(t *testing.T) {
	svc := &fakeAppSvc{
		createFunc: func(_ context.Context, _, _ string) (*model.Application, error) {
			t.Fatal("service should not be called when JSON bind fails")
			return nil, nil
		},
	}
	r := setupRouter(svc)
	body := bytes.NewBufferString(`{"repo_url":"https://github.com/o/r"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/applications", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if _, ok := out["error"]; !ok {
		t.Fatalf("expected error field, got %+v", out)
	}
}

func TestApplicationHandler_Create_409(t *testing.T) {
	svc := &fakeAppSvc{
		createFunc: func(_ context.Context, _, _ string) (*model.Application, error) {
			return nil, apperror.Conflictf("dup")
		},
	}
	r := setupRouter(svc)
	body := bytes.NewBufferString(`{"name":"x","repo_url":"https://github.com/o/r"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/applications", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestApplicationHandler_Create_500(t *testing.T) {
	svc := &fakeAppSvc{
		createFunc: func(_ context.Context, _, _ string) (*model.Application, error) {
			return nil, errors.New("boom")
		},
	}
	r := setupRouter(svc)
	body := bytes.NewBufferString(`{"name":"x","repo_url":"https://github.com/o/r"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/applications", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestApplicationHandler_List_200(t *testing.T) {
	svc := &fakeAppSvc{
		listFunc: func(_ context.Context, _, _ int) ([]model.Application, int64, error) {
			return []model.Application{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}, 2, nil
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["total"].(float64) != 2 {
		t.Fatalf("bad total: %+v", out)
	}
	items, ok := out["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("bad items: %+v", out)
	}
}

func TestApplicationHandler_List_ClampPagination(t *testing.T) {
	var gotPage, gotSize int
	svc := &fakeAppSvc{
		listFunc: func(_ context.Context, page, size int) ([]model.Application, int64, error) {
			gotPage, gotSize = page, size
			return nil, 0, nil
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications?page=0&size=999", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if gotPage != 1 || gotSize != 20 {
		t.Fatalf("clamp failed: page=%d size=%d", gotPage, gotSize)
	}
}

func TestApplicationHandler_List_BadRequest(t *testing.T) {
	svc := &fakeAppSvc{
		listFunc: func(_ context.Context, _, _ int) ([]model.Application, int64, error) {
			return nil, 0, apperror.BadRequestf("bad")
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestApplicationHandler_List_InternalError(t *testing.T) {
	svc := &fakeAppSvc{
		listFunc: func(_ context.Context, _, _ int) ([]model.Application, int64, error) {
			return nil, 0, errors.New("boom")
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestApplicationHandler_Get_200(t *testing.T) {
	svc := &fakeAppSvc{
		getFunc: func(_ context.Context, id uint) (*model.Application, error) {
			return &model.Application{ID: id, Name: "demo", RepoURL: "https://github.com/o/r", Enabled: true}, nil
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications/7", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["id"].(float64) != 7 {
		t.Fatalf("bad json: %+v", out)
	}
}

func TestApplicationHandler_Get_BadID(t *testing.T) {
	svc := &fakeAppSvc{}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications/abc", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if !strings.Contains(out["error"].(string), "invalid id") {
		t.Fatalf("bad error: %+v", out)
	}
}

func TestApplicationHandler_Get_NotFound(t *testing.T) {
	svc := &fakeAppSvc{
		getFunc: func(_ context.Context, _ uint) (*model.Application, error) {
			return nil, apperror.NotFoundf("nope")
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications/9", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestApplicationHandler_Get_InternalError(t *testing.T) {
	svc := &fakeAppSvc{
		getFunc: func(_ context.Context, _ uint) (*model.Application, error) {
			return nil, errors.New("boom")
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications/9", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestApplicationHandler_Update_200(t *testing.T) {
	enabled := false
	svc := &fakeAppSvc{
		updateFunc: func(_ context.Context, id uint, _, _ *string, _ *bool) (*model.Application, error) {
			return &model.Application{ID: id, Name: "x", RepoURL: "https://github.com/o/r", Enabled: enabled}, nil
		},
	}
	r := setupRouter(svc)
	body := bytes.NewBufferString(`{"name":"x","enabled":false}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/applications/3", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["id"].(float64) != 3 {
		t.Fatalf("bad json: %+v", out)
	}
}

func TestApplicationHandler_Update_BadID(t *testing.T) {
	svc := &fakeAppSvc{}
	r := setupRouter(svc)
	body := bytes.NewBufferString(`{}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/applications/abc", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestApplicationHandler_Update_BadJSON(t *testing.T) {
	svc := &fakeAppSvc{
		updateFunc: func(_ context.Context, _ uint, _, _ *string, _ *bool) (*model.Application, error) {
			t.Fatal("service should not be called when JSON bind fails")
			return nil, nil
		},
	}
	r := setupRouter(svc)
	body := bytes.NewBufferString(`{`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/applications/3", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestApplicationHandler_Update_NotFound(t *testing.T) {
	svc := &fakeAppSvc{
		updateFunc: func(_ context.Context, _ uint, _, _ *string, _ *bool) (*model.Application, error) {
			return nil, apperror.NotFoundf("nope")
		},
	}
	r := setupRouter(svc)
	body := bytes.NewBufferString(`{}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/applications/3", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestApplicationHandler_Delete_204(t *testing.T) {
	var deletedID uint
	svc := &fakeAppSvc{
		deleteFunc: func(_ context.Context, id uint) error {
			deletedID = id
			return nil
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/applications/5", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if deletedID != 5 {
		t.Fatalf("id=%d", deletedID)
	}
}

func TestApplicationHandler_Delete_BadID(t *testing.T) {
	svc := &fakeAppSvc{
		deleteFunc: func(_ context.Context, _ uint) error {
			t.Fatal("service should not be called for bad id")
			return nil
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/applications/abc", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestApplicationHandler_Delete_NotFound(t *testing.T) {
	svc := &fakeAppSvc{
		deleteFunc: func(_ context.Context, _ uint) error {
			return apperror.NotFoundf("nope")
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/applications/5", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d", w.Code)
	}
}
