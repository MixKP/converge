package web

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/service"

	"github.com/gin-gonic/gin"
)

type bookingService interface {
	Evaluate(ctx context.Context, req models.BookingRequest) (*models.BookingResponse, error)
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
