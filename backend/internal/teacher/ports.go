package teacher

import (
	"context"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type Store interface {
	AddTeacher(ctx context.Context, name, email string) (*Teacher, error)
	SetStatus(ctx context.Context, teacherID int, status string) error
	GetActiveTeachers(ctx context.Context) ([]Teacher, error)
	GetTeachersBySubject(ctx context.Context, subjectID int) ([]Teacher, error)
	GetAllAvailability(ctx context.Context) ([]TeacherAvailability, error)
	GetBranches(ctx context.Context) ([]shared.Branch, error)
	GetSubjects(ctx context.Context) ([]shared.Subject, error)
	FindTeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error)
	ReplaceWeeklyAvailability(ctx context.Context, teacherID int, slots []shared.WeeklySlot) error
	SaveRawSubmission(ctx context.Context, teacherID int, rawPayload []byte) error
}
