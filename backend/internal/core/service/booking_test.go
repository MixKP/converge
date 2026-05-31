package service

import (
	"context"
	"database/sql"
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

func (m *mockEngineRepo) FindExactMatch(ctx context.Context, subjectID, branchID int, slot models.WeeklySlot, durationMinutes int, teacherIDVal interface{}) (*ports.BookingMatch, error) {
	args := m.Called(ctx, subjectID, branchID, slot, durationMinutes, teacherIDVal)
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

func (m *mockEngineRepo) CreateBooking(ctx context.Context, req models.ConfirmBookingRequest) (*models.Booking, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Booking), args.Error(1)
}

func (m *mockEngineRepo) DeleteBooking(ctx context.Context, bookingID int) error {
	args := m.Called(ctx, bookingID)
	return args.Error(0)
}

func (m *mockEngineRepo) FindAllBookings(ctx context.Context) ([]models.Booking, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Booking), args.Error(1)
}

type mockBookingEngine struct {
	mock.Mock
}

func (m *mockBookingEngine) FindAlternativesForSlot(ctx context.Context, req models.BookingRequest, window models.WeeklySlot) ([]models.BookingAlternative, error) {
	args := m.Called(ctx, req, window)
	return args.Get(0).([]models.BookingAlternative), args.Error(1)
}

func (m *mockBookingEngine) HasResourceChecker() bool { return false }
func (m *mockBookingEngine) HasCommuteCalc() bool     { return false }

func TestBookingService_ExactMatchFound_ReturnsExactMatch(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	s := slot(0, "09:00", "10:00")
	req := bookingReq(1, 1, s, 60)

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
	repo.On("FindExactMatch", mock.Anything, 1, 1, s, 60, interface{}(nil)).Return(exactMatch, nil)

	result, err := svc.Evaluate(context.Background(), req)

	assert.NoError(t, err)
	assert.Len(t, result.Results, 1)
	assert.NotNil(t, result.Results[0].ExactMatch)
	assert.Equal(t, 100, result.Results[0].ExactMatch.Score)
	assert.Equal(t, "Alice", result.Results[0].ExactMatch.TeacherName)
	assert.Empty(t, result.Results[0].Alternatives)
	repo.AssertExpectations(t)
}

func TestBookingService_NoExactMatch_ReturnsAlternatives(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	s := slot(0, "09:00", "10:00")
	req := bookingReq(1, 1, s, 60)

	repo.On("FindExactMatch", mock.Anything, 1, 1, s, 60, interface{}(nil)).Return(nil, nil)
	engine.On("FindAlternativesForSlot", mock.Anything, req, s).Return([]models.BookingAlternative{}, nil)

	result, err := svc.Evaluate(context.Background(), req)

	assert.NoError(t, err)
	assert.Nil(t, result.Results[0].ExactMatch)
	assert.Contains(t, result.Results[0].Message, "No exact match")
}

func TestBookingService_MissingSubjectID_ValidationError(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	req := bookingReq(0, 1, slot(0, "09:00", "10:00"), 60)

	_, err := svc.Evaluate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subject_id")
}

func TestBookingService_MissingBranchID_ValidationError(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	req := bookingReq(1, 0, slot(0, "09:00", "10:00"), 60)

	_, err := svc.Evaluate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch_id")
}

func TestBookingService_EmptySlots_ValidationError(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	req := models.BookingRequest{
		SubjectID:       1,
		BranchID:        1,
		DurationMinutes: 60,
	}

	_, err := svc.Evaluate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preferred_slots")
}

func TestBookingService_InvalidDayOfWeek_ValidationError(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	req := bookingReq(1, 1, slot(7, "09:00", "10:00"), 60)

	_, err := svc.Evaluate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "day_of_week")
}

