package service

import (
	"context"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/ports"

	"github.com/stretchr/testify/assert"
)

func TestWeightedScorer_ExactTeacherMatch_ReturnsFullPoints(t *testing.T) {
	scorer := NewWeightedScorer()
	preferredID := 1
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, window(0, "13:00", "14:00"))
	candidate.MatchedSlot = window(0, "13:00", "14:00")

	result := scorer.Score(context.Background(), candidate)

	assert.Equal(t, 100, result.Score)
	assert.Contains(t, result.Reasons, "Preferred teacher")
	assert.Contains(t, result.Reasons, "Inside preferred window")
	assert.Contains(t, result.Reasons, "Plenty of buffer")
}

func TestWeightedScorer_NoTeacherPreference_ReturnsNeutralTeacherScore(t *testing.T) {
	scorer := NewWeightedScorer()
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, 0, window(0, "13:00", "14:00"))
	candidate.MatchedSlot = window(0, "13:00", "14:00")

	result := scorer.Score(context.Background(), candidate)

	assert.Equal(t, 80, result.Score)
	assert.Equal(t, 20, scoreTeacherOnly(scorer, candidate))
}

func TestWeightedScorer_WrongTeacher_ReturnsZeroTeacherScore(t *testing.T) {
	scorer := NewWeightedScorer()
	preferredID := 42
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, window(0, "13:00", "14:00"))

	result := scorer.Score(context.Background(), candidate)

	assert.Contains(t, result.Reasons, "Different teacher")
	assert.Equal(t, 0, scoreTeacherOnly(scorer, candidate))
}

func TestWeightedScorer_InsidePreferredWindow_ReturnsFullTimeScore(t *testing.T) {
	scorer := NewWeightedScorer()
	preferredID := 1
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, window(0, "13:00", "14:00"))
	candidate.MatchedSlot = window(0, "13:00", "14:00")

	result := scorer.Score(context.Background(), candidate)

	assert.Contains(t, result.Reasons, "Inside preferred window")
	assert.Equal(t, 30, scoreTimeOnly(scorer, candidate))
}

func TestWeightedScorer_NearWindow_ReturnsNearScore(t *testing.T) {
	scorer := NewWeightedScorer()
	preferredID := 1
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, window(0, "13:00", "14:00"))
	candidate.MatchedSlot = window(0, "13:00", "14:00")
	candidate.StartTime = candidate.StartTime.Add(10 * time.Minute)
	candidate.EndTime = candidate.EndTime.Add(10 * time.Minute)

	result := scorer.Score(context.Background(), candidate)

	assert.Contains(t, result.Reasons, "Near window — off by 10m")
	assert.Equal(t, 25, scoreTimeOnly(scorer, candidate))
}

func TestWeightedScorer_FarFromWindow_ReturnsFarScore(t *testing.T) {
	scorer := NewWeightedScorer()
	preferredID := 1
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, window(0, "13:00", "14:00"))
	candidate.MatchedSlot = window(0, "13:00", "14:00")
	candidate.StartTime = candidate.StartTime.Add(5 * time.Hour)
	candidate.EndTime = candidate.EndTime.Add(5 * time.Hour)

	scorer.Score(context.Background(), candidate)

	assert.Equal(t, 0, scoreTimeOnly(scorer, candidate))
}

func TestWeightedScorer_NoMatchedWindow_ReturnsNeutralTimeScore(t *testing.T) {
	scorer := NewWeightedScorer()
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, 0, window(0, "13:00", "14:00"))

	assert.Equal(t, 15, scoreTimeOnly(scorer, candidate))
}

func TestWeightedScorer_WideAvailabilityFit_HighFitScore(t *testing.T) {
	scorer := NewWeightedScorer()
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, 0, window(0, "12:00", "13:00"))

	result := scorer.Score(context.Background(), candidate)

	assert.Contains(t, result.Reasons, "Plenty of buffer")
	assert.Equal(t, 30, scoreFitOnly(scorer, candidate))
}

func TestWeightedScorer_TightAvailabilityFit_LowerFitScore(t *testing.T) {
	scorer := NewWeightedScorer()
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	preferredID := 1
	candidate := makeCandidate(teacher, preferredID, window(1, "09:00", "10:00"))
	candidate.AvailabilitySlots = []models.WeeklySlot{
		{DayOfWeek: 1, Start: "09:00", End: "10:30"},
	}
	candidate.StartTime = time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)
	candidate.EndTime = time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)

	result := scorer.Score(context.Background(), candidate)

	assert.Contains(t, result.Reasons, "Very tight fit")
	assert.True(t, scoreFitOnly(scorer, candidate) < 10)
}

func TestWeightedScorer_ConfigurableWeights(t *testing.T) {
	scorer := &WeightedScorer{TeacherWeight: 50, TimeWeight: 50, FitWeight: 0}
	preferredID := 1
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, window(0, "13:00", "14:00"))
	candidate.MatchedSlot = window(0, "13:00", "14:00")

	result := scorer.Score(context.Background(), candidate)

	assert.Equal(t, 100, result.Score)
	assert.Equal(t, 0, scoreFitOnly(scorer, candidate))
}

func scoreTeacherOnly(s *WeightedScorer, c ports.ScorableCandidate) int {
	score, _ := s.scoreTeacherPreference(c)
	return score
}

func scoreTimeOnly(s *WeightedScorer, c ports.ScorableCandidate) int {
	score, _ := s.scoreTimeProximity(c)
	return score
}

func scoreFitOnly(s *WeightedScorer, c ports.ScorableCandidate) int {
	score, _ := s.scoreAvailabilityFit(c)
	return score
}

func window(day int, start, end string) models.WeeklySlot {
	return models.WeeklySlot{DayOfWeek: day, Start: models.TimeHHMM(start), End: models.TimeHHMM(end)}
}

func makeCandidate(teacher models.Teacher, preferredTeacherID int, prefSlot models.WeeklySlot) ports.ScorableCandidate {
	req := models.BookingRequest{
		SubjectID:      1,
		BranchID:       1,
		PreferredSlots: []models.WeeklySlot{prefSlot},
	}
	if preferredTeacherID > 0 {
		req.PreferredTeacherID = &preferredTeacherID
	}

	anchorDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	desiredWeekday := time.Weekday((prefSlot.DayOfWeek + 1) % 7)
	for anchorDate.Weekday() != desiredWeekday {
		anchorDate = anchorDate.Add(24 * time.Hour)
	}
	parsedStart, _ := time.Parse("15:04", string(prefSlot.Start))
	parsedEnd, _ := time.Parse("15:04", string(prefSlot.End))
	startTime := anchorDate.Add(time.Duration(parsedStart.Hour())*time.Hour + time.Duration(parsedStart.Minute())*time.Minute)
	endTime := anchorDate.Add(time.Duration(parsedEnd.Hour())*time.Hour + time.Duration(parsedEnd.Minute())*time.Minute)

	return ports.ScorableCandidate{
		Teacher:           teacher,
		StartTime:         startTime,
		EndTime:           endTime,
		Request:           req,
		AvailabilitySlots: []models.WeeklySlot{{DayOfWeek: prefSlot.DayOfWeek, Start: "09:00", End: "17:00"}},
	}
}
