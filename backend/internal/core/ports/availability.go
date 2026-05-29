package ports

import (
	"context"

	"github.com/RinZ5/converge/backend/internal/core/models"
)

type AvailabilityRepository interface {
	GetActiveTeachers(ctx context.Context) ([]models.Teacher, error)
	GetTeachersBySubject(ctx context.Context, subjectID int) ([]models.Teacher, error)
	GetAllAvailability(ctx context.Context) ([]models.TeacherAvailability, error)
	GetBranches(ctx context.Context) ([]models.Branch, error)
	GetSubjects(ctx context.Context) ([]models.Subject, error)
	ReplaceWeeklyAvailability(ctx context.Context, teacherID int, slots []models.WeeklySlot) error
	SaveRawSubmission(ctx context.Context, teacherID int, rawPayload []byte) error
}
