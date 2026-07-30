package adapter

import (
	"context"
	"time"

	"github.com/RinZ5/converge/backend/internal/commute"
)

// CommuteAdapter is the leaf source of commute time: 0 minutes when the
// branches are the same, otherwise commute.DefaultCommuteMinutes. It stands in
// for a real distance/maps source and satisfies both the commute.CommuteSource
// port (used by commute.Service) and scheduling.CommuteEstimate (used by the
// CLP engine).
type CommuteAdapter struct{}

func NewCommuteAdapter() *CommuteAdapter {
	return &CommuteAdapter{}
}

func (a *CommuteAdapter) CommuteMinutes(ctx context.Context, sourceBranchID, destBranchID int) (int, error) {
	// tradeoff: static 0/30 heuristic -> real branch-distance lookup
	if sourceBranchID == destBranchID {
		return 0, nil
	}
	return commute.DefaultCommuteMinutes, nil
}

func (a *CommuteAdapter) Estimate(ctx context.Context, fromBranchID, toBranchID int, arrivalTime time.Time) (time.Duration, error) {
	mins, err := a.CommuteMinutes(ctx, fromBranchID, toBranchID)
	if err != nil {
		return 0, err
	}
	return time.Duration(mins) * time.Minute, nil
}
