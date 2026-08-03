package web

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/gin-gonic/gin"
)

func requestID(c *gin.Context) string {
	return shared.RequestIDFromContext(c.Request.Context())
}

func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// respondFieldUpdateErr writes the appropriate HTTP response for an error
// returned by a single-field update service call (e.g. SetStatus, SetGender,
// SetCapacity), dispatching on error type: NotFoundError -> 404,
// ValidationError -> 400, anything else -> 500. entity is a lowercase noun
// (e.g. "teacher", "branch") used for both the log key (entity_id) and the
// not-found log message.
func respondFieldUpdateErr(c *gin.Context, logger *slog.Logger, err error, op, entity string, entityID int, genericMsg string) {
	entityLogKey := entity + "_id"
	var notFoundErr *shared.NotFoundError
	var valErr *shared.ValidationError
	switch {
	case errors.As(err, &notFoundErr):
		logger.Warn(entity+" not found",
			"request_id", requestID(c),
			"op", op,
			entityLogKey, entityID,
		)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.As(err, &valErr):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		logger.Error("request failed",
			"request_id", requestID(c),
			"op", op,
			entityLogKey, entityID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": genericMsg})
	}
}

type MessageResponse struct {
	Message string `json:"message" example:"Availability saved successfully"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"failed to retrieve teachers"`
}
