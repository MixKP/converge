package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/RinZ5/converge/backend/internal/teacher"

	"github.com/gin-gonic/gin"
)

type teacherService interface {
	GetActiveTeachers(ctx context.Context) ([]teacher.Teacher, error)
	GetTeachersBySubject(ctx context.Context, subjectID int) ([]teacher.Teacher, error)
	GetAllAvailability(ctx context.Context) ([]teacher.TeacherAvailability, error)
	GetBranches(ctx context.Context) ([]shared.Branch, error)
	GetSubjects(ctx context.Context) ([]shared.Subject, error)
	SubmitWeeklyAvailability(ctx context.Context, teacherID int, slots []shared.WeeklySlot) error
}

type AvailabilityHandler struct {
	svc    teacherService
	logger *slog.Logger
}

func NewAvailabilityHandler(svc teacherService, logger *slog.Logger) *AvailabilityHandler {
	return &AvailabilityHandler{svc: svc, logger: logger}
}

func (h *AvailabilityHandler) requestID(c *gin.Context) string {
	return shared.RequestIDFromContext(c.Request.Context())
}

// GetTeachers godoc
// @Summary      List active teachers
// @Description  Returns all teachers with active status, ordered by name. Optionally filter by subject_id.
// @Tags         teachers
// @Produce      json
// @Param        subject_id  query  int  false  "Filter by subject ID"
// @Success      200  {array}  teacher.Teacher
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /teachers [get]
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
			h.logger.Error("request failed",
				"request_id", h.requestID(c),
				"op", "GetTeachers/GetTeachersBySubject",
				"subject_id", subjectID,
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve teachers"})
			return
		}
		c.JSON(http.StatusOK, teachers)
		return
	}

	teachers, err := h.svc.GetActiveTeachers(c.Request.Context())
	if err != nil {
		h.logger.Error("request failed",
			"request_id", h.requestID(c),
			"op", "GetTeachers/GetActiveTeachers",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve teachers"})
		return
	}
	c.JSON(http.StatusOK, teachers)
}

// GetAllAvailability godoc
// @Summary      Get all teacher availability
// @Description  Returns all active teachers with their weekly availability slots, grouped by teacher
// @Tags         availability
// @Produce      json
// @Success      200  {array}  teacher.TeacherAvailability
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /availability [get]
func (h *AvailabilityHandler) GetAllAvailability(c *gin.Context) {
	availability, err := h.svc.GetAllAvailability(c.Request.Context())
	if err != nil {
		h.logger.Error("request failed",
			"request_id", h.requestID(c),
			"op", "GetAllAvailability",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve availability"})
		return
	}
	c.JSON(http.StatusOK, availability)
}

// GetBranches godoc
// @Summary      List branches
// @Description  Returns all branches ordered by id
// @Tags         branches
// @Produce      json
// @Success      200  {array}  shared.Branch
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /branches [get]
func (h *AvailabilityHandler) GetBranches(c *gin.Context) {
	branches, err := h.svc.GetBranches(c.Request.Context())
	if err != nil {
		h.logger.Error("request failed",
			"request_id", h.requestID(c),
			"op", "GetBranches",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve branches"})
		return
	}
	c.JSON(http.StatusOK, branches)
}

// GetSubjects godoc
// @Summary      List subjects
// @Description  Returns all subjects ordered by id
// @Tags         subjects
// @Produce      json
// @Success      200  {array}  shared.Subject
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /subjects [get]
func (h *AvailabilityHandler) GetSubjects(c *gin.Context) {
	subjects, err := h.svc.GetSubjects(c.Request.Context())
	if err != nil {
		h.logger.Error("request failed",
			"request_id", h.requestID(c),
			"op", "GetSubjects",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subjects"})
		return
	}
	c.JSON(http.StatusOK, subjects)
}

// SubmitWeeklyAvailability godoc
// @Summary      Submit weekly availability
// @Description  Replaces all existing availability for a teacher with the provided weekly schedule (full replace, not merge). Rejects overlapping slots within the same day.
// @Tags         availability
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]interface{}  true  "Weekly availability payload"
// @Success      201  {object}  scheduling.MessageResponse
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /availability [post]
func (h *AvailabilityHandler) SubmitWeeklyAvailability(c *gin.Context) {
	var payload struct {
		TeacherID int                 `json:"teacher_id"`
		Weekly    []shared.WeeklySlot `json:"weekly"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var valErr *teacher.ValidationError
	if err := h.svc.SubmitWeeklyAvailability(c.Request.Context(), payload.TeacherID, payload.Weekly); err != nil {
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			h.logger.Error("request failed",
				"request_id", h.requestID(c),
				"op", "SubmitWeeklyAvailability",
				"teacher_id", payload.TeacherID,
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save availability"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Availability saved successfully"})
}
