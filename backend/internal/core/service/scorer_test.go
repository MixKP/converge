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
	preferredStart := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, &preferredStart)
	candidate.StartTime = time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	candidate.EndTime = time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)

	result := scorer.Score(context.Background(), candidate)

	assert.Equal(t, 100, result.Score)
	assert.Contains(t, result.Reasons, "Preferred teacher")
	assert.Contains(t, result.Reasons, "Exact time match")
	assert.Contains(t, result.Reasons, "Plenty of buffer")
}

func TestWeightedScorer_NoTeacherPreference_ReturnsNeutralTeacherScore(t *testing.T) {
	scorer := NewWeightedScorer()
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, 0, nil)
	candidate.StartTime = time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	candidate.EndTime = time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)

	result := scorer.Score(context.Background(), candidate)

	assert.Equal(t, 65, result.Score)
	assert.Equal(t, 20, scoreTeacherOnly(scorer, candidate))
}

func TestWeightedScorer_WrongTeacher_ReturnsZeroTeacherScore(t *testing.T) {
	scorer := NewWeightedScorer()
	preferredID := 42
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, nil)

	result := scorer.Score(context.Background(), candidate)

	assert.Contains(t, result.Reasons, "Different teacher")
	assert.Equal(t, 0, scoreTeacherOnly(scorer, candidate))
}

func TestWeightedScorer_ExactTimeMatch_ReturnsFullTimeScore(t *testing.T) {
	scorer := NewWeightedScorer()
	preferredID := 1
	preferredStart := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, &preferredStart)
	candidate.StartTime = time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	candidate.EndTime = time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)

	result := scorer.Score(context.Background(), candidate)

	assert.Contains(t, result.Reasons, "Exact time match")
	assert.Equal(t, 30, scoreTimeOnly(scorer, candidate))
}

func TestWeightedScorer_TimeOffBy30min_ReturnsPartialTimeScore(t *testing.T) {
	scorer := NewWeightedScorer()
	preferredID := 1
	preferredStart := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, &preferredStart)
	candidate.StartTime = time.Date(2026, 6, 1, 13, 30, 0, 0, time.UTC)
	candidate.EndTime = time.Date(2026, 6, 1, 14, 30, 0, 0, time.UTC)

	result := scorer.Score(context.Background(), candidate)

	assert.Equal(t, 25, scoreTimeOnly(scorer, candidate))
	assert.Contains(t, result.Reasons, "Within 1hr — off by 30m")
}

func TestWeightedScorer_TimeOffBy2hrs_ReturnsLowTimeScore(t *testing.T) {
	scorer := NewWeightedScorer()
	preferredID := 1
	preferredStart := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, &preferredStart)
	candidate.StartTime = time.Date(2026, 6, 1, 15, 0, 0, 0, time.UTC)
	candidate.EndTime = time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC)

	result := scorer.Score(context.Background(), candidate)

	assert.Equal(t, 15, scoreTimeOnly(scorer, candidate))
	assert.Contains(t, result.Reasons, "Within 2hrs — off by 120m")
}

func TestWeightedScorer_NoPreferredTime_ReturnsNeutralTimeScore(t *testing.T) {
	scorer := NewWeightedScorer()
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, 0, nil)
	candidate.StartTime = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	candidate.EndTime = time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)

	_ = scorer.Score(context.Background(), candidate)

	assert.Equal(t, 15, scoreTimeOnly(scorer, candidate))
}

func TestWeightedScorer_WideAvailabilityFit_HighFitScore(t *testing.T) {
	scorer := NewWeightedScorer()
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, 0, nil)
	candidate.StartTime = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	candidate.EndTime = time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)

	result := scorer.Score(context.Background(), candidate)

	assert.Contains(t, result.Reasons, "Plenty of buffer")
	assert.Equal(t, 30, scoreFitOnly(scorer, candidate))
}

func TestWeightedScorer_TightAvailabilityFit_LowerFitScore(t *testing.T) {
	scorer := NewWeightedScorer()
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	preferredID := 1
	candidate := makeCandidate(teacher, preferredID, nil)
	candidate.AvailabilitySlots = []models.WeeklySlot{
		{DayOfWeek: 1, Start: "09:00", End: "10:30"},
	}
	candidate.StartTime = time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	candidate.EndTime = time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

	result := scorer.Score(context.Background(), candidate)

	assert.Contains(t, result.Reasons, "Very tight fit")
	assert.True(t, scoreFitOnly(scorer, candidate) < 10)
}

func TestWeightedScorer_ConfigurableWeights(t *testing.T) {
	scorer := &WeightedScorer{TeacherWeight: 50, TimeWeight: 50, FitWeight: 0}
	preferredID := 1
	preferredStart := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	teacher := models.Teacher{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, &preferredStart)
	candidate.StartTime = time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	candidate.EndTime = time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)

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

func makeCandidate(teacher models.Teacher, preferredTeacherID int, preferredStart *time.Time) ports.ScorableCandidate {
	req := models.BookingRequest{
		SubjectID: 1,
		BranchID:  1,
	}
	if preferredTeacherID > 0 {
		req.PreferredTeacherID = &preferredTeacherID
	}
	if preferredStart != nil {
		req.PreferredStart = *preferredStart
	}
	return ports.ScorableCandidate{
		Teacher:           teacher,
		Request:           req,
		AvailabilitySlots: []models.WeeklySlot{{DayOfWeek: 1, Start: "09:00", End: "17:00"}},
	}
}