func TestBookingService_TwoSlots_OneExactMatch_OneCLP(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	monSlot := slot(0, "09:00", "10:00")
	friSlot := slot(4, "12:00", "14:00")
	req := models.BookingRequest{
		SubjectID: 1,
		BranchID:  1,
		PreferredSlots: []models.WeeklySlot{
			monSlot,
			friSlot,
		},
		DurationMinutes: 60,
	}

	exactMatch := &ports.BookingMatch{
		Booking:     models.Booking{ID: 42, TeacherID: 1, BranchID: 1, SubjectID: 1, StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)},
		TeacherName: "Alice",
	}
	repo.On("FindExactMatch", mock.Anything, 1, 1, monSlot, 60, interface{}(nil)).Return(exactMatch, nil)
	repo.On("FindExactMatch", mock.Anything, 1, 1, friSlot, 60, interface{}(nil)).Return(nil, nil)

	alts := []models.BookingAlternative{
		{TeacherID: 3, TeacherName: "Carol", Score: 70},
	}
	engine.On("FindAlternativesForSlot", mock.Anything, req, friSlot).Return(alts, nil)

	result, err := svc.Evaluate(context.Background(), req)

	assert.NoError(t, err)
	assert.Len(t, result.Results, 2)
	assert.NotNil(t, result.Results[0].ExactMatch)
	assert.Equal(t, "Alice", result.Results[0].ExactMatch.TeacherName)
	assert.Nil(t, result.Results[1].ExactMatch)
	assert.Len(t, result.Results[1].Alternatives, 1)
	assert.Equal(t, "Carol", result.Results[1].Alternatives[0].TeacherName)
}

func TestBookingService_ConfirmSuccess(t *testing.T) {
	repo := new(mockEngineRepo)
	engine := new(mockBookingEngine)
	svc := NewBookingService(repo, engine)

	req := models.ConfirmBookingRequest{
		TeacherID:  1,
		BranchID:   1,
		SubjectID:  1,
		StartTime:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		ClientName: "John Doe",
	}

	expected := &models.Booking{
		ID:         10,
		TeacherID:  1,
		BranchID:   1,
		SubjectID:  1,
		StartTime:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		ClientName: "John Doe",
	}
	repo.On("CreateBooking", mock.Anything, req).Return(expected, nil)

	result, err := svc.Confirm(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, 10, result.ID)
	assert.Equal(t, "John Doe", result.ClientName)
	repo.AssertExpectations(t)
}

func TestBookingService_ConfirmMissingFields(t *testing.T) {
	svc := NewBookingService(new(mockEngineRepo), new(mockBookingEngine))

	tests := []struct {
		name string
		req  models.ConfirmBookingRequest
		err  string
	}{
		{"teacher", models.ConfirmBookingRequest{BranchID: 1, SubjectID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour), ClientName: "X"}, "teacher_id"},
		{"branch", models.ConfirmBookingRequest{TeacherID: 1, SubjectID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour), ClientName: "X"}, "branch_id"},
		{"subject", models.ConfirmBookingRequest{TeacherID: 1, BranchID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour), ClientName: "X"}, "subject_id"},
		{"start_time", models.ConfirmBookingRequest{TeacherID: 1, BranchID: 1, SubjectID: 1, EndTime: time.Now(), ClientName: "X"}, "start_time"},
		{"end_time", models.ConfirmBookingRequest{TeacherID: 1, BranchID: 1, SubjectID: 1, StartTime: time.Now(), ClientName: "X"}, "end_time"},
		{"time_reversed", models.ConfirmBookingRequest{TeacherID: 1, BranchID: 1, SubjectID: 1, StartTime: time.Now().Add(time.Hour), EndTime: time.Now(), ClientName: "X"}, "end_time must be after start_time"},
		{"client_name", models.ConfirmBookingRequest{TeacherID: 1, BranchID: 1, SubjectID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}, "client_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Confirm(context.Background(), tt.req)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
		})
	}
}

func TestBookingService_CancelSuccess(t *testing.T) {
	repo := new(mockEngineRepo)
	svc := NewBookingService(repo, new(mockBookingEngine))

	repo.On("DeleteBooking", mock.Anything, 10).Return(nil)

	err := svc.Cancel(context.Background(), 10)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBookingService_CancelNotFound(t *testing.T) {
	repo := new(mockEngineRepo)
	svc := NewBookingService(repo, new(mockBookingEngine))

	repo.On("DeleteBooking", mock.Anything, 999).Return(sql.ErrNoRows)

	err := svc.Cancel(context.Background(), 999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBookingService_CancelInvalidID(t *testing.T) {
	svc := NewBookingService(new(mockEngineRepo), new(mockBookingEngine))

	err := svc.Cancel(context.Background(), 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "booking_id must be positive")
}
