package service

import (
	"context"
	"fmt"
	"log"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/ports"
)

type bookingEngine interface {
	FindAlternativesForSlot(ctx context.Context, req models.BookingRequest, window models.WeeklySlot) ([]models.BookingAlternative, error)
	HasResourceChecker() bool
	HasCommuteCalc() bool
}

type BookingService struct {
	repo   ports.BookingRepository
	engine bookingEngine
}

func NewBookingService(repo ports.BookingRepository, engine bookingEngine) *BookingService {
	return &BookingService{repo: repo, engine: engine}
}

func (s *BookingService) Evaluate(ctx context.Context, req models.BookingRequest) (*models.BookingResponse, error) {
	if req.SubjectID <= 0 {
		return nil, &ValidationError{Msg: "subject_id must be positive"}
	}
	if req.BranchID <= 0 {
		return nil, &ValidationError{Msg: "branch_id must be positive"}
	}
	if len(req.PreferredSlots) == 0 {
		return nil, &ValidationError{Msg: "preferred_slots must not be empty"}
	}
	for _, slot := range req.PreferredSlots {
		if slot.DayOfWeek < 0 || slot.DayOfWeek > 6 {
			return nil, &ValidationError{Msg: fmt.Sprintf("day_of_week must be between 0 and 6, got %d", slot.DayOfWeek)}
		}
		if slot.Start == "" || slot.End == "" {
			return nil, &ValidationError{Msg: "slot start and end must not be empty"}
		}
		if slot.Start >= slot.End {
			return nil, &ValidationError{Msg: fmt.Sprintf("slot start %s must be before end %s", string(slot.Start), string(slot.End))}
		}
	}

	var teacherIDVal interface{}
	if req.PreferredTeacherID != nil && *req.PreferredTeacherID > 0 {
		teacherIDVal = *req.PreferredTeacherID
	}

	var results []models.SlotResult

	for _, slot := range req.PreferredSlots {
		match, err := s.repo.FindExactMatch(ctx, req.SubjectID, req.BranchID, slot, req.DurationMinutes, teacherIDVal)
		if err != nil {
			log.Printf("BookingService.Evaluate: FindExactMatch error: %v", err)
			return nil, err
		}

		if match != nil {
			results = append(results, models.SlotResult{
				Slot: slot,
				ExactMatch: &models.BookingAlternative{
					TeacherID:   match.Booking.TeacherID,
					TeacherName: match.TeacherName,
					BranchID:    match.Booking.BranchID,
					SubjectID:   match.Booking.SubjectID,
					StartTime:   match.Booking.StartTime,
					EndTime:     match.Booking.EndTime,
					Score:       100,
					Reasons:     []string{"Exact match"},
				},
				Message: "Exact match found",
			})
			continue
		}

		alternatives, err := s.engine.FindAlternativesForSlot(ctx, req, slot)
		if err != nil {
			log.Printf("BookingService.Evaluate: FindAlternatives error: %v", err)
			return nil, err
		}

		results = append(results, models.SlotResult{
			Slot:         slot,
			Alternatives: alternatives,
			Message:      s.buildMessage(alternatives),
		})
	}

	return &models.BookingResponse{Results: results}, nil
}

func (s *BookingService) buildMessage(alternatives []models.BookingAlternative) string {
	if len(alternatives) == 0 {
		return "No exact match found. No alternatives available."
	}

	checked := s.checkedResources()
	if checked == "" {
		return fmt.Sprintf("No exact match found. %d alternative(s) returned. Room availability not checked.", len(alternatives))
	}
	return fmt.Sprintf("No exact match found. %d alternative(s) returned. %s", len(alternatives), checked)
}

func (s *BookingService) checkedResources() string {
	var parts []string
	if s.engine.HasResourceChecker() {
		parts = append(parts, "room checked")
	}
	if s.engine.HasCommuteCalc() {
		parts = append(parts, "commute calculated")
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("%s.", joinParts(parts))
}

func joinParts(parts []string) string {
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += " and " + parts[i]
	}
	return result
}
