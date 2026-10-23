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
	GetAllTeachers(ctx context.Context) ([]teacher.Teacher, error)
	GetInactiveTeachers(ctx context.Context) ([]teacher.Teacher, error)
	GetTeachersBySubject(ctx context.Context, subjectID int) ([]teacher.Teacher, error)
	GetAllAvailability(ctx context.Context) ([]teacher.TeacherAvailability, error)
	GetSubjects(ctx context.Context) ([]shared.Subject, error)
	SubmitWeeklyAvailability(ctx context.Context, teacherID int, slots []shared.WeeklySlot) (int, error)
	AddTeacher(ctx context.Context, name, email, gender string) (*teacher.Teacher, error)
	SetStatus(ctx context.Context, teacherID int, status string) error
	SetGender(ctx context.Context, teacherID int, gender string) error
	SubjectsForTeacher(ctx context.Context, teacherID int) ([]shared.Subject, error)
	SetTeacherSubjects(ctx context.Context, teacherID int, subjectIDs []int) error
}

type AvailabilityHandler struct {
	svc    teacherService
	logger *slog.Logger
}

func NewAvailabilityHandler(svc teacherService, logger *slog.Logger) *AvailabilityHandler {
	return &AvailabilityHandler{svc: svc, logger: logger}
}

// GetTeachers godoc
// GetTeachers godoc
// @Summary      List teachers
// @Description  Returns teachers ordered by name. Filter by status (active|deactivated|all, default all). Optionally filter by subject_id, which returns active teachers only and ignores status.
// @Tags         teachers
// @Produce      json
// @Param        subject_id  query  int     false  "Filter by subject ID (active teachers only)"
// @Param        status      query  string  false  "Filter by status"  Enums(active, deactivated, all)
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
				"request_id", requestID(c),
				"op", "GetTeachers/GetTeachersBySubject",
				"subject_id", subjectID,
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve teachers"})
			return
		}
		c.JSON(http.StatusOK, orEmpty(teachers))
		return
	}

	status := c.Query("status")
	var (
		teachers []teacher.Teacher
		err      error
	)
	switch status {
	case "", "all":
		teachers, err = h.svc.GetAllTeachers(c.Request.Context())
	case "active":
		teachers, err = h.svc.GetActiveTeachers(c.Request.Context())
	case "deactivated":
		teachers, err = h.svc.GetInactiveTeachers(c.Request.Context())
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status: must be active, deactivated, or all"})
		return
	}
	if err != nil {
		h.logger.Error("request failed",
			"request_id", requestID(c),
			"op", "GetTeachers",
			"status", status,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve teachers"})
		return
	}
	c.JSON(http.StatusOK, orEmpty(teachers))
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
			"request_id", requestID(c),
			"op", "GetAllAvailability",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve availability"})
		return
	}
	c.JSON(http.StatusOK, availability)
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
			"request_id", requestID(c),
			"op", "GetSubjects",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subjects"})
		return
	}
	c.JSON(http.StatusOK, subjects)
}

// SubmitWeeklyAvailability godoc
// @Summary      Replace a teacher's weekly availability
// @Description  Replaces all existing availability for a teacher with the provided weekly schedule (full replace, not merge) and deletes every booking of that teacher from the start of the current week (Sunday 00:00 Asia/Bangkok) onward, including ones earlier this week. Rejects overlapping slots within the same day. Idempotent: repeating the request deletes nothing further.
// @Tags         availability
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int                                true  "Teacher ID"
// @Param        body  body  teacher.UpdateAvailabilityRequest  true  "Weekly slots"
// @Success      200  {object}  teacher.SubmitAvailabilityResponse
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      404  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /teachers/{id}/availability [put]
func (h *AvailabilityHandler) SubmitWeeklyAvailability(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid teacher id"})
		return
	}

	var req teacher.UpdateAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deletedBookings, err := h.svc.SubmitWeeklyAvailability(c.Request.Context(), id, req.Weekly)
	if err != nil {
		respondFieldUpdateErr(c, h.logger, err, "SubmitWeeklyAvailability", "teacher", id, "failed to save availability")
		return
	}

	c.JSON(http.StatusOK, teacher.SubmitAvailabilityResponse{
		Message:         "Availability saved successfully",
		DeletedBookings: deletedBookings,
	})
}

