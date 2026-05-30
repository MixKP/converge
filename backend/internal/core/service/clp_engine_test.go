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

type mockBookingRepo struct {
	mock.Mock
}

func (m *mockBookingRepo) FindExactMatch(ctx context.Context, req models.BookingRequest) (*ports.BookingMatch, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.BookingMatch), args.Error(1)
}

func (m *mockBookingRepo) FindTeachersBySubject(ctx context.Context, subjectID int) ([]models.Teacher, error) {
	args := m.Called(ctx, subjectID)
	return args.Get(0).([]models.Teacher), args.Error(1)
}

func (m *mockBookingRepo) FindConflictingBookings(ctx context.Context, teacherID int, startTime, endTime time.Time) ([]models.Booking, error) {
	args := m.Called(ctx, teacherID, startTime, endTime)
	return args.Get(0).([]models.Booking), args.Error(1)
}

func (m *mockBookingRepo) FindTeacherAvailability(ctx context.Context, teacherID int) ([]models.WeeklySlot, error) {
	args := m.Called(ctx, teacherID)
	return args.Get(0).([]models.WeeklySlot), args.Error(1)
}

type mockScorer struct {
	mock.Mock
}

func (m *mockScorer) Score(ctx context.Context, candidate ports.ScorableCandidate) ports.ScoreResult {
	args := m.Called(ctx, candidate)
	return args.Get(0).(ports.ScoreResult)
}

func newEngine(repo *mockBookingRepo, scorer *mockScorer) *CLPEngine {
	return &CLPEngine{repo: repo, scorer: scorer}
}

func TestCLPEngine_TeacherWithoutSubject_ReturnsEmpty(t *testing.T) {
	repo := new(mockBookingRepo)
	scorer := new(mockScorer)
	engine := newEngine(repo, scorer)

	repo.On("FindTeachersBySubject", mock.Anything, 1).Return([]models.Teacher{}, nil)

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), 60)

	result, err := engine.FindAlternatives(context.Background(), req)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestCLPEngine_TeacherWithConflict_Pruned(t *testing.T) {
	repo := new(mockBookingRepo)
	scorer := new(mockScorer)
	engine := newEngine(repo, scorer)

	teacher := models.Teacher{ID: 1, Name: "Alice"}
	repo.On("FindTeachersBySubject", mock.Anything, 1).Return([]models.Teacher{teacher}, nil)
	repo.On("FindTeacherAvailability", mock.Anything, 1).Return([]models.WeeklySlot{
		{DayOfWeek: 0, Start: "09:00", End: "17:00"},
	}, nil)
	conflict := []models.Booking{{ID: 42}}
	repo.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(conflict, nil)

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), 60)

	result, err := engine.FindAlternatives(context.Background(), req)

	assert.NoError(t, err)
	assert.Empty(t, result)
	scorer.AssertNotCalled(t, "Score")
}

func TestCLPEngine_TeacherOutsideAvailability_Pruned(t *testing.T) {
	repo := new(mockBookingRepo)
	scorer := new(mockScorer)
	engine := newEngine(repo, scorer)

	teacher := models.Teacher{ID: 1, Name: "Alice"}
	repo.On("FindTeachersBySubject", mock.Anything, 1).Return([]models.Teacher{teacher}, nil)
	repo.On("FindTeacherAvailability", mock.Anything, 1).Return([]models.WeeklySlot{
		{DayOfWeek: 0, Start: "14:00", End: "16:00"},
	}, nil)
	repo.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Booking{}, nil)

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), 60)

	result, err := engine.FindAlternatives(context.Background(), req)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestCLPEngine_NoConflict_CandidateScored(t *testing.T) {
	repo := new(mockBookingRepo)
	scorer := new(mockScorer)
	engine := newEngine(repo, scorer)

	teacher := models.Teacher{ID: 1, Name: "Alice"}
	repo.On("FindTeachersBySubject", mock.Anything, 1).Return([]models.Teacher{teacher}, nil)
	repo.On("FindTeacherAvailability", mock.Anything, 1).Return([]models.WeeklySlot{
		{DayOfWeek: 0, Start: "09:00", End: "17:00"},
	}, nil)
	repo.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Booking{}, nil)
	scorer.On("Score", mock.Anything, mock.AnythingOfType("ports.ScorableCandidate")).
		Return(ports.ScoreResult{Score: 85, Reasons: []string{"Good match"}})

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC), 60)

	result, err := engine.FindAlternatives(context.Background(), req)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(result), 1)
	for _, r := range result {
		assert.Equal(t, 85, r.Score)
		assert.Equal(t, "Alice", r.TeacherName)
	}
}

