package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type stubCommuteSource struct{ minutes int }

func (s stubCommuteSource) Minutes(ctx context.Context) (int, error) { return s.minutes, nil }

func TestCommuteAdapter_DefaultCommute(t *testing.T) {
	a := NewCommuteAdapter(stubCommuteSource{minutes: 45})

	d, err := a.DefaultCommute(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 45*time.Minute, d)
}

func TestNewCommuteAdapter_NilSource_Panics(t *testing.T) {
	const msg = "adapter: NewCommuteAdapter requires a non-nil CommuteConfigSource"

	t.Run("nil source", func(t *testing.T) {
		var nilSrc CommuteConfigSource
		assert.PanicsWithValue(t, msg, func() { NewCommuteAdapter(nilSrc) })
	})

	t.Run("typed nil source", func(t *testing.T) {
		var typedNil *typedNilSource
		assert.PanicsWithValue(t, msg, func() { NewCommuteAdapter(typedNil) })
	})
}

type typedNilSource struct{}

func (s *typedNilSource) Minutes(ctx context.Context) (int, error) { return 0, nil }
