package scheduling

import (
	"context"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockStore struct {
	mock.Mock
}

func (m *mockStore) FindExactMatch(ctx context.Context, subjectID, branchID int, slot shared.WeeklySlot, durationMinutes int, teacherID shared.Option[int]) (*BookingMatch, error) {
	args := m.Called(ctx, subjectID, branchID, slot, durationMinutes, teacherID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BookingMatch), args.Error(1)
}

func (m *mockStore) FindConflictingBookings(ctx context.Context, teacherID int, startTime, endTime time.Time) ([]Booking, error) {
	args := m.Called(ctx, teacherID, startTime, endTime)
	return args.Get(0).([]Booking), args.Error(1)
}

func (m *mockStore) CreateBooking(ctx context.Context, req ConfirmBookingRequest) (*Booking, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Booking), args.Error(1)
}

func (m *mockStore) DeleteBooking(ctx context.Context, bookingID int) error {
	args := m.Called(ctx, bookingID)
	return args.Error(0)
}

func (m *mockStore) FindAllBookings(ctx context.Context) ([]Booking, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Booking), args.Error(1)
}

func (m *mockStore) GetBranches(ctx context.Context) ([]Branch, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Branch), args.Error(1)
}

func (m *mockStore) GetSubjects(ctx context.Context) ([]Subject, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Subject), args.Error(1)
}

type mockEngine struct {
	mock.Mock
}

func (m *mockEngine) FindAlternativesForSlot(ctx context.Context, req BookingRequest, window shared.WeeklySlot) ([]BookingAlternative, error) {
	args := m.Called(ctx, req, window)
	return args.Get(0).([]BookingAlternative), args.Error(1)
}

func slot(day int, start, end string) shared.WeeklySlot {
	return shared.WeeklySlot{DayOfWeek: day, Start: shared.TimeHHMM(start), End: shared.TimeHHMM(end)}
}

func bookingReq(subjectID, branchID int, s shared.WeeklySlot, dur int) BookingRequest {
	return BookingRequest{
		SubjectID:       subjectID,
		BranchID:        branchID,
		PreferredSlots:  []shared.WeeklySlot{s},
		DurationMinutes: dur,
		RequiredGender:  "female",
	}
}

func TestSchedulingService_Evaluate_ExactMatch(t *testing.T) {
	store := new(mockStore)
	engine := new(mockEngine)
	svc := NewSchedulingService(store, store, engine)

	s := slot(0, "09:00", "10:00")
	req := bookingReq(1, 1, s, 60)

	exactMatch := &BookingMatch{
		Booking: Booking{
			ID:        42,
			TeacherID: 1,
			BranchID:  1,
			SubjectID: 1,
			StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		},
		TeacherName: "Alice",
	}
	store.On("FindExactMatch", mock.Anything, 1, 1, s, 60, shared.None[int]()).Return(exactMatch, nil)

	result, err := svc.Evaluate(context.Background(), req)

	assert.NoError(t, err)
	assert.Len(t, result.Results, 1)
	assert.NotNil(t, result.Results[0].ExactMatch)
	assert.Equal(t, 100, result.Results[0].ExactMatch.Score)
	assert.Equal(t, "Alice", result.Results[0].ExactMatch.TeacherName)
	assert.Empty(t, result.Results[0].Alternatives)
	store.AssertExpectations(t)
}

func TestSchedulingService_Evaluate_Alternatives(t *testing.T) {
	store := new(mockStore)
	engine := new(mockEngine)
	svc := NewSchedulingService(store, store, engine)

	s := slot(0, "09:00", "10:00")
	req := bookingReq(1, 1, s, 60)

	store.On("FindExactMatch", mock.Anything, 1, 1, s, 60, shared.None[int]()).Return(nil, nil)
	engine.On("FindAlternativesForSlot", mock.Anything, req, s).Return([]BookingAlternative{}, nil)

	result, err := svc.Evaluate(context.Background(), req)

	assert.NoError(t, err)
	assert.Nil(t, result.Results[0].ExactMatch)
	assert.Contains(t, result.Results[0].Message, "No exact match")
}

func TestSchedulingService_Evaluate_MissingSubject(t *testing.T) {
	store := new(mockStore)
	engine := new(mockEngine)
	svc := NewSchedulingService(store, store, engine)

	req := bookingReq(0, 1, slot(0, "09:00", "10:00"), 60)

	_, err := svc.Evaluate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subject_id")
}

func TestSchedulingService_Evaluate_MissingBranch(t *testing.T) {
	store := new(mockStore)
	engine := new(mockEngine)
	svc := NewSchedulingService(store, store, engine)

	req := bookingReq(1, 0, slot(0, "09:00", "10:00"), 60)

	_, err := svc.Evaluate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch_id")
}

func TestSchedulingService_Evaluate_EmptySlots(t *testing.T) {
	store := new(mockStore)
	engine := new(mockEngine)
	svc := NewSchedulingService(store, store, engine)

	req := BookingRequest{
		SubjectID:       1,
		BranchID:        1,
		DurationMinutes: 60,
		RequiredGender:  "female",
	}

	_, err := svc.Evaluate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preferred_slots")
}

func TestSchedulingService_Evaluate_InvalidDay(t *testing.T) {
	store := new(mockStore)
	engine := new(mockEngine)
	svc := NewSchedulingService(store, store, engine)

	req := bookingReq(1, 1, slot(7, "09:00", "10:00"), 60)

	_, err := svc.Evaluate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "day_of_week")
}

