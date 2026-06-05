package adapter

import (
	"context"
	"time"

	"github.com/RinZ5/converge/backend/internal/commute"
)

type CommuteAdapter struct {
	svc *commute.Service
}

func NewCommuteAdapter(svc *commute.Service) *CommuteAdapter {
	return &CommuteAdapter{svc: svc}
}

func (a *CommuteAdapter) Estimate(ctx context.Context, fromBranchID, toBranchID int, arrivalTime time.Time) (time.Duration, error) {
	return a.svc.Estimate(ctx, fromBranchID, toBranchID, arrivalTime)
}
