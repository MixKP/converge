package service

import (
	"context"
	"encoding/json"
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

func (s *AvailabilityService) SubmitWeeklyAvailability(ctx context.Context, payload models.AvailabilityPayload) error {
	if err := validateWeeklySlots(payload.Weekly); err != nil {
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
						daySlots[i-1].Start + "-" + daySlots[i-1].End + " and " +
						daySlots[i].Start + "-" + daySlots[i].End,
				}
			}
		}
	}
	return nil
}
