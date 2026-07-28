package adapter

import (
	"context"

	"github.com/RinZ5/converge/backend/internal/branch"
)

type BranchCapacityAdapter struct {
	svc *branch.Service
}

func NewBranchCapacityAdapter(svc *branch.Service) *BranchCapacityAdapter {
	if svc == nil {
		panic("adapter: NewBranchCapacityAdapter requires a non-nil branch.Service")
	}
	return &BranchCapacityAdapter{svc: svc}
}

func (a *BranchCapacityAdapter) GetCapacity(ctx context.Context, branchID int) (int, error) {
	return a.svc.GetCapacity(ctx, branchID)
}
