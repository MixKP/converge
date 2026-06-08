package scheduling

import (
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFitsAvailabilityWithinSlot(t *testing.T) {
	slots := []shared.WeeklySlot{
		{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("17:00")},
	}
	start := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC) // Monday
	end := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)

	assert.True(t, fitsAvailability(slots, start, end))
}

func TestFitsAvailabilityExactBoundary(t *testing.T) {
	slots := []shared.WeeklySlot{
		{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("17:00")},
	}
	start := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 17, 0, 0, 0, time.UTC)

	assert.True(t, fitsAvailability(slots, start, end))
}

func TestFitsAvailabilityBeforeSlot(t *testing.T) {
	slots := []shared.WeeklySlot{
		{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("17:00")},
	}
	start := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	assert.False(t, fitsAvailability(slots, start, end))
}

func TestFitsAvailabilityAfterSlot(t *testing.T) {
	slots := []shared.WeeklySlot{
		{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("17:00")},
	}
	start := time.Date(2026, 6, 1, 17, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 18, 0, 0, 0, time.UTC)

	assert.False(t, fitsAvailability(slots, start, end))
}

func TestFitsAvailabilityWrongDay(t *testing.T) {
	slots := []shared.WeeklySlot{
		{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("17:00")},
	}
	start := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC) // Tuesday
	end := time.Date(2026, 6, 2, 11, 0, 0, 0, time.UTC)

	assert.False(t, fitsAvailability(slots, start, end))
}

func TestFitsAvailabilityEmptySlots(t *testing.T) {
	assert.False(t, fitsAvailability([]shared.WeeklySlot{},
		time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)))
}

func TestFitsAvailabilityBangkokTimezone(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Bangkok")
	require.NoError(t, err)

	anchor := shared.AnchorDateForDay(0, loc) // next Monday in Bangkok
	start := anchor.Add(10 * time.Hour)       // 10:00 Bangkok
	end := anchor.Add(11 * time.Hour)         // 11:00 Bangkok

	slots := []shared.WeeklySlot{
		{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("17:00")},
	}

	assert.True(t, fitsAvailability(slots, start, end))
}

func TestGenerateOffsetsStandardRange(t *testing.T) {
	prefStart := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	duration := 60 * time.Minute

	offsets := generateOffsets(prefStart, duration)

	assert.Len(t, offsets, 9)
	assert.Equal(t, "11:00", offsets[0].start.Format("15:04"))
	assert.Equal(t, "12:00", offsets[0].end.Format("15:04"))
	assert.Equal(t, "15:00", offsets[8].start.Format("15:04"))
	assert.Equal(t, "16:00", offsets[8].end.Format("15:04"))
}

func TestGenerateOffsetsZeroDuration(t *testing.T) {
	prefStart := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)

	offsets := generateOffsets(prefStart, 0)

	assert.Len(t, offsets, 9)
}

func TestDeduplicateByTeacherSolver(t *testing.T) {
	candidates := []solverCandidate{
		{Teacher: TeacherInfo{ID: 1, Name: "Alice"}, Score: 80},
		{Teacher: TeacherInfo{ID: 2, Name: "Bob"}, Score: 90},
		{Teacher: TeacherInfo{ID: 1, Name: "Alice"}, Score: 70},
	}

	result := deduplicateByTeacherSolver(candidates)

	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].Teacher.ID)
	assert.Equal(t, 2, result[1].Teacher.ID)
}

func TestDeduplicateByTeacherSolverAllUnique(t *testing.T) {
	candidates := []solverCandidate{
		{Teacher: TeacherInfo{ID: 1, Name: "Alice"}, Score: 80},
		{Teacher: TeacherInfo{ID: 2, Name: "Bob"}, Score: 90},
	}

	result := deduplicateByTeacherSolver(candidates)

	assert.Len(t, result, 2)
}

func TestDeduplicateByTeacherSolverEmpty(t *testing.T) {
	result := deduplicateByTeacherSolver(nil)
	assert.Empty(t, result)
}

func TestTimeOfDay(t *testing.T) {
	ts := time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC)
	result := timeOfDay(ts)

	assert.Equal(t, 0, result.Year())
	assert.Equal(t, time.January, result.Month())
	assert.Equal(t, 1, result.Day())
	assert.Equal(t, 10, result.Hour())
	assert.Equal(t, 30, result.Minute())
}

func TestTimeOfDayBangkok(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	ts := time.Date(2026, 6, 1, 18, 0, 0, 0, loc)
	result := timeOfDay(ts)

	assert.Equal(t, 18, result.Hour(), "timeOfDay should use local hour")
	assert.Equal(t, 0, result.Minute())
}
