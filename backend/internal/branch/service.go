package branch

import (
	"context"
	"log/slog"
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

func (s *Service) GetCapacity(ctx context.Context, branchID int) (int, error) {
	b, err := s.store.GetBranchByID(ctx, branchID)
	if err != nil {
		return 0, err
	}
	return b.Capacity, nil
}

func (s *Service) SetCapacity(ctx context.Context, branchID, capacity int) error {
	return s.store.SetCapacity(ctx, branchID, capacity)
}
