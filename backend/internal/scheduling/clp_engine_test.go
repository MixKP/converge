package scheduling

import (
	"context"
	"errors"
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

func (m *mockCLPBookingStore) FindBookingsByBranch(ctx context.Context, branchID int, startTime, endTime time.Time) ([]Booking, error) {
	args := m.Called(ctx, branchID, startTime, endTime)
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

type mockCLPBranchCapacity struct {
	mock.Mock
}

func (m *mockCLPBranchCapacity) GetCapacity(ctx context.Context, branchID int) (int, error) {
	args := m.Called(ctx, branchID)
	return args.Int(0), args.Error(1)
}

// zeroBranchCapacity is a no-op BranchCapacityCheck for tests that don't
// exercise branch-capacity behavior; capacity=0 means "unenforced".
type zeroBranchCapacity struct{}

func (zeroBranchCapacity) GetCapacity(ctx context.Context, branchID int) (int, error) {
	return 0, nil
}

func clpSlot(day int, start, end string) shared.WeeklySlot {
	return shared.WeeklySlot{DayOfWeek: day, Start: shared.TimeHHMM(start), End: shared.TimeHHMM(end)}
}

func TestCLPEngine_Alternatives_NoTeachers(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, zeroBranchCapacity{}, slog.Default())

	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{}, nil)

	req := bookingReq(1, 1, clpSlot(0, "09:00", "10:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestCLPEngine_Alternatives_NoTeachers_SkipsBranchCapacityLookup(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	branchCap := new(mockCLPBranchCapacity)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, branchCap, slog.Default())

	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{}, nil)

	req := bookingReq(1, 1, clpSlot(0, "09:00", "10:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.Empty(t, result)
	branchCap.AssertNotCalled(t, "GetCapacity", mock.Anything, mock.Anything)
	bStore.AssertNotCalled(t, "FindBookingsByBranch", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCLPEngine_Alternatives_ConflictPruned(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, zeroBranchCapacity{}, slog.Default())

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
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, zeroBranchCapacity{}, slog.Default())

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
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, zeroBranchCapacity{}, slog.Default())

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
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, zeroBranchCapacity{}, slog.Default())

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
	branchCap := new(mockCLPBranchCapacity)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, branchCap, slog.Default())

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
	branchCap.AssertNotCalled(t, "GetCapacity", mock.Anything, mock.Anything)
}

