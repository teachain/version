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