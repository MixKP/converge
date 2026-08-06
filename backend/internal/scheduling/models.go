package scheduling

import (
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type TimeHHMM = shared.TimeHHMM
type WeeklySlot = shared.WeeklySlot
type Subject = shared.Subject

// Booking carries the display names alongside the foreign keys so a client
// listing bookings does not need to resolve them: /branches is admin-only, so
// students and parents cannot look a branch name up themselves. The name
// fields are only populated by the list queries; the overlap/conflict scans
// leave them empty, as they do for StudentName.
type Booking struct {
	ID          int       `json:"id"`
	TeacherID   int       `json:"teacher_id"`
	TeacherName string    `json:"teacher_name,omitempty"`
	BranchID    int       `json:"branch_id"`
	BranchName  string    `json:"branch_name,omitempty"`
	SubjectID   int       `json:"subject_id"`
	SubjectName string    `json:"subject_name,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	StudentID   int       `json:"student_id"`
	StudentName string    `json:"student_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type BookingRequest struct {
	SubjectID          int                 `json:"subject_id"          binding:"required" example:"1"`
	BranchID           int                 `json:"branch_id"           binding:"required" example:"1"`
	PreferredSlots     []shared.WeeklySlot `json:"preferred_slots"     binding:"required"`
	DurationMinutes    int                 `json:"duration_minutes,omitempty" example:"60"`
	PreferredTeacherID shared.Option[int]  `json:"preferred_teacher_id,omitempty" swaggertype:"integer" example:"5"`
	RequiredGender     string              `json:"required_gender" binding:"required,oneof=male female lgbtq+" example:"female"`
}

type ConfirmBookingRequest struct {
	TeacherID int       `json:"teacher_id"  binding:"required" example:"1"`
	BranchID  int       `json:"branch_id"   binding:"required" example:"1"`
	SubjectID int       `json:"subject_id"  binding:"required" example:"1"`
	StartTime time.Time `json:"start_time"  binding:"required" example:"2026-06-01T09:00:00Z"`
	EndTime   time.Time `json:"end_time"    binding:"required" example:"2026-06-01T10:00:00Z"`
	StudentID int       `json:"student_id"  binding:"required" example:"3"`
}

type BookingAlternative struct {
	TeacherID      int                `json:"teacher_id"`
	TeacherName    string             `json:"teacher_name"`
	BranchID       int                `json:"branch_id"`
	SubjectID      int                `json:"subject_id"`
	StartTime      time.Time          `json:"start_time"`
	EndTime        time.Time          `json:"end_time"`
	Score          int                `json:"score"`
	Reasons        []string           `json:"reasons"`
	CommuteMinutes shared.Option[int] `json:"commute_minutes,omitempty" swaggertype:"integer"`
}

type SlotResult struct {
	Slot         shared.WeeklySlot    `json:"slot"`
	ExactMatch   *BookingAlternative  `json:"exact_match,omitempty"`
	Alternatives []BookingAlternative `json:"alternatives,omitempty"`
	Message      string               `json:"message"`
}

type BookingResponse struct {
	Results []SlotResult `json:"results"`
}

type BookingMatch struct {
	Booking     Booking
	TeacherName string
}

type ScorableCandidate struct {
	Teacher           TeacherInfo
	StartTime         time.Time
	EndTime           time.Time
	Request           BookingRequest
	AvailabilitySlots []shared.WeeklySlot
	MatchedSlot       shared.WeeklySlot
}

type ScoreResult struct {
	Score   int
	Reasons []string
}

type TeacherInfo struct {
	ID     int
	Name   string
	Gender string
}

type MessageResponse struct {
	Message string `json:"message" example:"Availability saved successfully"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"failed to retrieve teachers"`
}
