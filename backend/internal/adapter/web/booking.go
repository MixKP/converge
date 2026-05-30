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
//	@Description	Tries to find an exact match for the booking request via the CLP engine. If none is found, returns up to 3 best alternatives.
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
