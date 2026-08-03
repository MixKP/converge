package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/RinZ5/converge/backend/internal/scheduling"
	"github.com/RinZ5/converge/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type bookingService interface {
	Evaluate(ctx context.Context, req scheduling.BookingRequest) (*scheduling.BookingResponse, error)
	Confirm(ctx context.Context, req scheduling.ConfirmBookingRequest) (*scheduling.Booking, error)
	Cancel(ctx context.Context, bookingID int) error
	ListAll(ctx context.Context) ([]scheduling.Booking, error)
	ListForStudentIDs(ctx context.Context, studentIDs []int) ([]scheduling.Booking, error)
}

// guardianResolver returns the students a parent may view. Satisfied by user.Service.
type guardianResolver interface {
	StudentIDsForParent(ctx context.Context, parentID int) ([]int, error)
}

type BookingHandler struct {
	svc      bookingService
	guardian guardianResolver
	logger   *slog.Logger
}

func NewBookingHandler(svc bookingService, guardian guardianResolver, logger *slog.Logger) *BookingHandler {
	return &BookingHandler{svc: svc, guardian: guardian, logger: logger}
}

// CreateBooking godoc
// @Summary      Evaluate a booking request
// @Description  Evaluates each preferred_slot independently and returns per-slot results with either an exact_match or alternatives.
// @Tags         bookings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body    scheduling.BookingRequest  true  "Booking request"
// @Success      200   {object}  scheduling.BookingResponse
// @Failure      400   {object}  scheduling.ErrorResponse
// @Failure      401   {object}  scheduling.ErrorResponse
// @Failure      403   {object}  scheduling.ErrorResponse
// @Failure      500   {object}  scheduling.ErrorResponse
// @Router       /bookings [post]
func (h *BookingHandler) CreateBooking(c *gin.Context) {
	var req scheduling.BookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.Evaluate(c.Request.Context(), req)
	if err != nil {
		var valErr *scheduling.ValidationError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			h.logger.Error("request failed",
				"request_id", requestID(c),
				"op", "CreateBooking/Evaluate",
				"subject_id", req.SubjectID,
				"branch_id", req.BranchID,
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to evaluate booking"})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

// ConfirmBooking godoc
// @Summary      Confirm a booking
// @Description  Creates the actual booking in the database after the client selects a result from the evaluate endpoint.
// @Tags         bookings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  scheduling.ConfirmBookingRequest  true  "Booking confirmation"
// @Success      201   {object}  scheduling.Booking
// @Failure      400   {object}  scheduling.ErrorResponse
// @Failure      401   {object}  scheduling.ErrorResponse
// @Failure      403   {object}  scheduling.ErrorResponse
// @Failure      409   {object}  scheduling.ErrorResponse
// @Failure      500   {object}  scheduling.ErrorResponse
// @Router       /bookings/confirm [post]
func (h *BookingHandler) ConfirmBooking(c *gin.Context) {
	var req scheduling.ConfirmBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.Confirm(c.Request.Context(), req)
	if err != nil {
		var valErr *scheduling.ValidationError
		var confErr *scheduling.ConflictError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if errors.As(err, &confErr) {
			h.logger.Warn("booking conflict",
				"request_id", requestID(c),
				"op", "ConfirmBooking/Confirm",
				"teacher_id", req.TeacherID,
				"start_time", req.StartTime,
				"error", err,
			)
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			h.logger.Error("request failed",
				"request_id", requestID(c),
				"op", "ConfirmBooking/Confirm",
				"teacher_id", req.TeacherID,
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm booking"})
		}
		return
	}

	c.JSON(http.StatusCreated, result)
}

// CancelBooking godoc
// @Summary      Cancel a booking
// @Description  Deletes a confirmed booking by ID. Returns 404 if not found.
// @Tags         bookings
// @Security     BearerAuth
// @Param        id  path  int  true  "Booking ID"
// @Success      200  {object}  scheduling.MessageResponse
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      404  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /bookings/{id} [delete]
func (h *BookingHandler) CancelBooking(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	err = h.svc.Cancel(c.Request.Context(), id)
	if err != nil {
		var valErr *scheduling.ValidationError
		var notFoundErr *scheduling.NotFoundError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if errors.As(err, &notFoundErr) {
			h.logger.Warn("booking not found",
				"request_id", requestID(c),
				"op", "CancelBooking/Cancel",
				"booking_id", id,
			)
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			h.logger.Error("request failed",
				"request_id", requestID(c),
				"op", "CancelBooking/Cancel",
				"booking_id", id,
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel booking"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Booking cancelled successfully"})
}

// ListBookings godoc
// @Summary      List bookings (scoped by role)
// @Description  Admin sees all bookings; a student sees only their own; a parent sees their students' bookings.
// @Tags         bookings
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   scheduling.Booking
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /bookings [get]
func (h *BookingHandler) ListBookings(c *gin.Context) {
	ctx := c.Request.Context()
	p, ok := PrincipalFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	bookings, err := h.bookingsForPrincipal(ctx, p)
	if err != nil {
		h.logger.Error("request failed",
			"request_id", requestID(c),
			"op", "ListBookings",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list bookings"})
		return
	}
	c.JSON(http.StatusOK, orEmpty(bookings))
}

func (h *BookingHandler) bookingsForPrincipal(ctx context.Context, p shared.Principal) ([]scheduling.Booking, error) {
	switch p.Role {
	case shared.RoleAdmin:
		return h.svc.ListAll(ctx)
	case shared.RoleStudent:
		return h.svc.ListForStudentIDs(ctx, []int{p.UserID})
	case shared.RoleParent:
		ids, err := h.guardian.StudentIDsForParent(ctx, p.UserID)
		if err != nil {
			return nil, err
		}
		return h.svc.ListForStudentIDs(ctx, ids)
	default:
		return []scheduling.Booking{}, nil
	}
}
