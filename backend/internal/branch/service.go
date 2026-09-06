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
	if err := validateCapacity(capacity); err != nil {
		return nil, err
	}
	return s.store.AddBranch(ctx, name, capacity)
}

func (s *Service) SetCapacity(ctx context.Context, branchID, capacity int) error {
	if err := shared.ValidateAll(branchID,
		shared.PositiveInt("branch_id", func(id int) int { return id }),
	); err != nil {
		return err
	}
	if err := validateCapacity(capacity); err != nil {
		return err
	}
	return s.store.SetCapacity(ctx, branchID, capacity)
}

func validateCapacity(capacity int) error {
	if capacity == -1 || capacity > 0 {
		return nil
	}
	return &ValidationError{Msg: "capacity must be -1 for unlimited or a positive integer"}
}

func (s *Service) SetStatus(ctx context.Context, branchID int, status string) error {
	if err := shared.ValidateAll(branchID,
		shared.PositiveInt("branch_id", func(id int) int { return id }),
	); err != nil {
		return err
	}
	return s.store.SetStatus(ctx, branchID, status)
}
