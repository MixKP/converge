package room

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRoomService_CheckAvailability_Success(t *testing.T) {
	svc := NewService(slog.Default())
	assert.NotNil(t, svc)
	ok, err := svc.CheckAvailability(context.Background(), 1, time.Now(), time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.True(t, ok)
}
