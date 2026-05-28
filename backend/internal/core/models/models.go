package models

import (
	"time"
)

type Teacher struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type WeeklySlot struct {
	DayOfWeek int    `json:"day_of_week"`
	Start     string `json:"start"`
	End       string `json:"end"`
}

type AvailabilityPayload struct {
	TeacherID int          `json:"teacher_id"`
	Weekly    []WeeklySlot `json:"weekly"`
}

type Subject struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Branch struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Booking struct {
	ID        int       `json:"id"`
	TeacherID int       `json:"teacher_id"`
	BranchID  int       `json:"branch_id"`
	SubjectID int       `json:"subject_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	CreatedAt time.Time `json:"created_at"`
}

type BookingRequest struct {
	SubjectID          int       `json:"subject_id"`
	BranchID           int       `json:"branch_id"`
	PreferredStart     time.Time `json:"preferred_start"`
	DurationMinutes    int       `json:"duration_minutes"`
	PreferredTeacherID *int      `json:"preferred_teacher_id,omitempty"`
}
