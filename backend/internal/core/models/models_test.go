package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeacherJSON(t *testing.T) {
	tc := Teacher{ID: 1, Name: "Alice"}
	b, err := json.Marshal(tc)
	require.NoError(t, err)

	var decoded Teacher
	require.NoError(t, json.Unmarshal(b, &decoded))
	assert.Equal(t, tc, decoded)
}

func TestSubjectJSON(t *testing.T) {
	s := Subject{ID: 10, Name: "Mathematics"}
	b, err := json.Marshal(s)
	require.NoError(t, err)

	var decoded Subject
	require.NoError(t, json.Unmarshal(b, &decoded))
	assert.Equal(t, s, decoded)
}

func TestBranchJSON(t *testing.T) {
	b := Branch{ID: 5, Name: "Downtown Campus"}
	raw, err := json.Marshal(b)
	require.NoError(t, err)

	var decoded Branch
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, b, decoded)
}

func TestAvailabilityPayloadJSON(t *testing.T) {
	p := AvailabilityPayload{
		TeacherID: 42,
		Weekly: []WeeklySlot{
			{DayOfWeek: 1, Start: "09:00", End: "10:00"},
			{DayOfWeek: 3, Start: "14:00", End: "16:00"},
		},
	}
	b, err := json.Marshal(p)
	require.NoError(t, err)

	var decoded AvailabilityPayload
	require.NoError(t, json.Unmarshal(b, &decoded))
	assert.Equal(t, p, decoded)
}

func TestBookingJSON(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	later := now.Add(90 * time.Minute)
	bk := Booking{
		ID:        100,
		TeacherID: 42,
		BranchID:  5,
		SubjectID: 10,
		StartTime: now,
		EndTime:   later,
		CreatedAt: now,
	}
	raw, err := json.Marshal(bk)
	require.NoError(t, err)

	var decoded Booking
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, bk.ID, decoded.ID)
	assert.Equal(t, bk.TeacherID, decoded.TeacherID)
	assert.Equal(t, bk.BranchID, decoded.BranchID)
	assert.Equal(t, bk.SubjectID, decoded.SubjectID)
	assert.True(t, bk.StartTime.Equal(decoded.StartTime))
	assert.True(t, bk.EndTime.Equal(decoded.EndTime))
}

func TestBookingRequestOmitEmpty(t *testing.T) {
	withoutTeacher := BookingRequest{
		SubjectID:       10,
		BranchID:        5,
		PreferredStart:  time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		DurationMinutes: 90,
	}
	b, err := json.Marshal(withoutTeacher)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "preferred_teacher_id")

	id := 42
	withTeacher := BookingRequest{
		SubjectID:          10,
		BranchID:           5,
		PreferredStart:     time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		DurationMinutes:    90,
		PreferredTeacherID: &id,
	}
	b2, err := json.Marshal(withTeacher)
	require.NoError(t, err)
	assert.Contains(t, string(b2), "preferred_teacher_id")
}
