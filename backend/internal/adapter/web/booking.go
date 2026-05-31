package web

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/service"

	"github.com/gin-gonic/gin"
)

type bookingService interface {
	Evaluate(ctx context.Context, req models.BookingRequest) (*models.BookingResponse, error)
	Confirm(ctx context.Context, req models.ConfirmBookingRequest) (*models.Booking, error)
	Cancel(ctx context.Context, bookingID int) error
}

type BookingHandler struct {
	svc bookingService
}

func NewBookingHandler(svc bookingService) *BookingHandler {
	return &BookingHandler{svc: svc}
}

// CreateBooking godoc
//
//	@Summary		Evaluate a booking request
//	@Description	Evaluates each preferred_slot independently and returns per-slot results with either an exact_match or alternatives.
//	@Description
//	@Description	**Required fields:**
//	@Description	- `subject_id` (int): The subject to book. Must be positive.
//	@Description	- `branch_id` (int): The branch location. Must be positive.
//	@Description	- `preferred_slots` (array): One or more time windows. Each slot has:
//	@Description	  - `day_of_week`: 0=Monday through 6=Sunday
//	@Description	  - `start`: Time in HH:MM format (e.g. "09:00")
//	@Description	  - `end`: Time in HH:MM format, must be after start
//	@Description
//	@Description	**Optional fields:**
//	@Description	- `duration_minutes` (int): Session length. When omitted or 0, the full window (end - start) is used as the session duration. When set, candidates are generated in 30-minute steps within the window matching this duration.
//	@Description	- `preferred_teacher_id` (int): Prioritize a specific teacher. When omitted, all teachers scored neutrally (+20). When set, matching teacher gets +40, others +0.
//	@Description
//	@Description	**Response:** `results[]` has one entry per slot, in request order. Each entry has either `exact_match` (score=100, teacher fully matches) or `alternatives` (up to 3, ranked by score 0-100).
//	@Tags			bookings
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.BookingRequest	true	"Booking request"
//	@Success		200		{object}	models.BookingResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/bookings [post]
func (h *BookingHandler) CreateBooking(c *gin.Context) {
	var req models.BookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.Evaluate(c.Request.Context(), req)
	if err != nil {
		var valErr *service.ValidationError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			log.Printf("BookingHandler.CreateBooking: Evaluate error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to evaluate booking"})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

// ConfirmBooking godoc
//
//	@Summary		Confirm a booking
//	@Description	Creates the actual booking in the database after the client selects a result from the evaluate endpoint.
//	@Description
//	@Description	**Required fields:**
//	@Description	- `teacher_id` (int): The teacher assigned to the booking.
//	@Description	- `branch_id` (int): The branch location.
//	@Description	- `subject_id` (int): The subject to book.
//	@Description	- `start_time` (RFC3339): Start time of the booking (use the value directly from the evaluate response).
//	@Description	- `end_time` (RFC3339): End time of the booking (use the value directly from the evaluate response).
//	@Description	- `client_name` (string): Name of the client making the booking.
//	@Description
//	@Description	**Constraints:** Overlapping bookings for the same teacher are rejected (409 Conflict) via EXCLUDE gist constraint.
//	@Tags			bookings
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.ConfirmBookingRequest	true	"Booking confirmation"
//	@Success		201		{object}	models.Booking
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		409		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/bookings/confirm [post]
func (h *BookingHandler) ConfirmBooking(c *gin.Context) {
	var req models.ConfirmBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.Confirm(c.Request.Context(), req)
	if err != nil {
		var valErr *service.ValidationError
		var confErr *service.ConflictError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if errors.As(err, &confErr) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			log.Printf("BookingHandler.ConfirmBooking: Confirm error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm booking"})
		}
		return
	}

	c.JSON(http.StatusCreated, result)
}

// CancelBooking godoc
//
//	@Summary		Cancel a booking
//	@Description	Deletes a confirmed booking by ID. Returns 404 if not found.
//	@Tags			bookings
//	@Param			id	path	int	true	"Booking ID"
//	@Success		200	{object}	models.MessageResponse
//	@Failure		400	{object}	models.ErrorResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/bookings/{id} [delete]
func (h *BookingHandler) CancelBooking(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	err = h.svc.Cancel(c.Request.Context(), id)
	if err != nil {
		var valErr *service.ValidationError
		var notFoundErr *service.NotFoundError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if errors.As(err, &notFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			log.Printf("BookingHandler.CancelBooking: Cancel error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel booking"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Booking cancelled successfully"})
}
