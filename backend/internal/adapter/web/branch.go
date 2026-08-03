package web

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/RinZ5/converge/backend/internal/branch"

	"github.com/gin-gonic/gin"
)

type branchService interface {
	GetBranches(ctx context.Context) ([]branch.Branch, error)
	SetCapacity(ctx context.Context, branchID, capacity int) error
}

type BranchHandler struct {
	svc    branchService
	logger *slog.Logger
}

func NewBranchHandler(svc branchService, logger *slog.Logger) *BranchHandler {
	return &BranchHandler{svc: svc, logger: logger}
}

// GetBranches godoc
// @Summary      List branches
// @Description  Returns all branches ordered by id
// @Tags         branches
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}  branch.Branch
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /branches [get]
func (h *BranchHandler) GetBranches(c *gin.Context) {
	branches, err := h.svc.GetBranches(c.Request.Context())
	if err != nil {
		h.logger.Error("request failed",
			"request_id", requestID(c),
			"op", "GetBranches",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve branches"})
		return
	}
	c.JSON(http.StatusOK, branches)
}

// UpdateBranchCapacity godoc
// @Summary      Update a branch's capacity
// @Description  Sets how many bookings a branch can hold concurrently. 0 means unlimited/unenforced.
// @Tags         branches
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int                          true  "Branch ID"
// @Param        body  body  branch.UpdateCapacityRequest  true  "New capacity"
// @Success      200  {object}  scheduling.MessageResponse
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      404  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /branches/{id}/capacity [patch]
func (h *BranchHandler) UpdateBranchCapacity(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
		return
	}

	var req branch.UpdateCapacityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	capacity, ok := req.Capacity.Value()
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "capacity is required"})
		return
	}

	if err := h.svc.SetCapacity(c.Request.Context(), id, capacity); err != nil {
		respondFieldUpdateErr(c, h.logger, err, "UpdateBranchCapacity/SetCapacity", "branch", id, "failed to update branch capacity")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Branch capacity updated successfully"})
}
