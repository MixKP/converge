package adapter

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/room"
	"github.com/stretchr/testify/assert"
)

func TestRoomAdapter_CheckAvailability_Success(t *testing.T) {
	svc := room.NewService(slog.Default())
	adapter := NewRoomAdapter(svc)

	ok, err := adapter.CheckAvailability(context.Background(), 1, time.Now(), time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.True(t, ok)
}
