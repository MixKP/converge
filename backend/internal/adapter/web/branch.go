package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/RinZ5/converge/backend/internal/branch"
	"github.com/RinZ5/converge/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type branchService interface {
	GetBranches(ctx context.Context) ([]branch.Branch, error)
	AddBranch(ctx context.Context, name string, capacity int) (*branch.Branch, error)
	SetCapacity(ctx context.Context, branchID, capacity int) error
	SetStatus(ctx context.Context, branchID int, status string) error
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

// CreateBranch godoc
// @Summary      Create a branch
// @Description  Adds a new branch with a name and optional capacity (0 means unlimited/unenforced)
// @Tags         branches
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  branch.CreateBranchRequest  true  "New branch payload"
// @Success      201  {object}  branch.Branch
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      409  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /branches [post]
func (h *BranchHandler) CreateBranch(c *gin.Context) {
	var req branch.CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newBranch, err := h.svc.AddBranch(c.Request.Context(), req.Name, req.Capacity)
	if err != nil {
		var confErr *shared.ConflictError
		var valErr *shared.ValidationError
		switch {
		case errors.As(err, &confErr):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.As(err, &valErr):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			h.logger.Error("request failed",
				"request_id", requestID(c),
				"op", "CreateBranch",
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create branch"})
		}
		return
	}

	c.JSON(http.StatusCreated, newBranch)
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

// UpdateBranchStatus godoc
// @Summary      Toggle branch active/deactivated status
// @Description  Sets a branch's status to active or deactivated. Deactivated branches stay listed for history but cannot take new bookings.
// @Tags         branches
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int                         true  "Branch ID"
// @Param        body  body  branch.UpdateStatusRequest  true  "New status"
// @Success      200  {object}  scheduling.MessageResponse
// @Failure      400  {object}  scheduling.ErrorResponse
// @Failure      401  {object}  scheduling.ErrorResponse
// @Failure      403  {object}  scheduling.ErrorResponse
// @Failure      404  {object}  scheduling.ErrorResponse
// @Failure      500  {object}  scheduling.ErrorResponse
// @Router       /branches/{id}/status [patch]
func (h *BranchHandler) UpdateBranchStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
		return
	}

	var req branch.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.SetStatus(c.Request.Context(), id, req.Status); err != nil {
		respondFieldUpdateErr(c, h.logger, err, "UpdateBranchStatus/SetStatus", "branch", id, "failed to update branch status")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Branch status updated successfully"})
}
