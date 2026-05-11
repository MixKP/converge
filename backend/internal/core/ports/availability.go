package ports

import (
	"context"

	"github.com/RinZ5/converge/backend/internal/core/models"
)

type AvailabilityRepository interface {
	GetActiveTeachers(ctx context.Context) ([]models.Teacher, error)
	ReplaceWeeklyAvailability(ctx context.Context, teacherID int, slots []models.WeeklySlot) error
	SaveRawSubmission(ctx context.Context, teacherID int, rawPayload []byte) error
}
