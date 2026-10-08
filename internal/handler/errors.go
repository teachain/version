package handler

import (
	"errors"
	"net/http"

	"github.com/teachain/version/internal/apperror"
)

func writeError(c interface {
	Status(int)
	JSON(int, any)
}, err error) {
	var ae *apperror.Error
	if errors.As(err, &ae) {
		var status int
		switch ae.Kind {
		case apperror.NotFound:
			status = http.StatusNotFound
		case apperror.Conflict:
			status = http.StatusConflict
		case apperror.BadRequest:
			status = http.StatusBadRequest
		default:
			status = http.StatusInternalServerError
		}
		c.JSON(status, map[string]any{"error": ae.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
}