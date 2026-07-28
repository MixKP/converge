package adapter

import (
	"context"
	"time"

	"github.com/RinZ5/converge/backend/internal/branch"
)

type BranchCapacityAdapter struct {
	svc *branch.Service
}

func NewBranchCapacityAdapter(svc *branch.Service) *BranchCapacityAdapter {
	return &BranchCapacityAdapter{svc: svc}
}

func (a *BranchCapacityAdapter) CheckCapacity(ctx context.Context, branchID int, startTime, endTime time.Time) (bool, error) {
	return a.svc.CheckCapacity(ctx, branchID, startTime, endTime)
}
