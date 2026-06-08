package scheduling

import (
	"context"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
)

func TestScorerExactTeacherMatchReturnsFullPoints(t *testing.T) {
	scorer := NewWeightedScorer()
	preferredID := 1
	teacher := TeacherInfo{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, window(0, "13:00", "14:00"))

	result := scorer.Score(context.Background(), candidate)

	assert.Equal(t, 100, result.Score)
	assert.Contains(t, result.Reasons[0], "Your preferred teacher")
	assert.Contains(t, result.Reasons[1], "Plenty of availability")
}

func TestScorerNoTeacherPreferenceReturnsNeutral(t *testing.T) {
	scorer := NewWeightedScorer()
	teacher := TeacherInfo{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, 0, window(0, "13:00", "14:00"))

	result := scorer.Score(context.Background(), candidate)

	assert.Equal(t, 75, result.Score)
}

func TestScorerWrongTeacherReturnsZeroTeacherScore(t *testing.T) {
	scorer := NewWeightedScorer()
	preferredID := 42
	teacher := TeacherInfo{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, window(0, "13:00", "14:00"))

	result := scorer.Score(context.Background(), candidate)

	assert.Contains(t, result.Reasons[0], "Can also teach this subject")
}

func TestScorerConfigurableWeights(t *testing.T) {
	scorer := &WeightedScorer{TeacherWeight: 50, FitWeight: 50}
	preferredID := 1
	teacher := TeacherInfo{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, preferredID, window(0, "13:00", "14:00"))

	result := scorer.Score(context.Background(), candidate)

	assert.Equal(t, 100, result.Score)
}

func TestScorerLowBufferScore(t *testing.T) {
	scorer := NewWeightedScorer()
	teacher := TeacherInfo{ID: 1, Name: "Alice"}
	candidate := makeCandidate(teacher, 0, window(0, "13:00", "14:00"))
	candidate.AvailabilitySlots = []shared.WeeklySlot{
		{DayOfWeek: 0, Start: shared.TimeHHMM("13:30"), End: shared.TimeHHMM("14:30")},
	}

	result := scorer.Score(context.Background(), candidate)

	assert.Less(t, result.Score, 75)
	assert.Contains(t, result.Reasons[1], "Tightly fits schedule")
}

func window(day int, start, end string) shared.WeeklySlot {
	return shared.WeeklySlot{DayOfWeek: day, Start: shared.TimeHHMM(start), End: shared.TimeHHMM(end)}
}

func makeCandidate(teacher TeacherInfo, preferredTeacherID int, prefSlot shared.WeeklySlot) ScorableCandidate {
	req := BookingRequest{
		SubjectID:      1,
		BranchID:       1,
		PreferredSlots: []shared.WeeklySlot{prefSlot},
	}
	if preferredTeacherID > 0 {
		req.PreferredTeacherID = &preferredTeacherID
	}

	anchorDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	desiredWeekday := time.Weekday(prefSlot.DayOfWeek) // 0=Sunday ... 6=Saturday
	for anchorDate.Weekday() != desiredWeekday {
		anchorDate = anchorDate.Add(24 * time.Hour)
	}
	parsedStart, _ := time.Parse("15:04", string(prefSlot.Start))
	parsedEnd, _ := time.Parse("15:04", string(prefSlot.End))
	startTime := anchorDate.Add(time.Duration(parsedStart.Hour())*time.Hour + time.Duration(parsedStart.Minute())*time.Minute)
	endTime := anchorDate.Add(time.Duration(parsedEnd.Hour())*time.Hour + time.Duration(parsedEnd.Minute())*time.Minute)

	return ScorableCandidate{
		Teacher:           teacher,
		StartTime:         startTime,
		EndTime:           endTime,
		Request:           req,
		AvailabilitySlots: []shared.WeeklySlot{{DayOfWeek: prefSlot.DayOfWeek, Start: "09:00", End: "17:00"}},
	}
}
