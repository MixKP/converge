package branch

import (
	"context"
	"log/slog"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type ValidationError = shared.ValidationError

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

func (s *Service) AddBranch(ctx context.Context, name string, capacity int) (*Branch, error) {
	if err := shared.ValidateAll(name,
		shared.NonEmpty("name", func(n string) string { return n }),
	); err != nil {
		return nil, err
	}
	// capacity 0 means unlimited/unenforced, so only negatives are rejected.
	if err := shared.ValidateAll(capacity,
		shared.NonNegativeInt("capacity", func(c int) int { return c }),
	); err != nil {
		return nil, err
	}
	return s.store.AddBranch(ctx, name, capacity)
}

func (s *Service) GetCapacity(ctx context.Context, branchID int) (int, error) {
	if err := shared.ValidateAll(branchID,
		shared.PositiveInt("branch_id", func(id int) int { return id }),
	); err != nil {
		return 0, err
	}
	b, err := s.store.GetBranchByID(ctx, branchID)
	if err != nil {
		return 0, err
	}
	return b.Capacity, nil
}

func (s *Service) SetCapacity(ctx context.Context, branchID, capacity int) error {
	if err := shared.ValidateAll(branchID,
		shared.PositiveInt("branch_id", func(id int) int { return id }),
	); err != nil {
		return err
	}
	// capacity 0 means unlimited/unenforced, so only negatives are rejected.
	if err := shared.ValidateAll(capacity,
		shared.NonNegativeInt("capacity", func(c int) int { return c }),
	); err != nil {
		return err
	}
	return s.store.SetCapacity(ctx, branchID, capacity)
}
