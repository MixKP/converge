package scheduling

import (
	"context"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
)

// ---- Driven ports (what Scheduling needs from outside) ----

type BookingStore interface {
	FindExactMatch(ctx context.Context, subjectID, branchID int, slot shared.WeeklySlot, durationMinutes int, teacherID shared.Option[int], gender string) (*BookingMatch, error)
	FindConflictingBookings(ctx context.Context, teacherID int, startTime, endTime time.Time) ([]Booking, error)
	CreateBooking(ctx context.Context, req ConfirmBookingRequest) (*Booking, error)
	DeleteBooking(ctx context.Context, bookingID int) error
	FindAllBookings(ctx context.Context) ([]Booking, error)
}

type TeacherRoster interface {
	TeachersBySubject(ctx context.Context, subjectID int) ([]TeacherInfo, error)
	TeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error)
}

type ReferenceStore interface {
	GetSubjects(ctx context.Context) ([]Subject, error)
}

type CommuteEstimate interface {
	Estimate(ctx context.Context, fromBranchID, toBranchID int, arrivalTime time.Time) (time.Duration, error)
}

type BranchCapacityCheck interface {
	CheckCapacity(ctx context.Context, branchID int, startTime, endTime time.Time) (bool, error)
}

// ---- Domain port ----

type Scorer interface {
	Score(ctx context.Context, candidate ScorableCandidate) ScoreResult
}
