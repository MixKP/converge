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

func TestBranchCapacityAdapter_GetCapacity_Success(t *testing.T) {
	store := &stubBranchStore{branch: &branch.Branch{ID: 1, Name: "Siam", Capacity: 3}}
	svc := branch.NewService(store, slog.Default())
	adapter := NewBranchCapacityAdapter(svc)

	capacity, err := adapter.GetCapacity(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, 3, capacity)
}
