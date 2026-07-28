package branch

import (
	"context"
	"time"
)

type BranchStore interface {
	GetBranches(ctx context.Context) ([]Branch, error)
	GetBranchByID(ctx context.Context, branchID int) (*Branch, error)
	CountOverlappingBookings(ctx context.Context, branchID int, startTime, endTime time.Time) (int, error)
}
