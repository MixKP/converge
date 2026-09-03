package adapter

import (
	"context"
	"log/slog"
	"testing"

	"github.com/RinZ5/converge/backend/internal/branch"
	"github.com/stretchr/testify/assert"
)

type stubBranchStore struct {
	branches []branch.Branch
	branch   *branch.Branch
}

func (s *stubBranchStore) GetBranches(ctx context.Context) ([]branch.Branch, error) {
	return s.branches, nil
}

func (s *stubBranchStore) GetBranchByID(ctx context.Context, branchID int) (*branch.Branch, error) {
	return s.branch, nil
}

func (s *stubBranchStore) AddBranch(ctx context.Context, name string, capacity int) (*branch.Branch, error) {
	return s.branch, nil
}

func (s *stubBranchStore) SetCapacity(ctx context.Context, branchID, capacity int) error {
	return nil
}

func TestBranchCapacityAdapter_GetCapacity_Success(t *testing.T) {
	store := &stubBranchStore{branch: &branch.Branch{ID: 1, Name: "Siam", Capacity: 3}}
	svc := branch.NewService(store, slog.Default())
	adapter := NewBranchCapacityAdapter(svc)

	capacity, err := adapter.GetCapacity(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, 3, capacity)
}

// TestNewBranchCapacityAdapter_NilService_Panics closes the typed-nil gap a
// prior review flagged: wrapping a nil *branch.Service in a
// BranchCapacityAdapter produces a non-nil BranchCapacityCheck interface (a
// "typed nil"), which CLPEngine's own `branchCapacity == nil` guard would not
// catch. The fix is at the source — refuse to construct the adapter at all
// around a nil service, so a typed-nil interface can never reach
// NewCLPEngine in the first place.
func TestNewBranchCapacityAdapter_NilService_Panics(t *testing.T) {
	var nilSvc *branch.Service
	assert.Panics(t, func() {
		NewBranchCapacityAdapter(nilSvc)
	})
}
