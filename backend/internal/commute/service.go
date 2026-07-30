package commute

import (
	"context"
	"log/slog"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type Service struct {
	source CommuteSource
	logger *slog.Logger
}

func NewService(source CommuteSource, logger *slog.Logger) *Service {
	return &Service{source: source, logger: logger}
}

// CommuteTime returns the commute time in minutes between two branches,
// delegating the actual estimate to the CommuteSource port.
func (s *Service) CommuteTime(ctx context.Context, sourceBranchID, destBranchID int) (int, error) {
	if err := shared.ValidateAll(sourceBranchID, shared.PositiveInt("source_branch", func(id int) int { return id })); err != nil {
		return 0, err
	}
	if err := shared.ValidateAll(destBranchID, shared.PositiveInt("destination_branch", func(id int) int { return id })); err != nil {
		return 0, err
	}
	return s.source.CommuteMinutes(ctx, sourceBranchID, destBranchID)
}
