package handler

import (
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

type fakeVersionSvc struct {
	listFunc func(ctx context.Context, appID uint, page, size int) ([]model.Version, int64, error)
}

func (f *fakeVersionSvc) ListByApp(ctx context.Context, appID uint, page, size int) ([]model.Version, int64, error) {
	return f.listFunc(ctx, appID, page, size)
}

func setupVersionRouter(svc *fakeVersionSvc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewVersionHandler(r, svc)
	return r
}

func TestVersionHandler_ListByApp(t *testing.T) {
	r := setupVersionRouter(&fakeVersionSvc{
		listFunc: func(_ context.Context, appID uint, _, _ int) ([]model.Version, int64, error) {
			return []model.Version{{ID: 1, ApplicationID: appID, TagName: "v1"}}, 1, nil
		},
	})
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

func TestVersionHandler_ListByApp_BadID(t *testing.T) {
	r := setupVersionRouter(&fakeVersionSvc{
		listFunc: func(_ context.Context, _ uint, _, _ int) ([]model.Version, int64, error) {
			t.Fatal("service should not be called for bad id")
			return nil, 0, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications/abc/versions", nil)
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

func TestVersionHandler_ListByApp_ClampPagination(t *testing.T) {
	var gotPage, gotSize int
	r := setupVersionRouter(&fakeVersionSvc{
		listFunc: func(_ context.Context, _ uint, page, size int) ([]model.Version, int64, error) {
			gotPage, gotSize = page, size
			return nil, 0, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications/1/versions?page=-1&size=0", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if gotPage != 1 || gotSize != 20 {
		t.Fatalf("clamp failed: page=%d size=%d", gotPage, gotSize)
	}
}

func TestVersionHandler_ListByApp_BadRequest(t *testing.T) {
	r := setupVersionRouter(&fakeVersionSvc{
		listFunc: func(_ context.Context, _ uint, _, _ int) ([]model.Version, int64, error) {
			return nil, 0, apperror.BadRequestf("bad")
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications/7/versions", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestVersionHandler_ListByApp_NotFound(t *testing.T) {
	r := setupVersionRouter(&fakeVersionSvc{
		listFunc: func(_ context.Context, _ uint, _, _ int) ([]model.Version, int64, error) {
			return nil, 0, apperror.NotFoundf("nope")
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications/7/versions", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestVersionHandler_ListByApp_InternalError(t *testing.T) {
	r := setupVersionRouter(&fakeVersionSvc{
		listFunc: func(_ context.Context, _ uint, _, _ int) ([]model.Version, int64, error) {
			return nil, 0, errors.New("boom")
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/applications/7/versions", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", w.Code)
	}
}
