package branch

import "context"

type BranchStore interface {
	GetBranches(ctx context.Context) ([]Branch, error)
	GetBranchByID(ctx context.Context, branchID int) (*Branch, error)
}
