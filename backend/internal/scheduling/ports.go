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
	FindBookingsByBranch(ctx context.Context, branchID int, startTime, endTime time.Time) ([]Booking, error)
	CreateBooking(ctx context.Context, req ConfirmBookingRequest) (*Booking, error)
	DeleteBooking(ctx context.Context, bookingID int) error
	FindAllBookings(ctx context.Context) ([]Booking, error)
	FindBookingsByStudentIDs(ctx context.Context, studentIDs []int) ([]Booking, error)
}

type TeacherRoster interface {
	TeachersBySubject(ctx context.Context, subjectID int) ([]TeacherInfo, error)
	TeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error)
}

type ReferenceStore interface {
	GetSubjects(ctx context.Context) ([]Subject, error)
}

type CommuteProvider interface {
	DefaultCommute(ctx context.Context) (time.Duration, error)
}

type BranchCapacityCheck interface {
	GetCapacity(ctx context.Context, branchID int) (int, error)
}

// ---- Domain port ----

type Scorer interface {
	Score(ctx context.Context, candidate ScorableCandidate) ScoreResult
}
