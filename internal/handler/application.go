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