package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/ports"
)

type WeightedScorer struct {
	TeacherWeight int
	TimeWeight    int
	FitWeight     int
}

func NewWeightedScorer() *WeightedScorer {
	return &WeightedScorer{TeacherWeight: 40, TimeWeight: 30, FitWeight: 30}
}

func (s *WeightedScorer) Score(ctx context.Context, candidate ports.ScorableCandidate) ports.ScoreResult {
	var reasons []string

	teacherScore, teacherReason := s.scoreTeacherPreference(candidate)
	reasons = appendReason(reasons, teacherReason)

	timeScore, timeReason := s.scoreTimeProximity(candidate)
	reasons = appendReason(reasons, timeReason)

	fitScore, fitReason := s.scoreAvailabilityFit(candidate)
	reasons = appendReason(reasons, fitReason)

	total := teacherScore + timeScore + fitScore

	return ports.ScoreResult{
		Score:   total,
		Reasons: reasons,
	}
}

func (s *WeightedScorer) scoreTeacherPreference(candidate ports.ScorableCandidate) (int, string) {
	if candidate.Request.PreferredTeacherID == nil {
		return 20, ""
	}
	if *candidate.Request.PreferredTeacherID == candidate.Teacher.ID {
		return s.TeacherWeight, "Preferred teacher"
	}
	return 0, "Different teacher"
}

func (s *WeightedScorer) scoreTimeProximity(candidate ports.ScorableCandidate) (int, string) {
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
	distanceMinutes := math.Min(distStart, distEnd)

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

func (s *WeightedScorer) scoreAvailabilityFit(candidate ports.ScorableCandidate) (int, string) {
	if len(candidate.AvailabilitySlots) == 0 {
		return s.FitWeight / 2, ""
	}

	proposedStart := timeOfDay(candidate.StartTime)
	proposedEnd := timeOfDay(candidate.EndTime)

	for _, slot := range candidate.AvailabilitySlots {
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

func parseTimeHHMM(t models.TimeHHMM) time.Time {
	parsed, _ := time.Parse("15:04", string(t))
	return time.Date(0, 1, 1, parsed.Hour(), parsed.Minute(), 0, 0, time.UTC)
}

func appendReason(reasons []string, reason string) []string {
	if reason == "" {
		return reasons
	}
	return append(reasons, reason)
}
