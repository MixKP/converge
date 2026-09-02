package db

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The seeded timetable has to satisfy the bookings EXCLUDE constraint before it
// reaches Postgres, and it has to actually exercise the admin schedule
// dashboard's filters — a timetable covering one teacher would pass the
// constraint and still be useless.
func TestBookingSeeds(t *testing.T) {
	seeds := bookingSeeds()

	require.NoError(t, validateBookingSeeds(seeds), "seeded timetable double-books a teacher")

	teachers := map[int]bool{}
	students := map[string]bool{}
	var past, upcoming int
	for _, s := range seeds {
		teachers[s.teacher] = true
		students[s.student] = true
		if s.dayOffset < 0 {
			past++
		} else if s.dayOffset > 0 {
			upcoming++
		}
		assert.Truef(t, s.hour >= 8 && s.hour <= 16,
			"class at %02d:00 falls outside teaching hours", s.hour)
	}

	assert.Len(t, teachers, 5, "every seeded teacher should have classes")
	assert.Len(t, students, 4, "every demo student should have classes")
	assert.GreaterOrEqual(t, past, 5, "the past tab needs classes to show")
	assert.GreaterOrEqual(t, upcoming, 5, "the upcoming tab needs classes to show")

	// David (index 3) is seeded as deactivated, so he must not be teaching a
	// class that is still to come.
	for _, s := range seeds {
		if s.teacher == 3 {
			assert.Negativef(t, s.dayOffset,
				"deactivated teacher booked on day %+d", s.dayOffset)
		}
	}
}

func TestValidateBookingSeedsRejectsDoubleBooking(t *testing.T) {
	err := validateBookingSeeds([]bookingSeed{
		{student: "student1", teacher: 0, dayOffset: 1, hour: 9},
		{student: "student2", teacher: 0, dayOffset: 1, hour: 9},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "double-book")

	// The same hour for a different teacher, and the same teacher at a
	// different hour, are both legitimate.
	assert.NoError(t, validateBookingSeeds([]bookingSeed{
		{student: "student1", teacher: 0, dayOffset: 1, hour: 9},
		{student: "student2", teacher: 1, dayOffset: 1, hour: 9},
		{student: "student3", teacher: 0, dayOffset: 1, hour: 10},
	}))
}

// withCommuteDemo resolves weekday-pinned classes against the run date, so the
// merged timetable is different on every day of the week. Any one of those days
// producing a double-booking would fail the whole seed, so all seven are checked.
func TestWithCommuteDemoIsValidOnEveryWeekday(t *testing.T) {
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC) // a Sunday
	require.Equal(t, time.Sunday, base.Weekday())

	for i := range 7 {
		day := base.AddDate(0, 0, i)
		merged := withCommuteDemo(bookingSeeds(), day)

		require.NoErrorf(t, validateBookingSeeds(merged),
			"merged timetable double-books a teacher when seeded on %s", day.Weekday())

		for _, demo := range commuteDemoSeeds() {
			offset := nextWeekdayOffset(day, demo.weekday)
			landsOn := day.AddDate(0, 0, offset).Weekday()
			assert.Equalf(t, demo.weekday, landsOn,
				"demo class for %s resolved to %s when seeded on %s",
				demo.student, landsOn, day.Weekday())

			assert.Truef(t, slices.ContainsFunc(merged, func(s bookingSeed) bool {
				return s.teacher == demo.teacher && s.dayOffset == offset && s.hour == demo.hour
			}), "demo class for teacher %d is missing when seeded on %s", demo.teacher, day.Weekday())
		}
	}
}

// A demo class must always be in the future, never today, so it cannot land in
// an hour that has already passed.
func TestNextWeekdayOffsetAlwaysMovesForward(t *testing.T) {
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	for i := range 7 {
		day := base.AddDate(0, 0, i)
		for wd := time.Sunday; wd <= time.Saturday; wd++ {
			offset := nextWeekdayOffset(day, wd)
			assert.GreaterOrEqual(t, offset, 1)
			assert.LessOrEqual(t, offset, 7)
			assert.Equal(t, wd, day.AddDate(0, 0, offset).Weekday())
		}
	}
}
