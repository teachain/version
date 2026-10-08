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