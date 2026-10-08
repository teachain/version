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