package branch

import "context"

type BranchStore interface {
	CreateBranch(ctx context.Context, name string, capacity int) (*Branch, error)
	GetBranches(ctx context.Context) ([]Branch, error)
	GetBranchByID(ctx context.Context, branchID int) (*Branch, error)
	SetCapacity(ctx context.Context, branchID, capacity int) error
}
