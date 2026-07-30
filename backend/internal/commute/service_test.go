package commute

import (
	"context"
	"log/slog"
	"testing"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
)

type stubSource struct {
	minutes int
	called  bool
}

func (s *stubSource) CommuteMinutes(ctx context.Context, sourceBranchID, destBranchID int) (int, error) {
	s.called = true
	return s.minutes, nil
}

func TestCommuteService_CommuteTime_DelegatesToSource(t *testing.T) {
	src := &stubSource{minutes: 30}
	svc := NewService(src, slog.Default())

	got, err := svc.CommuteTime(context.Background(), 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, 30, got)
	assert.True(t, src.called)
}

func TestCommuteService_CommuteTime_ValidatesBeforeSource(t *testing.T) {
	tests := []struct {
		name   string
		source int
		dest   int
	}{
		{name: "zero source", source: 0, dest: 1},
		{name: "negative destination", source: 1, dest: -2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &stubSource{minutes: 30}
			svc := NewService(src, slog.Default())

			_, err := svc.CommuteTime(context.Background(), tt.source, tt.dest)
			var valErr *shared.ValidationError
			assert.ErrorAs(t, err, &valErr)
			assert.False(t, src.called)
		})
	}
}
