package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/ports"
)

type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string {
	return e.Msg
}

type AvailabilityService struct {
	repo ports.AvailabilityRepository
}

func NewAvailabilityService(repo ports.AvailabilityRepository) *AvailabilityService {
	return &AvailabilityService{repo: repo}
}

func (s *AvailabilityService) GetActiveTeachers(ctx context.Context) ([]models.Teacher, error) {
	return s.repo.GetActiveTeachers(ctx)
}

func (s *AvailabilityService) GetTeachersBySubject(ctx context.Context, subjectID int) ([]models.Teacher, error) {
	return s.repo.GetTeachersBySubject(ctx, subjectID)
}

func (s *AvailabilityService) GetBranches(ctx context.Context) ([]models.Branch, error) {
	return s.repo.GetBranches(ctx)
}

func (s *AvailabilityService) GetSubjects(ctx context.Context) ([]models.Subject, error) {
	return s.repo.GetSubjects(ctx)
}

func (s *AvailabilityService) GetAllAvailability(ctx context.Context) ([]models.TeacherAvailability, error) {
	return s.repo.GetAllAvailability(ctx)
}

func (s *AvailabilityService) SubmitWeeklyAvailability(ctx context.Context, payload models.AvailabilityPayload) error {
	if err := validatePayload(payload); err != nil {
		return err
	}
	rawJSON, _ := json.Marshal(payload)
	if err := s.repo.ReplaceWeeklyAvailability(ctx, payload.TeacherID, payload.Weekly); err != nil {
		return err
	}
	if err := s.repo.SaveRawSubmission(ctx, payload.TeacherID, rawJSON); err != nil {
		log.Printf("Warning: failed to save submission audit log: %v", err)
	}
	return nil
}

func validatePayload(payload models.AvailabilityPayload) error {
	if payload.TeacherID <= 0 {
		return &ValidationError{Msg: "teacher_id must be positive"}
	}
	if len(payload.Weekly) == 0 {
		return &ValidationError{Msg: "weekly slots must not be empty"}
	}
	for _, slot := range payload.Weekly {
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
	return validateWeeklySlots(payload.Weekly)
}

func validateWeeklySlots(slots []models.WeeklySlot) error {
	byDay := make(map[int][]models.WeeklySlot)
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