func TestCLPEngine_ReturnsTop3_ByScore(t *testing.T) {
	repo := new(mockBookingRepo)
	scorer := new(mockScorer)
	engine := newEngine(repo, scorer)

	teachers := []models.Teacher{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
		{ID: 4, Name: "Diana"},
	}
	repo.On("FindTeachersBySubject", mock.Anything, 1).Return(teachers, nil)
	for _, tc := range teachers {
		repo.On("FindTeacherAvailability", mock.Anything, tc.ID).Return([]models.WeeklySlot{
			{DayOfWeek: 0, Start: "09:00", End: "17:00"},
		}, nil)
		repo.On("FindConflictingBookings", mock.Anything, tc.ID,
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Booking{}, nil)
	}

	scorer.On("Score", mock.Anything, mock.MatchedBy(func(c ports.ScorableCandidate) bool { return c.Teacher.ID == 1 })).
		Return(ports.ScoreResult{Score: 60, Reasons: []string{"ok"}})
	scorer.On("Score", mock.Anything, mock.MatchedBy(func(c ports.ScorableCandidate) bool { return c.Teacher.ID == 2 })).
		Return(ports.ScoreResult{Score: 50, Reasons: []string{"ok"}})
	scorer.On("Score", mock.Anything, mock.MatchedBy(func(c ports.ScorableCandidate) bool { return c.Teacher.ID == 3 })).
		Return(ports.ScoreResult{Score: 90, Reasons: []string{"ok"}})
	scorer.On("Score", mock.Anything, mock.MatchedBy(func(c ports.ScorableCandidate) bool { return c.Teacher.ID == 4 })).
		Return(ports.ScoreResult{Score: 70, Reasons: []string{"ok"}})

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC), 60)

	result, err := engine.FindAlternatives(context.Background(), req)

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "Charlie", result[0].TeacherName)
	assert.Equal(t, "Diana", result[1].TeacherName)
	assert.Equal(t, "Alice", result[2].TeacherName)
}

func TestCLPEngine_RoomCommuteSkipped_WhenNil(t *testing.T) {
	repo := new(mockBookingRepo)
	scorer := new(mockScorer)
	engine := newEngine(repo, scorer)

	teacher := models.Teacher{ID: 1, Name: "Alice"}
	repo.On("FindTeachersBySubject", mock.Anything, 1).Return([]models.Teacher{teacher}, nil)
	repo.On("FindTeacherAvailability", mock.Anything, 1).Return([]models.WeeklySlot{
		{DayOfWeek: 0, Start: "09:00", End: "17:00"},
	}, nil)
	repo.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Booking{}, nil)
	scorer.On("Score", mock.Anything, mock.Anything).Return(ports.ScoreResult{Score: 85, Reasons: []string{"ok"}})

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC), 60)

	result, err := engine.FindAlternatives(context.Background(), req)

	assert.NoError(t, err)
	for _, r := range result {
		assert.Nil(t, r.RoomAvailable)
		assert.Nil(t, r.CommuteMinutes)
	}
}

func TestCLPEngine_NoPreferredTime_GeneratesAllSlots(t *testing.T) {
	repo := new(mockBookingRepo)
	scorer := new(mockScorer)
	engine := newEngine(repo, scorer)

	teacher := models.Teacher{ID: 1, Name: "Alice"}
	repo.On("FindTeachersBySubject", mock.Anything, 1).Return([]models.Teacher{teacher}, nil)
	repo.On("FindTeacherAvailability", mock.Anything, 1).Return([]models.WeeklySlot{
		{DayOfWeek: 0, Start: "09:00", End: "11:00"},
	}, nil)
	repo.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Booking{}, nil)
	scorer.On("Score", mock.Anything, mock.Anything).Return(ports.ScoreResult{Score: 50, Reasons: []string{"ok"}})

	req := bookingReq(1, 1, time.Time{}, 60)

	result, err := engine.FindAlternatives(context.Background(), req)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.True(t, len(result) >= 1)
}

