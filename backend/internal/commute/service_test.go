package commute

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCommuteService_Estimate_Success(t *testing.T) {
	svc := NewService(slog.Default())
	assert.NotNil(t, svc)
	dur, err := svc.Estimate(context.Background(), 1, 2, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), dur)
}
