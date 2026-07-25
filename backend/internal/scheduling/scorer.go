package scheduling

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type WeightedScorer struct {
	TeacherWeight int
	FitWeight     int
}

func NewWeightedScorer() *WeightedScorer {
	return &WeightedScorer{TeacherWeight: 50, FitWeight: 50}
}

func (s *WeightedScorer) Score(ctx context.Context, candidate ScorableCandidate) ScoreResult {
	var reasons []string

	teacherScore, teacherReason := s.scoreTeacherPreference(candidate)
	reasons = append(reasons, teacherReason)

	fitScore, fitReason := s.scoreAvailabilityFit(candidate)
	reasons = append(reasons, fitReason)

	total := teacherScore + fitScore

	return ScoreResult{
		Score:   total,
		Reasons: reasons,
	}
}

func (s *WeightedScorer) scoreTeacherPreference(candidate ScorableCandidate) (int, string) {
	prefID, ok := candidate.Request.PreferredTeacherID.Value()
	if !ok {
		return s.TeacherWeight / 2, "Teaches this subject"
	}
	if prefID == candidate.Teacher.ID {
		return s.TeacherWeight, "Your preferred teacher"
	}
	return 0, "Can also teach this subject"
}

func (s *WeightedScorer) scoreAvailabilityFit(candidate ScorableCandidate) (int, string) {
	if candidate.StartTime.IsZero() || candidate.EndTime.IsZero() {
		return s.FitWeight / 2, "Schedule unavailable"
	}

	dayOfWeek := int(candidate.StartTime.Weekday()) // 0=Sunday ... 6=Saturday (matches frontend)
	proposedStart := timeOfDay(candidate.StartTime)
	proposedEnd := timeOfDay(candidate.EndTime)

	slots, err := loadTeacherAvailability(candidate)
	if err != nil || len(slots) == 0 {
		return s.FitWeight / 2, "Schedule unavailable"
	}

	for _, slot := range slots {
		if slot.DayOfWeek != dayOfWeek {
			continue
		}

		availStart := shared.ParseTimeHHMM(slot.Start)
		availEnd := shared.ParseTimeHHMM(slot.End)

		if (proposedStart.Equal(availStart) || proposedStart.After(availStart)) &&
			(proposedEnd.Equal(availEnd) || proposedEnd.Before(availEnd)) {
			marginBefore := float64(proposedStart.Sub(availStart).Minutes())
			marginAfter := float64(availEnd.Sub(proposedEnd).Minutes())
			buffer := math.Min(marginBefore, marginAfter)
			duration := float64(candidate.EndTime.Sub(candidate.StartTime).Minutes())
			ratio := buffer / duration

			switch {
			case ratio >= 2:
				return s.FitWeight, fmt.Sprintf("Plenty of availability (%.0fmin buffer)", buffer)
			case ratio >= 1:
				return 25, fmt.Sprintf("Well within schedule (%.0fmin buffer)", buffer)
			case ratio >= 0.5:
				return 15, fmt.Sprintf("Fits available time (%.0fmin cushion)", buffer)
			case buffer < 1:
				return 5, "Time adjusted to match availability"
			default:
				return 5, fmt.Sprintf("Tightly fits schedule (%.0fmin cushion)", buffer)
			}
		}
	}

	return 5, "Tightly fits schedule"
}

func loadTeacherAvailability(candidate ScorableCandidate) ([]shared.WeeklySlot, error) {
	return candidate.AvailabilitySlots, nil
}

func timeOfDay(t time.Time) time.Time {
	return time.Date(0, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
}
