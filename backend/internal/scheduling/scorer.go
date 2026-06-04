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
	TimeWeight    int
	FitWeight     int
}

func NewWeightedScorer() *WeightedScorer {
	return &WeightedScorer{TeacherWeight: 40, TimeWeight: 30, FitWeight: 30}
}

func (s *WeightedScorer) Score(ctx context.Context, candidate ScorableCandidate) ScoreResult {
	var reasons []string

	teacherScore, teacherReason := s.scoreTeacherPreference(candidate)
	reasons = appendReason(reasons, teacherReason)

	timeScore, timeReason := s.scoreTimeProximity(candidate)
	reasons = appendReason(reasons, timeReason)

	fitScore, fitReason := s.scoreAvailabilityFit(candidate)
	reasons = appendReason(reasons, fitReason)

	total := teacherScore + timeScore + fitScore

	return ScoreResult{
		Score:   total,
		Reasons: reasons,
	}
}

func (s *WeightedScorer) scoreTeacherPreference(candidate ScorableCandidate) (int, string) {
	if candidate.Request.PreferredTeacherID == nil {
		return 20, ""
	}
	if *candidate.Request.PreferredTeacherID == candidate.Teacher.ID {
		return s.TeacherWeight, "Preferred teacher"
	}
	return 0, "Different teacher"
}

func (s *WeightedScorer) scoreTimeProximity(candidate ScorableCandidate) (int, string) {
	matched := candidate.MatchedSlot
	if matched.Start == "" || matched.End == "" {
		return 15, ""
	}

	candStart := timeOfDay(candidate.StartTime)
	candEnd := timeOfDay(candidate.EndTime)
	winStart := parseTimeHHMM(matched.Start)
	winEnd := parseTimeHHMM(matched.End)

	if (candStart.Equal(winStart) || candStart.After(winStart)) && (candEnd.Equal(winEnd) || candEnd.Before(winEnd)) {
		return s.TimeWeight, "Inside preferred window"
	}

	distStart := math.Abs(candStart.Sub(winStart).Minutes())
	distEnd := math.Abs(candEnd.Sub(winEnd).Minutes())
	distanceMinutes := math.Max(distStart, distEnd)

	switch {
	case distanceMinutes <= 15:
		return 25, fmt.Sprintf("Near window — off by %dm", int(distanceMinutes))
	case distanceMinutes <= 60:
		return 15, fmt.Sprintf("Within 1hr of window — off by %dm", int(distanceMinutes))
	case distanceMinutes <= 120:
		return 5, fmt.Sprintf("Within 2hrs of window — off by %dm", int(distanceMinutes))
	default:
		return 0, fmt.Sprintf("Far from window — off by %dm", int(distanceMinutes))
	}
}

func (s *WeightedScorer) scoreAvailabilityFit(candidate ScorableCandidate) (int, string) {
	if len(candidate.AvailabilitySlots) == 0 {
		return s.FitWeight / 2, ""
	}

	proposedStart := timeOfDay(candidate.StartTime)
	proposedEnd := timeOfDay(candidate.EndTime)

	for _, slot := range candidate.AvailabilitySlots {
		slotWeekday := time.Weekday((slot.DayOfWeek + 1) % 7)
		if candidate.StartTime.Weekday() != slotWeekday {
			continue
		}

		availStart := parseTimeHHMM(slot.Start)
		availEnd := parseTimeHHMM(slot.End)

		if (proposedStart.Equal(availStart) || proposedStart.After(availStart)) && (proposedEnd.Equal(availEnd) || proposedEnd.Before(availEnd)) {
			marginBefore := float64(proposedStart.Sub(availStart).Minutes())
			marginAfter := float64(availEnd.Sub(proposedEnd).Minutes())
			buffer := math.Min(marginBefore, marginAfter)
			duration := float64(candidate.EndTime.Sub(candidate.StartTime).Minutes())
			ratio := buffer / duration

			switch {
			case ratio >= 2:
				return s.FitWeight, "Plenty of buffer"
			case ratio >= 1:
				return 25, "Good buffer"
			case ratio >= 0.5:
				return 15, "Tight fit"
			default:
				return 5, "Very tight fit"
			}
		}
	}

	return 5, "Very tight fit"
}

func timeOfDay(t time.Time) time.Time {
	return time.Date(0, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
}

func parseTimeHHMM(t shared.TimeHHMM) time.Time {
	parsed, err := time.Parse("15:04", string(t))
	if err != nil {
		return time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(0, 1, 1, parsed.Hour(), parsed.Minute(), 0, 0, time.UTC)
}

func parseTimeHHMMInLocation(t models.TimeHHMM, loc *time.Location) time.Time {
	parsed, err := time.ParseInLocation("15:04", string(t), loc)
	if err != nil {
		return time.Date(0, 1, 1, 0, 0, 0, 0, loc)
	}
	return parsed
}

func appendReason(reasons []string, reason string) []string {
	if reason == "" {
		return reasons
	}
	return append(reasons, reason)
}