// CreateTeacher godoc
// @Summary      Create a teacher
// @Description  Adds a new teacher with name, email, and gender
// @Tags         teachers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  teacher.CreateTeacherRequest  true  "New teacher payload"
// @Success      201  {object}  teacher.Teacher
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /teachers [post]
func (h *AvailabilityHandler) CreateTeacher(c *gin.Context) {
	var req teacher.CreateTeacherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newTeacher, err := h.svc.AddTeacher(c.Request.Context(), req.Name, req.Email, req.Gender)
	if err != nil {
		var valErr *teacher.ValidationError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			h.logger.Error("request failed",
				"request_id", requestID(c),
				"op", "CreateTeacher",
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create teacher"})
		}
		return
	}

	c.JSON(http.StatusCreated, newTeacher)
}

// UpdateTeacherStatus godoc
// @Summary      Toggle teacher active/deactivated status
// @Description  Sets a teacher's status to active or deactivated
// @Tags         teachers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int                         true  "Teacher ID"
// @Param        body  body  teacher.UpdateStatusRequest  true  "New status"
// @Success      200  {object}  scheduling.MessageResponse
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      404  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /teachers/{id}/status [patch]
func (h *AvailabilityHandler) UpdateTeacherStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid teacher id"})
		return
	}

	var req teacher.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.SetStatus(c.Request.Context(), id, req.Status); err != nil {
		respondFieldUpdateErr(c, h.logger, err, "UpdateTeacherStatus/SetStatus", "teacher", id, "failed to update teacher status")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Teacher status updated successfully"})
}

// UpdateTeacherGender godoc
// @Summary      Update a teacher's gender
// @Description  Sets a teacher's gender to male, female, or lgbtq+
// @Tags         teachers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int                         true  "Teacher ID"
// @Param        body  body  teacher.UpdateGenderRequest  true  "New gender"
// @Success      200  {object}  scheduling.MessageResponse
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      404  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /teachers/{id}/gender [patch]
func (h *AvailabilityHandler) UpdateTeacherGender(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid teacher id"})
		return
	}

	var req teacher.UpdateGenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.SetGender(c.Request.Context(), id, req.Gender); err != nil {
		respondFieldUpdateErr(c, h.logger, err, "UpdateTeacherGender/SetGender", "teacher", id, "failed to update teacher gender")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Teacher gender updated successfully"})
}

// GetTeacherSubjects godoc
// @Summary      List a teacher's subjects
// @Description  Returns the subjects a teacher is assigned to teach, ordered by name.
// @Tags         teachers
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "Teacher ID"
// @Success      200  {array}   shared.Subject
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /teachers/{id}/subjects [get]
func (h *AvailabilityHandler) GetTeacherSubjects(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid teacher id"})
		return
	}

	subjects, err := h.svc.SubjectsForTeacher(c.Request.Context(), id)
	if err != nil {
		var valErr *teacher.ValidationError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("request failed",
			"request_id", requestID(c),
			"op", "GetTeacherSubjects",
			"teacher_id", id,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve teacher subjects"})
		return
	}
	c.JSON(http.StatusOK, orEmpty(subjects))
}

// UpdateTeacherSubjects godoc
// @Summary      Replace a teacher's subjects
// @Description  Replaces the teacher's whole subject list with the one supplied (full replace, not merge). An empty list clears it.
// @Tags         teachers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int                           true  "Teacher ID"
// @Param        body  body  teacher.UpdateSubjectsRequest  true  "Subject ids"
// @Success      200  {object}  scheduling.MessageResponse
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      404  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /teachers/{id}/subjects [put]
func (h *AvailabilityHandler) UpdateTeacherSubjects(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid teacher id"})
		return
	}

	var req teacher.UpdateSubjectsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.SetTeacherSubjects(c.Request.Context(), id, req.SubjectIDs); err != nil {
		respondFieldUpdateErr(c, h.logger, err, "UpdateTeacherSubjects/SetTeacherSubjects", "teacher", id, "failed to update teacher subjects")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Teacher subjects updated successfully"})
}
