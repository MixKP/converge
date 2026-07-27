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
	teacherStore      TeacherStore
	availabilityStore AvailabilityStore
	referenceStore    ReferenceStore
	logger            *slog.Logger
}

func NewService(teacherStore TeacherStore, availabilityStore AvailabilityStore, referenceStore ReferenceStore, logger *slog.Logger) *Service {
	return &Service{
		teacherStore:      teacherStore,
		availabilityStore: availabilityStore,
		referenceStore:    referenceStore,
		logger:            logger,
	}
}

func (s *Service) AddTeacher(ctx context.Context, name, email, gender string) (*Teacher, error) {
	return s.teacherStore.AddTeacher(ctx, name, email, gender)
}

func (s *Service) SetStatus(ctx context.Context, teacherID int, status string) error {
	return s.teacherStore.SetStatus(ctx, teacherID, status)
}

func (s *Service) SetGender(ctx context.Context, teacherID int, gender string) error {
	return s.teacherStore.SetGender(ctx, teacherID, gender)
}

func (s *Service) GetActiveTeachers(ctx context.Context) ([]Teacher, error) {
	return s.teacherStore.GetActiveTeachers(ctx)
}

func (s *Service) GetTeachersBySubject(ctx context.Context, subjectID int) ([]Teacher, error) {
	return s.teacherStore.GetTeachersBySubject(ctx, subjectID)
}

func (s *Service) GetAllAvailability(ctx context.Context) ([]TeacherAvailability, error) {
	return s.availabilityStore.GetAllAvailability(ctx)
}

func (s *Service) TeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error) {
	return s.availabilityStore.FindTeacherAvailability(ctx, teacherID)
}

func (s *Service) GetBranches(ctx context.Context) ([]shared.Branch, error) {
	return s.referenceStore.GetBranches(ctx)
}

func (s *Service) GetSubjects(ctx context.Context) ([]shared.Subject, error) {
	return s.referenceStore.GetSubjects(ctx)
}

func (s *Service) SubmitWeeklyAvailability(ctx context.Context, teacherID int, slots []shared.WeeklySlot) error {
	if err := validateAvailabilityPayload(teacherID, slots); err != nil {
		return err
	}

	if err := s.availabilityStore.ReplaceWeeklyAvailability(ctx, teacherID, slots); err != nil {
		return fmt.Errorf("replace availability: %w", err)
	}

	rawJSON, err := json.Marshal(map[string]interface{}{
		"teacher_id": teacherID,
		"weekly":     slots,
	})
	if err != nil {
		s.logger.Warn("failed to marshal audit payload", slog.Int("teacher_id", teacherID), slog.Any("error", err))
		return nil
	}
	if err := s.availabilityStore.SaveRawSubmission(ctx, teacherID, rawJSON); err != nil {
		s.logger.Warn("failed to save submission audit log", slog.Int("teacher_id", teacherID), slog.Any("error", err))
	}
	return nil
}

type ValidationError = shared.ValidationError

func validateAvailabilityPayload(teacherID int, slots []shared.WeeklySlot) error {
	if err := shared.ValidateAll(teacherID,
		shared.PositiveInt("teacher_id", func(id int) int { return id }),
	); err != nil {
		return err
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
