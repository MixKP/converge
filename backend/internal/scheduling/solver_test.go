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
	start := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
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
	start := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
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

	anchor := shared.AnchorDateForDay(0, loc)
	start := anchor.Add(10 * time.Hour)
	end := anchor.Add(11 * time.Hour)

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

func TestSolverWithModel(t *testing.T) {
	teachers := []any{
		TeacherInfo{ID: 1, Name: "Alice"},
		TeacherInfo{ID: 2, Name: "Bob"},
	}
	offsets := []any{
		timeWindow{start: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), end: time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)},
		timeWindow{start: time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC), end: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)},
	}

	model := NewModel()
	model.AddVariable("teacher", teachers)
	model.AddVariable("offset", offsets)

	model.AddConstraint(func(a Assignment) bool {
		t := a.Value("teacher").(TeacherInfo)
		o := a.Value("offset").(timeWindow)
		if t.Name == "Alice" && o.start.Hour() == 10 {
			return false
		}
		return true
	})

	model.SetObjective(func(a Assignment) (int, []string) {
		t := a.Value("teacher").(TeacherInfo)
		return t.ID * 10, nil
	})

	solver := &Solver{}
	results := solver.Solve(*model, 3)

	assert.Len(t, results, 3)
	assert.Equal(t, 2, results[0].Value("teacher").(TeacherInfo).ID)
	assert.Equal(t, 1, results[2].Value("teacher").(TeacherInfo).ID)
	assert.Equal(t, 20, results[0].Score)
	assert.Equal(t, 10, results[2].Score)
}

func TestSolverWithModelTop2(t *testing.T) {
	values := []any{1, 2, 3}

	model := NewModel()
	model.AddVariable("x", values)
	model.SetObjective(func(a Assignment) (int, []string) {
		return a.Value("x").(int) * 10, nil
	})

	solver := &Solver{}
	results := solver.Solve(*model, 2)

	assert.Len(t, results, 2)
	assert.Equal(t, 30, results[0].Score)
	assert.Equal(t, 20, results[1].Score)
}

func TestSolverNoSolutions(t *testing.T) {
	model := NewModel()
	model.AddVariable("x", []any{1, 2})
	model.AddConstraint(func(a Assignment) bool { return false })

	solver := &Solver{}
	results := solver.Solve(*model, 3)

	assert.Empty(t, results)
}

func TestSolverEmptyModel(t *testing.T) {
	model := NewModel()
	solver := &Solver{}
	results := solver.Solve(*model, 3)

	assert.Len(t, results, 1)
}

func TestSolverSortOrder(t *testing.T) {
	values := []any{5, 1, 3, 2, 4}

	model := NewModel()
	model.AddVariable("v", values)
	model.SetObjective(func(a Assignment) (int, []string) {
		return a.Value("v").(int), nil
	})

	solver := &Solver{}
	results := solver.Solve(*model, 3)

	assert.Equal(t, 5, results[0].Score)
	assert.Equal(t, 4, results[1].Score)
	assert.Equal(t, 3, results[2].Score)
}
