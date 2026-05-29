package service

import (
	"context"
	"testing"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	teachers      []models.Teacher
	getErr        error
	replaceCalled bool
	replaceErr    error
	saveCalled    bool
	saveErr       error
}

func (m *mockRepo) GetActiveTeachers(ctx context.Context) ([]models.Teacher, error) {
	return m.teachers, m.getErr
}

func (m *mockRepo) ReplaceWeeklyAvailability(ctx context.Context, teacherID int, slots []models.WeeklySlot) error {
	m.replaceCalled = true
	return m.replaceErr
}

func (m *mockRepo) SaveRawSubmission(ctx context.Context, teacherID int, rawPayload []byte) error {
	m.saveCalled = true
	return m.saveErr
}

func TestSubmitAvailability_Success(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 42,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
			{DayOfWeek: 0, Start: models.TimeHHMM("11:00"), End: models.TimeHHMM("12:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)
	assert.NoError(t, err)
	assert.True(t, mock.replaceCalled)
	assert.True(t, mock.saveCalled)
}

func TestSubmitAvailability_Overlap(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("11:00")},
			{DayOfWeek: 0, Start: models.TimeHHMM("10:30"), End: models.TimeHHMM("12:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.Contains(t, err.Error(), "overlapping")
	assert.False(t, mock.replaceCalled)
}

func TestSubmitAvailability_EmptySlots(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly:    []models.WeeklySlot{},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.False(t, mock.replaceCalled)
}

func TestSubmitAvailability_ZeroTeacherID(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 0,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.False(t, mock.replaceCalled)
}

func TestSubmitAvailability_NegativeTeacherID(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: -1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.False(t, mock.replaceCalled)
}

func TestSubmitAvailability_DayOfWeekOutOfRange(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 7, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.False(t, mock.replaceCalled)
}

func TestSubmitAvailability_EmptyStart(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM(""), End: models.TimeHHMM("10:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.False(t, mock.replaceCalled)
}

func TestSubmitAvailability_EmptyEnd(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.False(t, mock.replaceCalled)
}

func TestSubmitAvailability_StartAfterEnd(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("10:00"), End: models.TimeHHMM("09:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.False(t, mock.replaceCalled)
}

func TestSubmitAvailability_TouchingSlots(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
			{DayOfWeek: 0, Start: models.TimeHHMM("10:00"), End: models.TimeHHMM("11:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)
	assert.NoError(t, err)
}

func TestSubmitAvailability_SingleSlot(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)
	assert.NoError(t, err)
}

func TestSubmitAvailability_UnsortedSlots(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("11:00"), End: models.TimeHHMM("12:00")},
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)
	assert.NoError(t, err)
}

func TestSubmitAvailability_ReplaceError(t *testing.T) {
	mock := &mockRepo{replaceErr: assert.AnError}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestSubmitAvailability_SaveError(t *testing.T) {
	mock := &mockRepo{saveErr: assert.AnError}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)
	assert.NoError(t, err)
	assert.True(t, mock.saveCalled)
}

func TestGetActiveTeachers_Success(t *testing.T) {
	mock := &mockRepo{
		teachers: []models.Teacher{
			{ID: 1, Name: "Alice", Email: "alice@test.com"},
			{ID: 2, Name: "Bob", Email: "bob@test.com"},
		},
	}
	svc := NewAvailabilityService(mock)

	teachers, err := svc.GetActiveTeachers(context.Background())
	assert.NoError(t, err)
	assert.Len(t, teachers, 2)
	assert.Equal(t, "alice@test.com", teachers[0].Email)
	assert.Equal(t, "bob@test.com", teachers[1].Email)
}

func TestGetActiveTeachers_Error(t *testing.T) {
	mock := &mockRepo{getErr: assert.AnError}
	svc := NewAvailabilityService(mock)

	teachers, err := svc.GetActiveTeachers(context.Background())
	assert.ErrorIs(t, err, assert.AnError)
	assert.Nil(t, teachers)
}

func TestSubmitAvailabilityCrossDaySlots(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("11:00")},
			{DayOfWeek: 1, Start: models.TimeHHMM("10:00"), End: models.TimeHHMM("12:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)
	assert.NoError(t, err)
}

func TestSubmitAvailabilityMultipleDaysValid(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
			{DayOfWeek: 0, Start: models.TimeHHMM("11:00"), End: models.TimeHHMM("12:00")},
			{DayOfWeek: 1, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("12:00")},
			{DayOfWeek: 4, Start: models.TimeHHMM("14:00"), End: models.TimeHHMM("16:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)
	assert.NoError(t, err)
	assert.True(t, mock.replaceCalled)
	assert.True(t, mock.saveCalled)
}

func TestSubmitAvailabilityOverlapMessageFormat(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAvailabilityService(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("11:00")},
			{DayOfWeek: 0, Start: models.TimeHHMM("10:30"), End: models.TimeHHMM("12:00")},
		},
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), payload)

	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "overlapping slots on day 0: 09:00-11:00 and 10:30-12:00", valErr.Error())
	assert.False(t, mock.replaceCalled)
}
