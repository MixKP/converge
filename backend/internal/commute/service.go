package commute

import (
	"context"
	"time"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Estimate(ctx context.Context, fromBranchID, toBranchID int, arrivalTime time.Time) (time.Duration, error) {
	return 0, nil
}
