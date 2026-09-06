package branch

import "context"

type BranchStore interface {
	GetBranches(ctx context.Context) ([]Branch, error)
	AddBranch(ctx context.Context, name string, capacity int) (*Branch, error)
	SetCapacity(ctx context.Context, branchID, capacity int) error
	SetStatus(ctx context.Context, branchID int, status string) error
}
