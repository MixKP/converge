package web

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/service"

	"github.com/gin-gonic/gin"
)

type availabilityService interface {
	GetActiveTeachers(ctx context.Context) ([]models.Teacher, error)
	GetTeachersBySubject(ctx context.Context, subjectID int) ([]models.Teacher, error)
	GetAllAvailability(ctx context.Context) ([]models.TeacherAvailability, error)
	GetBranches(ctx context.Context) ([]models.Branch, error)
	GetSubjects(ctx context.Context) ([]models.Subject, error)
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
//	@Description	Returns all teachers with active status, ordered by name. Optionally filter by subject_id.
//	@Tags			teachers
//	@Produce		json
//	@Param			subject_id	query		int		false	"Filter by subject ID"
//	@Success		200			{array}		models.Teacher
//	@Failure		400			{object}	models.ErrorResponse
//	@Failure		500			{object}	models.ErrorResponse
//	@Router			/teachers [get]
func (h *AvailabilityHandler) GetTeachers(c *gin.Context) {
	subjectIDStr := c.Query("subject_id")
	if subjectIDStr != "" {
		subjectID, err := strconv.Atoi(subjectIDStr)
		if err != nil || subjectID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
			return
		}
		teachers, err := h.svc.GetTeachersBySubject(c.Request.Context(), subjectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve teachers"})
			return
		}
		c.JSON(http.StatusOK, teachers)
		return
	}

	teachers, err := h.svc.GetActiveTeachers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve teachers"})
		return
	}
	c.JSON(http.StatusOK, teachers)
}

// GetAllAvailability godoc
//
//	@Summary		Get all teacher availability
//	@Description	Returns all active teachers with their weekly availability slots, grouped by teacher
//	@Tags			availability
//	@Produce		json
//	@Success		200	{array}		models.TeacherAvailability
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/availability [get]
func (h *AvailabilityHandler) GetAllAvailability(c *gin.Context) {
	availability, err := h.svc.GetAllAvailability(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve availability"})
		return
	}
	c.JSON(http.StatusOK, availability)
}

// GetBranches godoc
//
//	@Summary		List branches
//	@Description	Returns all branches ordered by id
//	@Tags			branches
//	@Produce		json
//	@Success		200	{array}		models.Branch
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/branches [get]
func (h *AvailabilityHandler) GetBranches(c *gin.Context) {
	branches, err := h.svc.GetBranches(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve branches"})
		return
	}
	c.JSON(http.StatusOK, branches)
}

// GetSubjects godoc
//
//	@Summary		List subjects
//	@Description	Returns all subjects ordered by id
//	@Tags			subjects
//	@Produce		json
//	@Success		200	{array}		models.Subject
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/subjects [get]
func (h *AvailabilityHandler) GetSubjects(c *gin.Context) {
	subjects, err := h.svc.GetSubjects(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subjects"})
		return
	}
	c.JSON(http.StatusOK, subjects)
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
//	@Router			/availability [post]
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
