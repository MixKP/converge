package commute

import (
	"context"
	"log/slog"
	"testing"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
)

type stubStore struct {
	minutes    int
	getCalled  bool
	setCalled  bool
	setMinutes int
}

func (s *stubStore) GetCommuteMinutes(ctx context.Context) (int, error) {
	s.getCalled = true
	return s.minutes, nil
}

func (s *stubStore) SetCommuteMinutes(ctx context.Context, minutes int) error {
	s.setCalled = true
	s.setMinutes = minutes
	return nil
}

func TestCommuteService_CommuteTime(t *testing.T) {
	t.Run("different branches reads configured value", func(t *testing.T) {
		store := &stubStore{minutes: 45}
		svc := NewService(store, slog.Default())

		got, err := svc.CommuteTime(context.Background(), 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, 45, got)
		assert.True(t, store.getCalled)
	})

	t.Run("same branch returns 0 without reading store", func(t *testing.T) {
		store := &stubStore{minutes: 45}
		svc := NewService(store, slog.Default())

		got, err := svc.CommuteTime(context.Background(), 3, 3)
		assert.NoError(t, err)
		assert.Equal(t, 0, got)
		assert.False(t, store.getCalled)
	})

	t.Run("invalid branch id validates before store", func(t *testing.T) {
		for _, tc := range []struct{ source, dest int }{{0, 1}, {1, -2}} {
			store := &stubStore{minutes: 45}
			svc := NewService(store, slog.Default())
			_, err := svc.CommuteTime(context.Background(), tc.source, tc.dest)
			var valErr *shared.ValidationError
			assert.ErrorAs(t, err, &valErr)
			assert.False(t, store.getCalled)
		}
	})
}

func TestCommuteService_Minutes(t *testing.T) {
	store := &stubStore{minutes: 20}
	svc := NewService(store, slog.Default())

	got, err := svc.Minutes(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 20, got)
}

func TestCommuteService_SetCommuteTime(t *testing.T) {
	t.Run("valid value delegates to store", func(t *testing.T) {
		store := &stubStore{}
		svc := NewService(store, slog.Default())

		err := svc.SetCommuteTime(context.Background(), 45)
		assert.NoError(t, err)
		assert.True(t, store.setCalled)
		assert.Equal(t, 45, store.setMinutes)
	})

	t.Run("zero is allowed", func(t *testing.T) {
		store := &stubStore{}
		svc := NewService(store, slog.Default())

		err := svc.SetCommuteTime(context.Background(), 0)
		assert.NoError(t, err)
		assert.True(t, store.setCalled)
	})

	t.Run("negative is rejected before store", func(t *testing.T) {
		store := &stubStore{}
		svc := NewService(store, slog.Default())

		err := svc.SetCommuteTime(context.Background(), -1)
		var valErr *shared.ValidationError
		assert.ErrorAs(t, err, &valErr)
		assert.False(t, store.setCalled)
	})
}
