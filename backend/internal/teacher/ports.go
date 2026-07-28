package teacher

import (
	"context"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type TeacherStore interface {
	AddTeacher(ctx context.Context, name, email, gender string) (*Teacher, error)
	SetStatus(ctx context.Context, teacherID int, status string) error
	SetGender(ctx context.Context, teacherID int, gender string) error
	GetActiveTeachers(ctx context.Context) ([]Teacher, error)
	GetTeachersBySubject(ctx context.Context, subjectID int) ([]Teacher, error)
}

type AvailabilityStore interface {
	GetAllAvailability(ctx context.Context) ([]TeacherAvailability, error)
	FindTeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error)
	ReplaceWeeklyAvailability(ctx context.Context, teacherID int, slots []shared.WeeklySlot) error
	SaveRawSubmission(ctx context.Context, teacherID int, rawPayload []byte) error
}

type ReferenceStore interface {
	GetSubjects(ctx context.Context) ([]shared.Subject, error)
}
