package service

import (
	"context"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/ports"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockEngineRepo struct {
	mock.Mock
}

func (m *mockEngineRepo) FindExactMatch(ctx context.Context, req models.BookingRequest) (*ports.BookingMatch, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.BookingMatch), args.Error(1)
}

func (m *mockEngineRepo) FindTeachersBySubject(ctx context.Context, subjectID int) ([]models.Teacher, error) {
	args := m.Called(ctx, subjectID)
	return args.Get(0).([]models.Teacher), args.Error(1)
}

func (m *mockEngineRepo) FindConflictingBookings(ctx context.Context, teacherID int, startTime, endTime time.Time) ([]models.Booking, error) {
	args := m.Called(ctx, teacherID, startTime, endTime)
	return args.Get(0).([]models.Booking), args.Error(1)
}

func (m *mockEngineRepo) FindTeacherAvailability(ctx context.Context, teacherID int) ([]models.WeeklySlot, error) {
	args := m.Called(ctx, teacherID)
	return args.Get(0).([]models.WeeklySlot), args.Error(1)
}

type mockBookingEngine struct {
	mock.Mock
}

func (m *mockBookingEngine) FindAlternatives(ctx context.Context, req models.BookingRequest) ([]models.BookingAlternative, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]models.BookingAlternative), args.Error(1)
}

func (m *mockBookingEngine) HasResourceChecker() bool { return false }
func (m *mockBookingEngine) HasCommuteCalc() bool     { return false }

func TestBookingService_ExactMatchFound_ReturnsExactMatch(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), 60)

	exactMatch := &ports.BookingMatch{
		Booking: models.Booking{
			ID:        42,
			TeacherID: 1,
			BranchID:  1,
			SubjectID: 1,
			StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		},
		TeacherName: "Alice",
	}
	repo.On("FindExactMatch", mock.Anything, req).Return(exactMatch, nil)

	result, err := svc.Evaluate(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result.ExactMatch)
	assert.Equal(t, 100, result.ExactMatch.Score)
	assert.Equal(t, "Alice", result.ExactMatch.TeacherName)
	assert.Empty(t, result.Alternatives)
	repo.AssertExpectations(t)
}

func TestBookingService_NoExactMatch_ReturnsAlternatives(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), 60)

	repo.On("FindExactMatch", mock.Anything, req).Return(nil, nil)
	engine.On("FindAlternatives", mock.Anything, req).Return([]models.BookingAlternative{}, nil)

	result, err := svc.Evaluate(context.Background(), req)

	assert.NoError(t, err)
	assert.Nil(t, result.ExactMatch)
	assert.Contains(t, result.Message, "No exact match")
}

func TestBookingService_MissingSubjectID_ValidationError(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	req := bookingReq(0, 1, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), 60)

	_, err := svc.Evaluate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subject_id")
}

func TestBookingService_MissingBranchID_ValidationError(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	req := bookingReq(1, 0, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), 60)

	_, err := svc.Evaluate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch_id")
}

func TestBookingService_InvalidDuration_ValidationError(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), 0)

	_, err := svc.Evaluate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duration")
}
