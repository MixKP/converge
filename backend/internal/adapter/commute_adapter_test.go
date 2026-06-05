package adapter

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/commute"
	"github.com/stretchr/testify/assert"
)

func TestCommuteAdapter(t *testing.T) {
	svc := commute.NewService(slog.Default())
	adapter := NewCommuteAdapter(svc)

	dur, err := adapter.Estimate(context.Background(), 1, 2, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), dur)
}
