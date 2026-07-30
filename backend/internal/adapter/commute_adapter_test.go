package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/commute"
	"github.com/stretchr/testify/assert"
)

func TestCommuteAdapter_CommuteMinutes(t *testing.T) {
	a := NewCommuteAdapter()

	same, err := a.CommuteMinutes(context.Background(), 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, 0, same)

	diff, err := a.CommuteMinutes(context.Background(), 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, commute.DefaultCommuteMinutes, diff)
}

func TestCommuteAdapter_Estimate(t *testing.T) {
	a := NewCommuteAdapter()

	same, err := a.Estimate(context.Background(), 1, 1, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), same)

	diff, err := a.Estimate(context.Background(), 1, 2, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(commute.DefaultCommuteMinutes)*time.Minute, diff)
}
