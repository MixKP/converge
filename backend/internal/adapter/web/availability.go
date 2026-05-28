package web

import (
	"context"
	"errors"
	"net/http"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/service"

	"github.com/gin-gonic/gin"
)

type availabilityService interface {
	GetActiveTeachers(ctx context.Context) ([]models.Teacher, error)
	SubmitWeeklyAvailability(ctx context.Context, payload models.AvailabilityPayload) error
}

type AvailabilityHandler struct {
	svc availabilityService
}

func NewAvailabilityHandler(svc availabilityService) *AvailabilityHandler {
	return &AvailabilityHandler{svc: svc}
}

func (h *AvailabilityHandler) GetTeachers(c *gin.Context) {
	teachers, err := h.svc.GetActiveTeachers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve teachers"})
		return
	}
	c.JSON(http.StatusOK, teachers)
}

func (h *AvailabilityHandler) SubmitWeeklyAvailability(c *gin.Context) {
	var payload models.AvailabilityPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.SubmitWeeklyAvailability(c.Request.Context(), payload); err != nil {
		var valErr *service.ValidationError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save availability"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Availability saved successfully"})
}
