package scheduling

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockCLPBookingStore struct {
	mock.Mock
}

func (m *mockCLPBookingStore) FindExactMatch(ctx context.Context, subjectID, branchID int, slot shared.WeeklySlot, durationMinutes int, teacherID shared.Option[int], gender string) (*BookingMatch, error) {
	args := m.Called(ctx, subjectID, branchID, slot, durationMinutes, teacherID, gender)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BookingMatch), args.Error(1)
}

func (m *mockCLPBookingStore) FindConflictingBookings(ctx context.Context, teacherID int, startTime, endTime time.Time) ([]Booking, error) {
	args := m.Called(ctx, teacherID, startTime, endTime)
	return args.Get(0).([]Booking), args.Error(1)
}

func (m *mockCLPBookingStore) CreateBooking(ctx context.Context, req ConfirmBookingRequest) (*Booking, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Booking), args.Error(1)
}

func (m *mockCLPBookingStore) DeleteBooking(ctx context.Context, bookingID int) error {
	args := m.Called(ctx, bookingID)
	return args.Error(0)
}

func (m *mockCLPBookingStore) FindAllBookings(ctx context.Context) ([]Booking, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Booking), args.Error(1)
}

type mockCLPTeacherRoster struct {
	mock.Mock
}

func (m *mockCLPTeacherRoster) TeachersBySubject(ctx context.Context, subjectID int) ([]TeacherInfo, error) {
	args := m.Called(ctx, subjectID)
	return args.Get(0).([]TeacherInfo), args.Error(1)
}

func (m *mockCLPTeacherRoster) TeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error) {
	args := m.Called(ctx, teacherID)
	return args.Get(0).([]shared.WeeklySlot), args.Error(1)
}

type mockCLPScorer struct {
	mock.Mock
}

func (m *mockCLPScorer) Score(ctx context.Context, candidate ScorableCandidate) ScoreResult {
	args := m.Called(ctx, candidate)
	return args.Get(0).(ScoreResult)
}

func clpSlot(day int, start, end string) shared.WeeklySlot {
	return shared.WeeklySlot{DayOfWeek: day, Start: shared.TimeHHMM(start), End: shared.TimeHHMM(end)}
}

func TestCLPEngine_Alternatives_NoTeachers(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, nil, slog.Default())

	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{}, nil)

	req := bookingReq(1, 1, clpSlot(0, "09:00", "10:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestCLPEngine_Alternatives_ConflictPruned(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, nil, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{
		clpSlot(0, "09:00", "17:00"),
	}, nil)
	loc := shared.LoadLocation()
	anchor := shared.AnchorDateForDay(0, loc)
	conflict := []Booking{
		{ID: 42, StartTime: anchor.Add(7 * time.Hour), EndTime: anchor.Add(12 * time.Hour)},
	}
	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(conflict, nil)

	req := bookingReq(1, 1, clpSlot(0, "09:00", "10:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.Empty(t, result)
	scorer.AssertNotCalled(t, "Score")
}

func TestCLPEngine_Alternatives_NoConflict(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, nil, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{
		clpSlot(0, "09:00", "17:00"),
	}, nil)
	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]Booking{}, nil)
	scorer.On("Score", mock.Anything, mock.AnythingOfType("ScorableCandidate")).
		Return(ScoreResult{Score: 85, Reasons: []string{"Good match"}})

	req := bookingReq(1, 1, clpSlot(0, "13:00", "14:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(result), 1)
	for _, r := range result {
		assert.Equal(t, 85, r.Score)
		assert.Equal(t, "Alice", r.TeacherName)
	}
}

func TestCLPEngine_Alternatives_Top3ByScore(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, nil, slog.Default())

	teachers := []TeacherInfo{
		{ID: 1, Name: "Alice", Gender: "female"},
		{ID: 2, Name: "Bob", Gender: "female"},
		{ID: 3, Name: "Charlie", Gender: "female"},
		{ID: 4, Name: "Diana", Gender: "female"},
	}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return(teachers, nil)
	for _, tc := range teachers {
		tRoster.On("TeacherAvailability", mock.Anything, tc.ID).Return([]shared.WeeklySlot{
			clpSlot(0, "09:00", "17:00"),
		}, nil)
		bStore.On("FindConflictingBookings", mock.Anything, tc.ID,
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]Booking{}, nil)
	}

	scorer.On("Score", mock.Anything, mock.MatchedBy(func(c ScorableCandidate) bool { return c.Teacher.ID == 1 })).
		Return(ScoreResult{Score: 60, Reasons: []string{"ok"}})
	scorer.On("Score", mock.Anything, mock.MatchedBy(func(c ScorableCandidate) bool { return c.Teacher.ID == 2 })).
		Return(ScoreResult{Score: 50, Reasons: []string{"ok"}})
	scorer.On("Score", mock.Anything, mock.MatchedBy(func(c ScorableCandidate) bool { return c.Teacher.ID == 3 })).
		Return(ScoreResult{Score: 90, Reasons: []string{"ok"}})
	scorer.On("Score", mock.Anything, mock.MatchedBy(func(c ScorableCandidate) bool { return c.Teacher.ID == 4 })).
		Return(ScoreResult{Score: 70, Reasons: []string{"ok"}})

	req := bookingReq(1, 1, clpSlot(0, "13:00", "14:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "Charlie", result[0].TeacherName)
	assert.Equal(t, "Diana", result[1].TeacherName)
	assert.Equal(t, "Alice", result[2].TeacherName)
}

func TestCLPEngine_Alternatives_RequiredGender_ExcludesMismatch(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, nil, slog.Default())

	teachers := []TeacherInfo{
		{ID: 1, Name: "Alice", Gender: "male"},
		{ID: 2, Name: "Betty", Gender: "female"},
	}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return(teachers, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 2).Return([]shared.WeeklySlot{
		clpSlot(0, "09:00", "17:00"),
	}, nil)
	bStore.On("FindConflictingBookings", mock.Anything, 2,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]Booking{}, nil)
	scorer.On("Score", mock.Anything, mock.AnythingOfType("ScorableCandidate")).
		Return(ScoreResult{Score: 80, Reasons: []string{"ok"}})

	req := bookingReq(1, 1, clpSlot(0, "13:00", "14:00"), 60)
	req.RequiredGender = "female"
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	for _, r := range result {
		assert.Equal(t, "Betty", r.TeacherName)
	}
	tRoster.AssertNotCalled(t, "TeacherAvailability", mock.Anything, 1)
	bStore.AssertNotCalled(t, "FindConflictingBookings", mock.Anything, 1, mock.Anything, mock.Anything)

	require.Len(t, teachers, 2, "the roster's own slice must not be mutated by the engine's gender filter")
	assert.Equal(t, "Alice", teachers[0].Name, "in-place filtering would have overwritten this with Betty's data")
}

func TestCLPEngine_Alternatives_RequiredGender_NoMatch_ReturnsEmpty(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, nil, slog.Default())

	teachers := []TeacherInfo{
		{ID: 1, Name: "Alice", Gender: "male"},
	}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return(teachers, nil)

	req := bookingReq(1, 1, clpSlot(0, "13:00", "14:00"), 60)
	req.RequiredGender = "female"
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.Empty(t, result)
	scorer.AssertNotCalled(t, "Score")
}
