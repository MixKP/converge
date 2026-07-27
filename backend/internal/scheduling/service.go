package scheduling

import (
	"context"
	"errors"
	"fmt"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type SchedulingService struct {
	bookingStore BookingStore
	refStore     ReferenceStore
	engine       bookingEngine
}

type bookingEngine interface {
	FindAlternativesForSlot(ctx context.Context, req BookingRequest, window shared.WeeklySlot) ([]BookingAlternative, error)
}

func NewSchedulingService(bookingStore BookingStore, refStore ReferenceStore, engine bookingEngine) *SchedulingService {
	return &SchedulingService{
		bookingStore: bookingStore,
		refStore:     refStore,
		engine:       engine,
	}
}

func (s *SchedulingService) Evaluate(ctx context.Context, req BookingRequest) (*BookingResponse, error) {
	if err := shared.ValidateAll(req,
		shared.PositiveInt("subject_id", func(r BookingRequest) int { return r.SubjectID }),
		shared.PositiveInt("branch_id", func(r BookingRequest) int { return r.BranchID }),
		shared.SliceNonEmpty("preferred_slots", func(r BookingRequest) int { return len(r.PreferredSlots) }),
	); err != nil {
		return nil, err
	}
	switch req.RequiredGender {
	case "male", "female", "lgbtq+":
	default:
		return nil, &ValidationError{Msg: fmt.Sprintf("required_gender must be one of male, female, lgbtq+, got %q", req.RequiredGender)}
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

	var results []SlotResult

	for _, slot := range req.PreferredSlots {
		match, err := s.bookingStore.FindExactMatch(ctx, req.SubjectID, req.BranchID, slot, req.DurationMinutes, req.PreferredTeacherID, req.RequiredGender)
		if err != nil {
			return nil, fmt.Errorf("find exact match: %w", err)
		}

		if match != nil {
			results = append(results, SlotResult{
				Slot: slot,
				ExactMatch: &BookingAlternative{
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
			return nil, fmt.Errorf("find alternatives for slot: %w", err)
		}

		msg := s.buildAlternativesMessage(alternatives)
		results = append(results, SlotResult{
			Slot:         slot,
			Alternatives: alternatives,
			Message:      msg,
		})
	}

	return &BookingResponse{Results: results}, nil
}

func (s *SchedulingService) buildAlternativesMessage(alternatives []BookingAlternative) string {
	if len(alternatives) == 0 {
		return "No exact match found. No alternatives available."
	}
	return fmt.Sprintf("No exact match found. %d alternative(s) returned.", len(alternatives))
}

func (s *SchedulingService) Confirm(ctx context.Context, req ConfirmBookingRequest) (*Booking, error) {
	if err := shared.ValidateAll(req,
		shared.PositiveInt("teacher_id", func(r ConfirmBookingRequest) int { return r.TeacherID }),
		shared.PositiveInt("branch_id", func(r ConfirmBookingRequest) int { return r.BranchID }),
		shared.PositiveInt("subject_id", func(r ConfirmBookingRequest) int { return r.SubjectID }),
		shared.NonEmpty("client_name", func(r ConfirmBookingRequest) string { return r.ClientName }),
	); err != nil {
		return nil, err
	}
	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		return nil, &ValidationError{Msg: "start_time and end_time must be provided"}
	}
	if !req.EndTime.After(req.StartTime) {
		return nil, &ValidationError{Msg: "end_time must be after start_time"}
	}

	booking, err := s.bookingStore.CreateBooking(ctx, req)
	if err != nil {
		if errors.Is(err, ErrBookingConflict) {
			return nil, &ConflictError{Msg: "Teacher already has a booking in this time range"}
		}
		return nil, err
	}
	return booking, nil
}

func (s *SchedulingService) Cancel(ctx context.Context, bookingID int) error {
	if err := shared.ValidateAll(bookingID,
		shared.PositiveInt("booking_id", func(id int) int { return id }),
	); err != nil {
		return err
	}
	err := s.bookingStore.DeleteBooking(ctx, bookingID)
	if err != nil {
		var notFound *NotFoundError
		if errors.As(err, &notFound) {
			return notFound
		}
		return fmt.Errorf("delete booking: %w", err)
	}
	return nil
}

func (s *SchedulingService) ListAll(ctx context.Context) ([]Booking, error) {
	bookings, err := s.bookingStore.FindAllBookings(ctx)
	if err != nil {
		return nil, fmt.Errorf("find all bookings: %w", err)
	}
	return bookings, nil
}

func (s *SchedulingService) GetBranches(ctx context.Context) ([]Branch, error) {
	return s.refStore.GetBranches(ctx)
}

func (s *SchedulingService) GetSubjects(ctx context.Context) ([]Subject, error) {
	return s.refStore.GetSubjects(ctx)
}
