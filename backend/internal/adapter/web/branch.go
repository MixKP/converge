package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/RinZ5/converge/backend/internal/branch"

	"github.com/gin-gonic/gin"
)

type branchService interface {
	GetBranches(ctx context.Context) ([]branch.Branch, error)
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
// @Success      200  {array}  branch.Branch
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
