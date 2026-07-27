package adapter

import (
	"context"

	"github.com/RinZ5/converge/backend/internal/scheduling"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/RinZ5/converge/backend/internal/teacher"
)

type TeacherRosterSource interface {
	GetTeachersBySubject(ctx context.Context, subjectID int) ([]teacher.Teacher, error)
	TeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error)
}

type TeacherRosterAdapter struct {
	src TeacherRosterSource
}

func NewTeacherRosterAdapter(src TeacherRosterSource) *TeacherRosterAdapter {
	return &TeacherRosterAdapter{src: src}
}

func (a *TeacherRosterAdapter) TeachersBySubject(ctx context.Context, subjectID int) ([]scheduling.TeacherInfo, error) {
	teachers, err := a.src.GetTeachersBySubject(ctx, subjectID)
	if err != nil {
		return nil, err
	}
	info := make([]scheduling.TeacherInfo, len(teachers))
	for i, t := range teachers {
		info[i] = scheduling.TeacherInfo{ID: t.ID, Name: t.Name, Gender: t.Gender}
	}
	return info, nil
}

func (a *TeacherRosterAdapter) TeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error) {
	return a.src.TeacherAvailability(ctx, teacherID)
}
