package db

import (
	"testing"

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
