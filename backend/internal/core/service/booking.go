package service

import (
	"context"
	"fmt"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/ports"
)

type bookingEngine interface {
	FindAlternatives(ctx context.Context, req models.BookingRequest) ([]models.BookingAlternative, error)
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
	if req.DurationMinutes <= 0 {
		return nil, &ValidationError{Msg: "duration_minutes must be positive"}
	}

	match, err := s.repo.FindExactMatch(ctx, req)
	if err != nil {
		return nil, err
	}

	if match != nil {
		alt := models.BookingAlternative{
			TeacherID:   match.Booking.TeacherID,
			TeacherName: match.TeacherName,
			BranchID:    match.Booking.BranchID,
			SubjectID:   match.Booking.SubjectID,
			StartTime:   match.Booking.StartTime,
			EndTime:     match.Booking.EndTime,
			Score:       100,
			Reasons:     []string{"Exact match"},
		}
		return &models.BookingResponse{
			ExactMatch: &alt,
			Message:    "Exact match found",
		}, nil
	}

	alternatives, err := s.engine.FindAlternatives(ctx, req)
	if err != nil {
		return nil, err
	}

	msg := s.buildMessage(alternatives)
	return &models.BookingResponse{
		Alternatives: alternatives,
		Message:      msg,
	}, nil
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
