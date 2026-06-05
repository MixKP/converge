package room

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	svc := NewService(slog.Default())
	assert.NotNil(t, svc)
}

func TestCheckAvailabilityReturnsTrue(t *testing.T) {
	svc := NewService(slog.Default())
	ok, err := svc.CheckAvailability(context.Background(), 1, time.Now(), time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.True(t, ok)
}
