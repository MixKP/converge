package ports

import (
	"context"
	"time"

	"github.com/RinZ5/converge/backend/internal/core/models"
)

type BookingMatch struct {
	Booking     models.Booking
	TeacherName string
}

type BookingRepository interface {
	FindExactMatch(ctx context.Context, req models.BookingRequest) (*BookingMatch, error)
	FindTeachersBySubject(ctx context.Context, subjectID int) ([]models.Teacher, error)
	FindConflictingBookings(ctx context.Context, teacherID int, startTime, endTime time.Time) ([]models.Booking, error)
	FindTeacherAvailability(ctx context.Context, teacherID int) ([]models.WeeklySlot, error)
}

type ScorableCandidate struct {
	Teacher           models.Teacher
	StartTime         time.Time
	EndTime           time.Time
	Request           models.BookingRequest
	AvailabilitySlots []models.WeeklySlot
}

type ScoreResult struct {
	Score   int
	Reasons []string
}

type Scorer interface {
	Score(ctx context.Context, candidate ScorableCandidate) ScoreResult
}

type ResourceChecker interface {
	CheckAvailability(ctx context.Context, branchID int, startTime, endTime time.Time) (bool, error)
}

type CommuteCalculator interface {
	Estimate(ctx context.Context, fromBranchID, toBranchID int, arrivalTime time.Time) (time.Duration, error)
}
