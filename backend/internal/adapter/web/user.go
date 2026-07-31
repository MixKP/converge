package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/RinZ5/converge/backend/internal/user"

	"github.com/gin-gonic/gin"
)

type userService interface {
	Register(ctx context.Context, name, password string) (*user.User, error)
	Login(ctx context.Context, name, password string) (*user.User, error)
}

type UserHandler struct {
	svc    userService
	logger *slog.Logger
}

func NewUserHandler(svc userService, logger *slog.Logger) *UserHandler {
	return &UserHandler{svc: svc, logger: logger}
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a user with a unique name and a bcrypt-hashed password
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body  user.RegisterRequest  true  "New user credentials"
// @Success      201  {object}  user.User
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      409  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req user.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.svc.Register(c.Request.Context(), req.Name, req.Password)
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
// @Description  Verifies a user's name and password. Returns the user on success. No token is issued yet.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body  user.LoginRequest  true  "User credentials"
// @Success      200  {object}  user.User
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req user.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.svc.Login(c.Request.Context(), req.Name, req.Password)
	if err != nil {
		// Bad input and bad credentials are both ValidationError; while auth is
		// stubbed, both map to 401 so login never distinguishes the two.
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

	c.JSON(http.StatusOK, u)
}
