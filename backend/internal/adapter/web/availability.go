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

// GetTeachers godoc
//
//	@Summary		List active teachers
//	@Description	Returns all teachers with active status, ordered by name
//	@Tags			teachers
//	@Produce		json
//	@Success		200	{array}		models.Teacher
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/api/teachers [get]
func (h *AvailabilityHandler) GetTeachers(c *gin.Context) {
	teachers, err := h.svc.GetActiveTeachers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve teachers"})
		return
	}
	c.JSON(http.StatusOK, teachers)
}

// SubmitWeeklyAvailability godoc
//
//	@Summary		Submit weekly availability
//	@Description	Replaces all existing availability for a teacher with the provided weekly schedule (full replace, not merge). Rejects overlapping slots within the same day.
//	@Tags			availability
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.AvailabilityPayload	true	"Weekly availability payload. day_of_week: 0=Monday through 6=Sunday"
//	@Success		201		{object}	models.MessageResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/api/availability [post]
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