func TestCLPEngine_Alternatives_BranchAtCapacity_Pruned(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	branchCap := new(mockCLPBranchCapacity)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, branchCap, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{
		clpSlot(0, "09:00", "17:00"),
	}, nil)
	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]Booking{}, nil)

	branchCap.On("GetCapacity", mock.Anything, 1).Return(1, nil)
	loc := shared.LoadLocation()
	anchor := shared.AnchorDateForDay(0, loc)
	existingBranchBookings := []Booking{
		{ID: 99, StartTime: anchor.Add(7 * time.Hour), EndTime: anchor.Add(12 * time.Hour)},
	}
	bStore.On("FindBookingsByBranch", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(existingBranchBookings, nil)

	req := bookingReq(1, 1, clpSlot(0, "09:00", "10:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.Empty(t, result)
	scorer.AssertNotCalled(t, "Score")
}

func TestCLPEngine_Alternatives_BranchUnderCapacity_Allowed(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	branchCap := new(mockCLPBranchCapacity)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, branchCap, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{
		clpSlot(0, "09:00", "17:00"),
	}, nil)
	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]Booking{}, nil)

	branchCap.On("GetCapacity", mock.Anything, 1).Return(2, nil)
	loc := shared.LoadLocation()
	anchor := shared.AnchorDateForDay(0, loc)
	existingBranchBookings := []Booking{
		{ID: 99, StartTime: anchor.Add(7 * time.Hour), EndTime: anchor.Add(12 * time.Hour)},
	}
	bStore.On("FindBookingsByBranch", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(existingBranchBookings, nil)
	scorer.On("Score", mock.Anything, mock.AnythingOfType("ScorableCandidate")).
		Return(ScoreResult{Score: 85, Reasons: []string{"Good match"}})

	req := bookingReq(1, 1, clpSlot(0, "09:00", "10:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestCLPEngine_Alternatives_BranchCapacityUnconfigured_NotEnforced(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	branchCap := new(mockCLPBranchCapacity)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, branchCap, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{
		clpSlot(0, "09:00", "17:00"),
	}, nil)
	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]Booking{}, nil)

	branchCap.On("GetCapacity", mock.Anything, 1).Return(0, nil)
	scorer.On("Score", mock.Anything, mock.AnythingOfType("ScorableCandidate")).
		Return(ScoreResult{Score: 85, Reasons: []string{"Good match"}})

	req := bookingReq(1, 1, clpSlot(0, "09:00", "10:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	bStore.AssertNotCalled(t, "FindBookingsByBranch", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCLPEngine_Alternatives_BranchCapacityLookupError_DegradesGracefully(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	branchCap := new(mockCLPBranchCapacity)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, branchCap, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{
		clpSlot(0, "09:00", "17:00"),
	}, nil)
	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]Booking{}, nil)

	branchCap.On("GetCapacity", mock.Anything, 1).Return(0, errors.New("db unreachable"))
	scorer.On("Score", mock.Anything, mock.AnythingOfType("ScorableCandidate")).
		Return(ScoreResult{Score: 85, Reasons: []string{"Good match"}})

	req := bookingReq(1, 1, clpSlot(0, "09:00", "10:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	bStore.AssertNotCalled(t, "FindBookingsByBranch", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCLPEngine_Alternatives_BranchBookingsLookupError_DegradesGracefully(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	branchCap := new(mockCLPBranchCapacity)
	engine := NewCLPEngine(bStore, tRoster, scorer, nil, branchCap, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{
		clpSlot(0, "09:00", "17:00"),
	}, nil)
	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]Booking{}, nil)

	branchCap.On("GetCapacity", mock.Anything, 1).Return(1, nil)
	bStore.On("FindBookingsByBranch", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return(([]Booking)(nil), errors.New("db unreachable"))
	scorer.On("Score", mock.Anything, mock.AnythingOfType("ScorableCandidate")).
		Return(ScoreResult{Score: 85, Reasons: []string{"Good match"}})

	req := bookingReq(1, 1, clpSlot(0, "09:00", "10:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// stubCommute is a CommuteProvider returning a fixed commute duration; the
// engine applies the same-branch-means-zero rule itself.
type stubCommute struct{ minutes int }

func (s stubCommute) DefaultCommute(ctx context.Context) (time.Duration, error) {
	return time.Duration(s.minutes) * time.Minute, nil
}

// Example A: a prior different-branch booking pads forward, so the requested
// 13:00 is blocked and the nearest valid slot 13:30 is offered with the
// commute reported.
func TestCLPEngine_Alternatives_PriorDifferentBranch_ShiftsLater(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, stubCommute{minutes: 30}, zeroBranchCapacity{}, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{clpSlot(0, "09:00", "17:00")}, nil)

	loc := shared.LoadLocation()
	anchor := shared.AnchorDateForDay(0, loc)
	conflicts := []Booking{{ID: 7, BranchID: 2, StartTime: anchor.Add(12 * time.Hour), EndTime: anchor.Add(13 * time.Hour)}}
	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(conflicts, nil)
	scorer.On("Score", mock.Anything, mock.AnythingOfType("ScorableCandidate")).
		Return(ScoreResult{Score: 80, Reasons: []string{"ok"}})

	req := bookingReq(1, 1, clpSlot(0, "13:00", "14:00"), 60) // request branch 1, conflict branch 2
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	require.NotEmpty(t, result)
	want := anchor.Add(13*time.Hour + 30*time.Minute)
	assert.True(t, result[0].StartTime.Equal(want), "want 13:30 got %s", result[0].StartTime)
	mins, ok := result[0].CommuteMinutes.Value()
	assert.True(t, ok)
	assert.Equal(t, 30, mins)
	assert.Equal(t, 80, result[0].Score) // display score is the raw quality
}

// Example B: a following different-branch booking pads backward, so the
// requested 12:00 is blocked and the nearest earlier slot 11:30 is offered.
func TestCLPEngine_Alternatives_FollowingDifferentBranch_ShiftsEarlier(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, stubCommute{minutes: 30}, zeroBranchCapacity{}, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{clpSlot(0, "09:00", "17:00")}, nil)

	loc := shared.LoadLocation()
	anchor := shared.AnchorDateForDay(0, loc)
	conflicts := []Booking{{ID: 8, BranchID: 2, StartTime: anchor.Add(13 * time.Hour), EndTime: anchor.Add(14 * time.Hour)}}
	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(conflicts, nil)
	scorer.On("Score", mock.Anything, mock.AnythingOfType("ScorableCandidate")).
		Return(ScoreResult{Score: 80, Reasons: []string{"ok"}})

	req := bookingReq(1, 1, clpSlot(0, "12:00", "13:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	require.NotEmpty(t, result)
	want := anchor.Add(11*time.Hour + 30*time.Minute)
	assert.True(t, result[0].StartTime.Equal(want), "want 11:30 got %s", result[0].StartTime)
}

// Proximity dominates the quality score: even when a far slot scores much
// higher on quality, the nearest valid slot is still offered.
func TestCLPEngine_Alternatives_ProximityBeatsQuality(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, stubCommute{minutes: 30}, zeroBranchCapacity{}, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{clpSlot(0, "09:00", "17:00")}, nil)

	loc := shared.LoadLocation()
	anchor := shared.AnchorDateForDay(0, loc)
	conflicts := []Booking{{ID: 9, BranchID: 2, StartTime: anchor.Add(12 * time.Hour), EndTime: anchor.Add(13 * time.Hour)}}
	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(conflicts, nil)

	far := anchor.Add(15 * time.Hour)
	scorer.On("Score", mock.Anything, mock.MatchedBy(func(c ScorableCandidate) bool { return c.StartTime.Equal(far) })).
		Return(ScoreResult{Score: 100, Reasons: []string{"far but high"}})
	scorer.On("Score", mock.Anything, mock.AnythingOfType("ScorableCandidate")).
		Return(ScoreResult{Score: 10, Reasons: []string{"near but low"}})

	req := bookingReq(1, 1, clpSlot(0, "13:00", "14:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	require.NotEmpty(t, result)
	want := anchor.Add(13*time.Hour + 30*time.Minute)
	assert.True(t, result[0].StartTime.Equal(want), "nearest slot must win despite lower quality; got %s", result[0].StartTime)
}

// A same-branch neighbor needs no travel time: the requested slot stays put
// and no commute is reported.
func TestCLPEngine_Alternatives_SameBranchNeighbor_NoShift(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, stubCommute{minutes: 30}, zeroBranchCapacity{}, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{clpSlot(0, "09:00", "17:00")}, nil)

	loc := shared.LoadLocation()
	anchor := shared.AnchorDateForDay(0, loc)
	conflicts := []Booking{{ID: 10, BranchID: 1, StartTime: anchor.Add(12 * time.Hour), EndTime: anchor.Add(13 * time.Hour)}}
	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(conflicts, nil)
	scorer.On("Score", mock.Anything, mock.AnythingOfType("ScorableCandidate")).
		Return(ScoreResult{Score: 80, Reasons: []string{"ok"}})

	req := bookingReq(1, 1, clpSlot(0, "13:00", "14:00"), 60)
	result, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])

	assert.NoError(t, err)
	require.NotEmpty(t, result)
	want := anchor.Add(13 * time.Hour)
	assert.True(t, result[0].StartTime.Equal(want), "want 13:00 (no shift) got %s", result[0].StartTime)
	_, ok := result[0].CommuteMinutes.Value()
	assert.False(t, ok, "no commute should be reported for a same-branch neighbor")
}

func TestCLPEngine_CommuteConflict(t *testing.T) {
	loc := shared.LoadLocation()
	anchor := shared.AnchorDateForDay(0, loc)
	start := anchor.Add(13 * time.Hour)
	end := anchor.Add(14 * time.Hour)
	priorDiffBranch := []Booking{{ID: 1, BranchID: 2, StartTime: anchor.Add(12 * time.Hour), EndTime: anchor.Add(13 * time.Hour)}}

	t.Run("different-branch adjacent booking blocks", func(t *testing.T) {
		bStore := new(mockCLPBookingStore)
		engine := NewCLPEngine(bStore, nil, nil, stubCommute{minutes: 30}, zeroBranchCapacity{}, slog.Default())
		bStore.On("FindConflictingBookings", mock.Anything, 1,
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(priorDiffBranch, nil)

		blocked, err := engine.CommuteConflict(context.Background(), 1, 1, start, end)
		assert.NoError(t, err)
		assert.True(t, blocked)
	})

	t.Run("same-branch adjacent booking does not block", func(t *testing.T) {
		bStore := new(mockCLPBookingStore)
		engine := NewCLPEngine(bStore, nil, nil, stubCommute{minutes: 30}, zeroBranchCapacity{}, slog.Default())
		sameBranch := []Booking{{ID: 1, BranchID: 1, StartTime: anchor.Add(12 * time.Hour), EndTime: anchor.Add(13 * time.Hour)}}
		bStore.On("FindConflictingBookings", mock.Anything, 1,
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(sameBranch, nil)

		blocked, err := engine.CommuteConflict(context.Background(), 1, 1, start, end)
		assert.NoError(t, err)
		assert.False(t, blocked)
	})

	t.Run("nil commute never blocks", func(t *testing.T) {
		bStore := new(mockCLPBookingStore)
		engine := NewCLPEngine(bStore, nil, nil, nil, zeroBranchCapacity{}, slog.Default())

		blocked, err := engine.CommuteConflict(context.Background(), 1, 1, start, end)
		assert.NoError(t, err)
		assert.False(t, blocked)
		bStore.AssertNotCalled(t, "FindConflictingBookings", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

// Regression for the commute-aware fetch window: the conflict lookup must be
// widened by the commute pad so a different-branch booking just outside the
// candidate range can still block an edge candidate. The mock only matches the
// widened bounds, so a non-widened window fails the test.
func TestCLPEngine_Alternatives_ConflictWindowWidenedByCommutePad(t *testing.T) {
	bStore := new(mockCLPBookingStore)
	tRoster := new(mockCLPTeacherRoster)
	scorer := new(mockCLPScorer)
	engine := NewCLPEngine(bStore, tRoster, scorer, stubCommute{minutes: 30}, zeroBranchCapacity{}, slog.Default())

	teacher := TeacherInfo{ID: 1, Name: "Alice", Gender: "female"}
	tRoster.On("TeachersBySubject", mock.Anything, 1).Return([]TeacherInfo{teacher}, nil)
	tRoster.On("TeacherAvailability", mock.Anything, 1).Return([]shared.WeeklySlot{clpSlot(0, "09:00", "17:00")}, nil)

	loc := shared.LoadLocation()
	anchor := shared.AnchorDateForDay(0, loc)
	prefStart := anchor.Add(13 * time.Hour) // request slot 13:00-14:00, duration 60m
	wantStart := prefStart.Add(-CandidateLookbehind).Add(-30 * time.Minute)
	wantEnd := prefStart.Add(CandidateLookahead).Add(60 * time.Minute).Add(30 * time.Minute)

	bStore.On("FindConflictingBookings", mock.Anything, 1,
		mock.MatchedBy(func(ts time.Time) bool { return ts.Equal(wantStart) }),
		mock.MatchedBy(func(ts time.Time) bool { return ts.Equal(wantEnd) })).Return([]Booking{}, nil)
	scorer.On("Score", mock.Anything, mock.Anything).Return(ScoreResult{Score: 80, Reasons: []string{"ok"}})

	req := bookingReq(1, 1, clpSlot(0, "13:00", "14:00"), 60)
	_, err := engine.FindAlternativesForSlot(context.Background(), req, req.PreferredSlots[0])
	assert.NoError(t, err)
	bStore.AssertExpectations(t)
}
