package commute

import (
	"context"
	"log/slog"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type Service struct {
	store  ConfigStore
	logger *slog.Logger
}

func NewService(store ConfigStore, logger *slog.Logger) *Service {
	return &Service{store: store, logger: logger}
}

// Minutes returns the globally-configured commute time in minutes.
func (s *Service) Minutes(ctx context.Context) (int, error) {
	return s.store.GetCommuteMinutes(ctx)
}

// CommuteTime returns the commute time in minutes between two branches: 0 when
// they are the same, otherwise the configured global value.
func (s *Service) CommuteTime(ctx context.Context, sourceBranchID, destBranchID int) (int, error) {
	if err := shared.ValidateAll(sourceBranchID, shared.PositiveInt("source_branch", func(id int) int { return id })); err != nil {
		return 0, err
	}
	if err := shared.ValidateAll(destBranchID, shared.PositiveInt("destination_branch", func(id int) int { return id })); err != nil {
		return 0, err
	}
	if sourceBranchID == destBranchID {
		return 0, nil
	}
	return s.store.GetCommuteMinutes(ctx)
}

// SetCommuteTime updates the globally-configured commute time in minutes.
func (s *Service) SetCommuteTime(ctx context.Context, minutes int) error {
	if err := shared.ValidateAll(minutes, shared.NonNegativeInt("commute_time", func(m int) int { return m })); err != nil {
		return err
	}
	return s.store.SetCommuteMinutes(ctx, minutes)
}
