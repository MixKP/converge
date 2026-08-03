package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/RinZ5/converge/backend/internal/user"

	"github.com/gin-gonic/gin"
)

type userService interface {
	Register(ctx context.Context, name, password, role string, studentIDs []int) (*user.User, error)
	Login(ctx context.Context, name, password string) (*user.User, string, error)
	ListStudents(ctx context.Context) ([]user.User, error)
	ListParents(ctx context.Context) ([]user.ParentWithStudents, error)
	StudentsForParent(ctx context.Context, parentID int) ([]user.User, error)
	AddStudentToParent(ctx context.Context, parentID, studentID int) error
	RemoveStudentFromParent(ctx context.Context, parentID, studentID int) error
}

type UserHandler struct {
	svc    userService
	logger *slog.Logger
}

func NewUserHandler(svc userService, logger *slog.Logger) *UserHandler {
	return &UserHandler{svc: svc, logger: logger}
}

// Register godoc
// @Summary      Register a new user (admin only)
// @Description  Admin-only. Creates a user with a unique name, a bcrypt-hashed password, and the given role.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  user.RegisterRequest  true  "New user credentials and role"
// @Success      201  {object}  user.User
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      409  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req user.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.svc.Register(c.Request.Context(), req.Name, req.Password, req.Role, req.StudentIDs)
	if err != nil {
		var confErr *user.ConflictError
		var valErr *user.ValidationError
		switch {
		case errors.As(err, &confErr):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.As(err, &valErr):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			h.logger.Error("request failed",
				"request_id", requestID(c),
				"op", "Register",
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register user"})
		}
		return
	}

	c.JSON(http.StatusCreated, u)
}

// Login godoc
// @Summary      Log in
// @Description  Verifies a user's name and password and returns a JWT access token plus the user.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body  user.LoginRequest  true  "User credentials"
// @Success      200  {object}  user.AuthResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req user.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, tok, err := h.svc.Login(c.Request.Context(), req.Name, req.Password)
	if err != nil {
		// Bad input and bad credentials are both ValidationError, and both map
		// to 401 so login never distinguishes the two.
		var valErr *user.ValidationError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("request failed",
			"request_id", requestID(c),
			"op", "Login",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to log in"})
		return
	}

	c.JSON(http.StatusOK, user.AuthResponse{Token: tok, User: *u})
}

// ListStudents godoc
// @Summary      List all students (admin only)
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   user.User
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /students [get]
func (h *UserHandler) ListStudents(c *gin.Context) {
	students, err := h.svc.ListStudents(c.Request.Context())
	if err != nil {
		h.respondUserErr(c, err, "ListStudents")
		return
	}
	c.JSON(http.StatusOK, students)
}

// ListParents godoc
// @Summary      List all parents with their students (admin only)
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   user.ParentWithStudents
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /parents [get]
func (h *UserHandler) ListParents(c *gin.Context) {
	parents, err := h.svc.ListParents(c.Request.Context())
	if err != nil {
		h.respondUserErr(c, err, "ListParents")
		return
	}
	c.JSON(http.StatusOK, parents)
}

// GetParentStudents godoc
// @Summary      List a parent's linked students (admin only)
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "Parent user ID"
// @Success      200  {array}   user.User
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /parents/{id}/students [get]
func (h *UserHandler) GetParentStudents(c *gin.Context) {
	parentID, ok := pathID(c, "id")
	if !ok {
		return
	}
	students, err := h.svc.StudentsForParent(c.Request.Context(), parentID)
	if err != nil {
		h.respondUserErr(c, err, "GetParentStudents")
		return
	}
	c.JSON(http.StatusOK, students)
}

// AddParentStudent godoc
// @Summary      Link a student to a parent (admin only)
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int                       true  "Parent user ID"
// @Param        body  body  user.LinkStudentRequest   true  "Student to link"
// @Success      200  {object}  scheduling.MessageResponse
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      404  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /parents/{id}/students [post]
func (h *UserHandler) AddParentStudent(c *gin.Context) {
	parentID, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req user.LinkStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.AddStudentToParent(c.Request.Context(), parentID, req.StudentID); err != nil {
		h.respondUserErr(c, err, "AddParentStudent")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "student linked"})
}

// RemoveParentStudent godoc
// @Summary      Unlink a student from a parent (admin only)
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  int  true  "Parent user ID"
// @Param        studentId  path  int  true  "Student user ID"
// @Success      200  {object}  scheduling.MessageResponse
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      404  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /parents/{id}/students/{studentId} [delete]
func (h *UserHandler) RemoveParentStudent(c *gin.Context) {
	parentID, ok := pathID(c, "id")
	if !ok {
		return
	}
	studentID, ok := pathID(c, "studentId")
	if !ok {
		return
	}
	if err := h.svc.RemoveStudentFromParent(c.Request.Context(), parentID, studentID); err != nil {
		h.respondUserErr(c, err, "RemoveParentStudent")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "student unlinked"})
}

func (h *UserHandler) respondUserErr(c *gin.Context, err error, op string) {
	var notFound *shared.NotFoundError
	var valErr *user.ValidationError
	var confErr *user.ConflictError
	switch {
	case errors.As(err, &notFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.As(err, &valErr):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.As(err, &confErr):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		h.logger.Error("request failed", "request_id", requestID(c), "op", op, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request failed"})
	}
}

func pathID(c *gin.Context, name string) (int, bool) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return 0, false
	}
	return id, true
}
