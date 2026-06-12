package adapter

import (
	"context"
	"errors"
	"testing"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/RinZ5/converge/backend/internal/teacher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockTeacherRosterSource struct {
	mock.Mock
}

func (m *mockTeacherRosterSource) GetTeachersBySubject(ctx context.Context, subjectID int) ([]teacher.Teacher, error) {
	args := m.Called(ctx, subjectID)
	return args.Get(0).([]teacher.Teacher), args.Error(1)
}

func (m *mockTeacherRosterSource) TeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error) {
	args := m.Called(ctx, teacherID)
	return args.Get(0).([]shared.WeeklySlot), args.Error(1)
}

func TestTeacherRosterAdapter_TeachersBySubject_Success(t *testing.T) {
	src := new(mockTeacherRosterSource)
	adapter := NewTeacherRosterAdapter(src)

	src.On("GetTeachersBySubject", mock.Anything, 1).Return([]teacher.Teacher{
		{ID: 1, Name: "Alice", Email: "alice@test.com"},
		{ID: 2, Name: "Bob", Email: "bob@test.com"},
	}, nil)

	result, err := adapter.TeachersBySubject(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].ID)
	assert.Equal(t, "Alice", result[0].Name)
	assert.Equal(t, 2, result[1].ID)
	assert.Equal(t, "Bob", result[1].Name)
}

func TestTeacherRosterAdapter_TeachersBySubject_Error(t *testing.T) {
	src := new(mockTeacherRosterSource)
	adapter := NewTeacherRosterAdapter(src)

	src.On("GetTeachersBySubject", mock.Anything, 1).Return(([]teacher.Teacher)(nil), errors.New("db error"))

	_, err := adapter.TeachersBySubject(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestTeacherRosterAdapter_TeacherAvailability_Success(t *testing.T) {
	src := new(mockTeacherRosterSource)
	adapter := NewTeacherRosterAdapter(src)

	expected := []shared.WeeklySlot{{DayOfWeek: 0, Start: "09:00", End: "10:00"}}
	src.On("TeacherAvailability", mock.Anything, 42).Return(expected, nil)

	result, err := adapter.TeacherAvailability(context.Background(), 42)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "09:00", string(result[0].Start))
}
