package adapter

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/branch"
	"github.com/stretchr/testify/assert"
)

type stubBranchStore struct {
	branches         []branch.Branch
	branch           *branch.Branch
	overlappingCount int
}

func (s *stubBranchStore) GetBranches(ctx context.Context) ([]branch.Branch, error) {
	return s.branches, nil
}

func (s *stubBranchStore) GetBranchByID(ctx context.Context, branchID int) (*branch.Branch, error) {
	return s.branch, nil
}

func (s *stubBranchStore) CountOverlappingBookings(ctx context.Context, branchID int, startTime, endTime time.Time) (int, error) {
	return s.overlappingCount, nil
}

func TestBranchCapacityAdapter_CheckCapacity_Success(t *testing.T) {
	store := &stubBranchStore{branch: &branch.Branch{ID: 1, Name: "Siam", Capacity: 3}, overlappingCount: 1}
	svc := branch.NewService(store, slog.Default())
	adapter := NewBranchCapacityAdapter(svc)

	ok, err := adapter.CheckCapacity(context.Background(), 1, time.Now(), time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.True(t, ok)
}