func TestSchedulingService_Evaluate_TwoSlots(t *testing.T) {
	store := new(mockStore)
	engine := new(mockEngine)
	svc := NewSchedulingService(store, store, engine)

	monSlot := slot(0, "09:00", "10:00")
	friSlot := slot(4, "12:00", "14:00")
	req := BookingRequest{
		SubjectID: 1,
		BranchID:  1,
		PreferredSlots: []shared.WeeklySlot{
			monSlot,
			friSlot,
		},
		DurationMinutes: 60,
		RequiredGender:  "female",
	}

	exactMatch := &BookingMatch{
		Booking:     Booking{ID: 42, TeacherID: 1, BranchID: 1, SubjectID: 1, StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)},
		TeacherName: "Alice",
	}
	store.On("FindExactMatch", mock.Anything, 1, 1, monSlot, 60, shared.None[int]()).Return(exactMatch, nil)
	store.On("FindExactMatch", mock.Anything, 1, 1, friSlot, 60, shared.None[int]()).Return(nil, nil)

	alts := []BookingAlternative{
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

func TestSchedulingService_Confirm_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewSchedulingService(store, store, new(mockEngine))

	req := ConfirmBookingRequest{
		TeacherID:  1,
		BranchID:   1,
		SubjectID:  1,
		StartTime:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		ClientName: "John Doe",
	}

	expected := &Booking{
		ID:         10,
		TeacherID:  1,
		BranchID:   1,
		SubjectID:  1,
		StartTime:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		ClientName: "John Doe",
	}
	store.On("CreateBooking", mock.Anything, req).Return(expected, nil)

	result, err := svc.Confirm(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, 10, result.ID)
	assert.Equal(t, "John Doe", result.ClientName)
	store.AssertExpectations(t)
}

func TestSchedulingService_Confirm_MissingFields(t *testing.T) {
	svc := NewSchedulingService(new(mockStore), new(mockStore), new(mockEngine))

	tests := []struct {
		name string
		req  ConfirmBookingRequest
		err  string
	}{
		{"teacher", ConfirmBookingRequest{BranchID: 1, SubjectID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour), ClientName: "X"}, "teacher_id"},
		{"branch", ConfirmBookingRequest{TeacherID: 1, SubjectID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour), ClientName: "X"}, "branch_id"},
		{"subject", ConfirmBookingRequest{TeacherID: 1, BranchID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour), ClientName: "X"}, "subject_id"},
		{"start_time", ConfirmBookingRequest{TeacherID: 1, BranchID: 1, SubjectID: 1, EndTime: time.Now(), ClientName: "X"}, "start_time"},
		{"end_time", ConfirmBookingRequest{TeacherID: 1, BranchID: 1, SubjectID: 1, StartTime: time.Now(), ClientName: "X"}, "end_time"},
		{"time_reversed", ConfirmBookingRequest{TeacherID: 1, BranchID: 1, SubjectID: 1, StartTime: time.Now().Add(time.Hour), EndTime: time.Now(), ClientName: "X"}, "end_time must be after start_time"},
		{"client_name", ConfirmBookingRequest{TeacherID: 1, BranchID: 1, SubjectID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}, "client_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Confirm(context.Background(), tt.req)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
		})
	}
}

func TestSchedulingService_Cancel_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewSchedulingService(store, store, new(mockEngine))

	store.On("DeleteBooking", mock.Anything, 10).Return(nil)

	err := svc.Cancel(context.Background(), 10)

	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func TestSchedulingService_Cancel_NotFound(t *testing.T) {
	store := new(mockStore)
	svc := NewSchedulingService(store, store, new(mockEngine))

	store.On("DeleteBooking", mock.Anything, 999).Return(&shared.NotFoundError{Msg: "booking 999 not found"})

	err := svc.Cancel(context.Background(), 999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSchedulingService_Cancel_InvalidID(t *testing.T) {
	svc := NewSchedulingService(new(mockStore), new(mockStore), new(mockEngine))

	err := svc.Cancel(context.Background(), 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "booking_id must be positive")
}

func TestSchedulingService_ListAll_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewSchedulingService(store, store, new(mockEngine))

	expected := []Booking{{ID: 1}, {ID: 2}}
	store.On("FindAllBookings", mock.Anything).Return(expected, nil)

	bookings, err := svc.ListAll(context.Background())
	assert.NoError(t, err)
	assert.Len(t, bookings, 2)
}

func TestSchedulingService_ListAll_Error(t *testing.T) {
	store := new(mockStore)
	svc := NewSchedulingService(store, store, new(mockEngine))

	store.On("FindAllBookings", mock.Anything).Return([]Booking{}, assert.AnError)

	_, err := svc.ListAll(context.Background())
	assert.Error(t, err)
}

func TestSchedulingService_GetBranches_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewSchedulingService(store, store, new(mockEngine))

	expected := []Branch{{ID: 1, Name: "Main"}}
	store.On("GetBranches", mock.Anything).Return(expected, nil)

	branches, err := svc.GetBranches(context.Background())
	assert.NoError(t, err)
	assert.Len(t, branches, 1)
}

func TestSchedulingService_GetSubjects_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewSchedulingService(store, store, new(mockEngine))

	expected := []Subject{{ID: 1, Name: "Math"}}
	store.On("GetSubjects", mock.Anything).Return(expected, nil)

	subjects, err := svc.GetSubjects(context.Background())
	assert.NoError(t, err)
	assert.Len(t, subjects, 1)
}
