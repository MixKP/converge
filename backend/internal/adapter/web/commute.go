package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/RinZ5/converge/backend/internal/commute"
	"github.com/RinZ5/converge/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type commuteService interface {
	CommuteTime(ctx context.Context, sourceBranchID, destBranchID int) (int, error)
}

type CommuteHandler struct {
	svc    commuteService
	logger *slog.Logger
}

func NewCommuteHandler(svc commuteService, logger *slog.Logger) *CommuteHandler {
	return &CommuteHandler{svc: svc, logger: logger}
}

// GetCommute godoc
// @Summary      Get commute time between two branches
// @Description  Returns the commute time in minutes. Without query params, returns the default commute time. With both source_branch and destination_branch, returns 0 when they are equal, otherwise the default.
// @Tags         commute
// @Produce      json
// @Param        source_branch       query  int  false  "Source branch ID"
// @Param        destination_branch  query  int  false  "Destination branch ID"
// @Success      200  {object}  commute.CommuteResponse
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /commute [get]
func (h *CommuteHandler) GetCommute(c *gin.Context) {
	sourceStr := c.Query("source_branch")
	destStr := c.Query("destination_branch")

	if sourceStr == "" && destStr == "" {
		c.JSON(http.StatusOK, commute.DefaultCommuteResponse{CommuteTime: commute.DefaultCommuteMinutes})
		return
	}

	if sourceStr == "" || destStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both source_branch and destination_branch are required"})
		return
	}

	source, err := strconv.Atoi(sourceStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source_branch"})
		return
	}

	dest, err := strconv.Atoi(destStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid destination_branch"})
		return
	}

	minutes, err := h.svc.CommuteTime(c.Request.Context(), source, dest)
	if err != nil {
		var valErr *shared.ValidationError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("request failed",
			"request_id", requestID(c),
			"op", "GetCommute",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute commute time"})
		return
	}

	c.JSON(http.StatusOK, commute.CommuteResponse{
		SourceBranch:      source,
		DestinationBranch: dest,
		CommuteTime:       minutes,
	})
}
