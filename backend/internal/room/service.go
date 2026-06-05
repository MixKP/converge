package room

import (
	"context"
	"log/slog"
	"time"
)

type Service struct {
	logger *slog.Logger
}

func NewService(logger *slog.Logger) *Service {
	return &Service{logger: logger}
}

func (s *Service) CheckAvailability(ctx context.Context, branchID int, startTime, endTime time.Time) (bool, error) {
	return true, nil
}
