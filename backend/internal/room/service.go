package room

import (
	"context"
	"time"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) CheckAvailability(ctx context.Context, branchID int, startTime, endTime time.Time) (bool, error) {
	return true, nil
}