func TestCLPEngine_SlotDuration_Respected(t *testing.T) {
	repo := new(mockBookingRepo)
	scorer := new(mockScorer)
	engine := newEngine(repo, scorer)

	teacher := models.Teacher{ID: 1, Name: "Alice"}
	repo.On("FindTeachersBySubject", mock.Anything, 1).Return([]models.Teacher{teacher}, nil)
	repo.On("FindTeacherAvailability", mock.Anything, 1).Return([]models.WeeklySlot{
		{DayOfWeek: 0, Start: "09:00", End: "10:00"},
	}, nil)
	repo.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Booking{}, nil)

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), 90)
	result, err := engine.FindAlternatives(context.Background(), req)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestCLPEngine_ConflictQuery_CalledForEachCandidateSlot(t *testing.T) {
	repo := new(mockBookingRepo)
	scorer := new(mockScorer)
	engine := newEngine(repo, scorer)

	teacher := models.Teacher{ID: 1, Name: "Alice"}
	repo.On("FindTeachersBySubject", mock.Anything, 1).Return([]models.Teacher{teacher}, nil)
	repo.On("FindTeacherAvailability", mock.Anything, 1).Return([]models.WeeklySlot{
		{DayOfWeek: 0, Start: "12:00", End: "14:00"},
	}, nil)
	repo.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Booking{}, nil)
	scorer.On("Score", mock.Anything, mock.Anything).Return(ports.ScoreResult{Score: 50, Reasons: []string{"ok"}})

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC), 60)
	result, err := engine.FindAlternatives(context.Background(), req)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	repo.AssertNumberOfCalls(t, "FindConflictingBookings", 3)
}

func TestCLPEngine_MultipleConflicts_DayFiltered(t *testing.T) {
	repo := new(mockBookingRepo)
	scorer := new(mockScorer)
	engine := newEngine(repo, scorer)

	teacher := models.Teacher{ID: 1, Name: "Alice"}
	repo.On("FindTeachersBySubject", mock.Anything, 1).Return([]models.Teacher{teacher}, nil)
	repo.On("FindTeacherAvailability", mock.Anything, 1).Return([]models.WeeklySlot{
		{DayOfWeek: 0, Start: "09:00", End: "09:30"},
		{DayOfWeek: 2, Start: "09:00", End: "17:00"},
	}, nil)

	req := bookingReq(1, 1, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), 60)
	result, err := engine.FindAlternatives(context.Background(), req)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestCLPEngine_NoPreferredTime_ScoresOnDefaultDate(t *testing.T) {
	repo := new(mockBookingRepo)
	scorer := new(mockScorer)
	engine := newEngine(repo, scorer)

	teacher := models.Teacher{ID: 1, Name: "Alice"}
	repo.On("FindTeachersBySubject", mock.Anything, 1).Return([]models.Teacher{teacher}, nil)
	repo.On("FindTeacherAvailability", mock.Anything, 1).Return([]models.WeeklySlot{
		{DayOfWeek: 1, Start: "09:00", End: "17:00"}, // Tuesday (June 1 is Monday, skip)
	}, nil)
	repo.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Booking{}, nil)

	req := bookingReq(1, 1, time.Time{}, 60)
	result, err := engine.FindAlternatives(context.Background(), req)

	assert.NoError(t, err)
	assert.Empty(t, result) // default date June 1 is Monday, slot is Tuesday
}

func bookingReq(subjectID, branchID int, preferredStart time.Time, durationMinutes int) models.BookingRequest {
	return models.BookingRequest{
		SubjectID:       subjectID,
		BranchID:        branchID,
		PreferredStart:  preferredStart,
		DurationMinutes: durationMinutes,
	}
}
