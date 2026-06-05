package adapter

import (
	"context"
	"time"

	"github.com/RinZ5/converge/backend/internal/room"
)

type RoomAdapter struct {
	svc *room.Service
}

func NewRoomAdapter(svc *room.Service) *RoomAdapter {
	return &RoomAdapter{svc: svc}
}

func (a *RoomAdapter) CheckAvailability(ctx context.Context, branchID int, startTime, endTime time.Time) (bool, error) {
	return a.svc.CheckAvailability(ctx, branchID, startTime, endTime)
}
