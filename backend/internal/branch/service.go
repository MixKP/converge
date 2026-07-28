package branch

import (
	"context"
	"log/slog"
	"time"
)

type Service struct {
	store  BranchStore
	logger *slog.Logger
}

func NewService(store BranchStore, logger *slog.Logger) *Service {
	return &Service{store: store, logger: logger}
}

func (s *Service) GetBranches(ctx context.Context) ([]Branch, error) {
	return s.store.GetBranches(ctx)
}

func (s *Service) CheckCapacity(ctx context.Context, branchID int, startTime, endTime time.Time) (bool, error) {
	b, err := s.store.GetBranchByID(ctx, branchID)
	if err != nil {
		return false, err
	}
	count, err := s.store.CountOverlappingBookings(ctx, branchID, startTime, endTime)
	if err != nil {
		return false, err
	}
	return count < b.Capacity, nil
}
