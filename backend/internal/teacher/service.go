package teacher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type Service struct {
	store  Store
	logger *slog.Logger
}

func NewService(store Store, logger *slog.Logger) *Service {
	return &Service{store: store, logger: logger}
}

func (s *Service) AddTeacher(ctx context.Context, name, email string) (*Teacher, error) {
	return s.store.AddTeacher(ctx, name, email)
}

func (s *Service) SetStatus(ctx context.Context, teacherID int, status string) error {
	return s.store.SetStatus(ctx, teacherID, status)
}

func (s *Service) GetActiveTeachers(ctx context.Context) ([]Teacher, error) {
	return s.store.GetActiveTeachers(ctx)
}

func (s *Service) GetTeachersBySubject(ctx context.Context, subjectID int) ([]Teacher, error) {
	return s.store.GetTeachersBySubject(ctx, subjectID)
}

func (s *Service) GetAllAvailability(ctx context.Context) ([]TeacherAvailability, error) {
	return s.store.GetAllAvailability(ctx)
}

func (s *Service) TeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error) {
	return s.store.FindTeacherAvailability(ctx, teacherID)
}

func (s *Service) GetBranches(ctx context.Context) ([]shared.Branch, error) {
	return s.store.GetBranches(ctx)
}

func (s *Service) GetSubjects(ctx context.Context) ([]shared.Subject, error) {
	return s.store.GetSubjects(ctx)
}

func (s *Service) SubmitWeeklyAvailability(ctx context.Context, teacherID int, slots []shared.WeeklySlot) error {
	if err := validateAvailabilityPayload(teacherID, slots); err != nil {
		return err
	}

	if err := s.store.ReplaceWeeklyAvailability(ctx, teacherID, slots); err != nil {
		return fmt.Errorf("replace availability: %w", err)
	}

	rawJSON, err := json.Marshal(map[string]interface{}{
		"teacher_id": teacherID,
		"weekly":     slots,
	})
	if err != nil {
		s.logger.Warn("failed to marshal audit payload", slog.Int("teacher_id", teacherID), slog.String("error", err.Error()))
		return nil
	}
	if err := s.store.SaveRawSubmission(ctx, teacherID, rawJSON); err != nil {
		s.logger.Warn("failed to save submission audit log", slog.Int("teacher_id", teacherID), slog.String("error", err.Error()))
	}
	return nil
}

type ValidationError = shared.ValidationError

func validateAvailabilityPayload(teacherID int, slots []shared.WeeklySlot) error {
	if teacherID <= 0 {
		return &ValidationError{Msg: "teacher_id must be positive"}
	}
	if len(slots) == 0 {
		return &ValidationError{Msg: "weekly slots must not be empty"}
	}
	for _, slot := range slots {
		if slot.DayOfWeek < 0 || slot.DayOfWeek > 6 {
			return &ValidationError{Msg: fmt.Sprintf("day_of_week must be between 0 and 6, got %d", slot.DayOfWeek)}
		}
		if slot.Start == "" {
			return &ValidationError{Msg: "start time must not be empty"}
		}
		if slot.End == "" {
			return &ValidationError{Msg: "end time must not be empty"}
		}
		if slot.Start >= slot.End {
			return &ValidationError{Msg: fmt.Sprintf("start time %s must be before end time %s", string(slot.Start), string(slot.End))}
		}
	}
	return validateWeeklySlots(slots)
}

func validateWeeklySlots(slots []shared.WeeklySlot) error {
	byDay := make(map[int][]shared.WeeklySlot)
	for _, s := range slots {
		byDay[s.DayOfWeek] = append(byDay[s.DayOfWeek], s)
	}
	for day, daySlots := range byDay {
		sort.Slice(daySlots, func(i, j int) bool { return daySlots[i].Start < daySlots[j].Start })
		for i := 1; i < len(daySlots); i++ {
			if daySlots[i].Start < daySlots[i-1].End {
				return &ValidationError{
					Msg: "overlapping slots on day " + strconv.Itoa(day) + ": " +
						string(daySlots[i-1].Start) + "-" + string(daySlots[i-1].End) + " and " +
						string(daySlots[i].Start) + "-" + string(daySlots[i].End),
				}
			}
		}
	}
	return nil
}
