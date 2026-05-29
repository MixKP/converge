package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type TimeHHMM string

func (t *TimeHHMM) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.Parse("15:04", s)
	if err != nil {
		return fmt.Errorf("invalid time format %q, expected HH:MM", s)
	}
	*t = TimeHHMM(parsed.Format("15:04"))
	return nil
}

func (t TimeHHMM) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(t))
}

type Teacher struct {
	ID    int    `json:"id"    example:"1"`
	Name  string `json:"name"  example:"Alice"`
	Email string `json:"email" example:"alice@example.com"`
}

type WeeklySlot struct {
	DayOfWeek int      `json:"day_of_week" example:"0"` // 0=Monday ... 6=Sunday
	Start     TimeHHMM `json:"start"       example:"09:00"`
	End       TimeHHMM `json:"end"         example:"10:00"`
}

type AvailabilityPayload struct {
	TeacherID int          `json:"teacher_id" example:"42"`
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

type MessageResponse struct {
	Message string `json:"message" example:"Availability saved successfully"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"failed to retrieve teachers"`
}
