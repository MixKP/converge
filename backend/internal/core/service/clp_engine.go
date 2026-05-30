package service

import (
	"context"
	"sort"
	"time"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/ports"
)

type CLPEngine struct {
	repo            ports.BookingRepository
	scorer          ports.Scorer
	resourceChecker ports.ResourceChecker
	commuteCalc     ports.CommuteCalculator
}

func NewCLPEngine(repo ports.BookingRepository, scorer ports.Scorer, rc ports.ResourceChecker, cc ports.CommuteCalculator) *CLPEngine {
	return &CLPEngine{
		repo:            repo,
		scorer:          scorer,
		resourceChecker: rc,
		commuteCalc:     cc,
	}
}

func (e *CLPEngine) FindAlternatives(ctx context.Context, req models.BookingRequest) ([]models.BookingAlternative, error) {
	teachers, err := e.repo.FindTeachersBySubject(ctx, req.SubjectID)
	if err != nil {
		return nil, err
	}

	var alternatives []models.BookingAlternative

	for _, teacher := range teachers {
		availSlots, err := e.repo.FindTeacherAvailability(ctx, teacher.ID)
		if err != nil {
			return nil, err
		}

		candidateSlots := e.generateCandidateSlots(req, availSlots)
		for _, slot := range candidateSlots {
			if e.hasConflict(ctx, teacher.ID, slot.startTime, slot.endTime) {
				continue
			}

			result := e.scorer.Score(ctx, ports.ScorableCandidate{
				Teacher:           teacher,
				StartTime:         slot.startTime,
				EndTime:           slot.endTime,
				Request:           req,
				AvailabilitySlots: availSlots,
			})

			alt := models.BookingAlternative{
				TeacherID:   teacher.ID,
				TeacherName: teacher.Name,
				BranchID:    req.BranchID,
				SubjectID:   req.SubjectID,
				StartTime:   slot.startTime,
				EndTime:     slot.endTime,
				Score:       result.Score,
				Reasons:     result.Reasons,
			}

			alternatives = append(alternatives, alt)
		}
	}

	sort.Slice(alternatives, func(i, j int) bool {
		return alternatives[i].Score > alternatives[j].Score
	})

	alternatives = deduplicateByTeacher(alternatives)

	if len(alternatives) > 3 {
		alternatives = alternatives[:3]
	}

	return alternatives, nil
}

type candidateSlot struct {
	startTime time.Time
	endTime   time.Time
}

func (e *CLPEngine) generateCandidateSlots(req models.BookingRequest, availSlots []models.WeeklySlot) []candidateSlot {
	duration := time.Duration(req.DurationMinutes) * time.Minute
	var candidates []candidateSlot

	for _, weekSlot := range availSlots {
		availStart, availEnd := e.resolveTimes(req, weekSlot)
		slotDuration := availEnd.Sub(availStart)

		if slotDuration < duration {
			continue
		}

		if req.PreferredStart.IsZero() {
			for t := availStart; t.Add(duration).Before(availEnd) || t.Add(duration).Equal(availEnd); t = t.Add(30 * time.Minute) {
				candidates = append(candidates, candidateSlot{startTime: t, endTime: t.Add(duration)})
			}
		} else {
			step := 30 * time.Minute
			windowStart := req.PreferredStart.Add(-2 * time.Hour)
			windowEnd := req.PreferredStart.Add(2 * time.Hour)
			for t := windowStart; t.Before(windowEnd) || t.Equal(windowEnd); t = t.Add(step) {
				if t.Before(availStart) || t.After(availEnd) {
					continue
				}
				if t.Add(duration).After(availEnd) {
					continue
				}
				candidates = append(candidates, candidateSlot{startTime: t, endTime: t.Add(duration)})
			}
		}
	}

	return candidates
}

func (e *CLPEngine) resolveTimes(req models.BookingRequest, weekSlot models.WeeklySlot) (time.Time, time.Time) {
	refTime := req.PreferredStart
	if refTime.IsZero() {
		refTime = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	}

	dayOfWeek := int(refTime.Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 7 // use 1-7 for matching
	}
	dayOfWeek-- // convert to 0=Monday

	if weekSlot.DayOfWeek != dayOfWeek {
		return time.Time{}, time.Time{}
	}

	parsedStart, _ := time.Parse("15:04", string(weekSlot.Start))
	parsedEnd, _ := time.Parse("15:04", string(weekSlot.End))

	date := refTime.Truncate(24 * time.Hour)
	availStart := date.Add(time.Duration(parsedStart.Hour())*time.Hour + time.Duration(parsedStart.Minute())*time.Minute)
	availEnd := date.Add(time.Duration(parsedEnd.Hour())*time.Hour + time.Duration(parsedEnd.Minute())*time.Minute)

	return availStart, availEnd
}

func (e *CLPEngine) hasConflict(ctx context.Context, teacherID int, startTime, endTime time.Time) bool {
	conflicts, err := e.repo.FindConflictingBookings(ctx, teacherID, startTime, endTime)
	if err != nil {
		return true
	}
	return len(conflicts) > 0
}

func deduplicateByTeacher(alternatives []models.BookingAlternative) []models.BookingAlternative {
	seen := make(map[int]bool)
	var result []models.BookingAlternative
	for _, a := range alternatives {
		if seen[a.TeacherID] {
			continue
		}
		seen[a.TeacherID] = true
		result = append(result, a)
	}
	return result
}

func (e *CLPEngine) HasResourceChecker() bool { return e.resourceChecker != nil }

func (e *CLPEngine) HasCommuteCalc() bool { return e.commuteCalc != nil }
