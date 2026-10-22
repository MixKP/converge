package teacher

import (
	"context"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type TeacherStore interface {
	AddTeacher(ctx context.Context, name, email, gender string) (*Teacher, error)
	SetStatus(ctx context.Context, teacherID int, status string) error
	SetGender(ctx context.Context, teacherID int, gender string) error
	GetActiveTeachers(ctx context.Context) ([]Teacher, error)
	GetAllTeachers(ctx context.Context) ([]Teacher, error)
	GetInactiveTeachers(ctx context.Context) ([]Teacher, error)
	GetTeachersBySubject(ctx context.Context, subjectID int) ([]Teacher, error)
	SubjectsForTeacher(ctx context.Context, teacherID int) ([]shared.Subject, error)
	// ReplaceTeacherSubjects is a full replace, not a merge: the checklist the
	// admin submits is the complete set, the same contract as
	// ReplaceWeeklyAvailability.
	ReplaceTeacherSubjects(ctx context.Context, teacherID int, subjectIDs []int) error
}

type AvailabilityStore interface {
	GetAllAvailability(ctx context.Context) ([]TeacherAvailability, error)
	FindTeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error)
	ReplaceWeeklyAvailability(ctx context.Context, teacherID int, slots []shared.WeeklySlot) error
	ReplaceWeeklyAvailabilityAndDeleteBookingsFrom(ctx context.Context, teacherID int, slots []shared.WeeklySlot, from time.Time) (int, error)
	SaveRawSubmission(ctx context.Context, teacherID int, rawPayload []byte) error
}

type ReferenceStore interface {
	GetSubjects(ctx context.Context) ([]shared.Subject, error)
}
