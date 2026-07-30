package commute

import "context"

// CommuteSource is the driven port for obtaining commute time in minutes
// between two branches. Implemented by an adapter (currently a static stub
// standing in for a real distance/maps source).
type CommuteSource interface {
	CommuteMinutes(ctx context.Context, sourceBranchID, destBranchID int) (int, error)
}
