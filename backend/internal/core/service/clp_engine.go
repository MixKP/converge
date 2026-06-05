package service

import (
	"context"
	"log"
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

func (e *CLPEngine) FindAlternativesForSlot(ctx context.Context, req models.BookingRequest, window models.WeeklySlot) ([]models.BookingAlternative, error) {
	teachers, err := e.repo.FindTeachersBySubject(ctx, req.SubjectID)
	if err != nil {
		return nil, err
	}

	var windowCandidates []models.BookingAlternative

	for _, teacher := range teachers {
		availSlots, err := e.repo.FindTeacherAvailability(ctx, teacher.ID)
		if err != nil {
			return nil, err
		}

		candidateSlots := e.generateCandidateSlots(req, availSlots, window)
		for _, slot := range candidateSlots {
			hasConf, err := e.hasConflict(ctx, teacher.ID, slot.startTime, slot.endTime)
			if err != nil {
				return nil, err
			}
			if hasConf {
				continue
			}

			result := e.scorer.Score(ctx, ports.ScorableCandidate{
				Teacher:           teacher,
				StartTime:         slot.startTime,
				EndTime:           slot.endTime,
				Request:           req,
				AvailabilitySlots: availSlots,
				MatchedSlot:       window,
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

			windowCandidates = append(windowCandidates, alt)
		}
	}

	sort.Slice(windowCandidates, func(i, j int) bool {
		return windowCandidates[i].Score > windowCandidates[j].Score
	})
	windowCandidates = deduplicateByTeacher(windowCandidates)
	if len(windowCandidates) > 3 {
		windowCandidates = windowCandidates[:3]
	}

	return windowCandidates, nil
}

type candidateSlot struct {
	startTime time.Time
	endTime   time.Time
}

func (e *CLPEngine) generateCandidateSlots(req models.BookingRequest, availSlots []models.WeeklySlot, window models.WeeklySlot) []candidateSlot {
	duration := time.Duration(req.DurationMinutes) * time.Minute
	if duration == 0 {
		winStart := parseTimeHHMM(window.Start)
		winEnd := parseTimeHHMM(window.End)
		duration = winEnd.Sub(winStart)
	}
	var candidates []candidateSlot

	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		log.Printf("Warning: failed to load Asia/Bangkok timezone, falling back to UTC: %v", err)
		loc = time.UTC
	}
	now := time.Now().In(loc)
	anchorDate := now.Truncate(24 * time.Hour)
	desiredWeekday := time.Weekday(window.DayOfWeek)
	for anchorDate.Weekday() != desiredWeekday {
		anchorDate = anchorDate.Add(24 * time.Hour)
	}

	for _, weekSlot := range availSlots {
		if weekSlot.DayOfWeek != window.DayOfWeek {
			continue
		}

		availStart := anchorDate.Add(time.Duration(parseTimeHHMMInLocation(weekSlot.Start, loc).Hour())*time.Hour + time.Duration(parseTimeHHMMInLocation(weekSlot.Start, loc).Minute())*time.Minute)
		availEnd := anchorDate.Add(time.Duration(parseTimeHHMMInLocation(weekSlot.End, loc).Hour())*time.Hour + time.Duration(parseTimeHHMMInLocation(weekSlot.End, loc).Minute())*time.Minute)

		if availEnd.Sub(availStart) < duration {
			continue
		}

		windowStart := anchorDate.Add(time.Duration(parseTimeHHMMInLocation(window.Start, loc).Hour())*time.Hour + time.Duration(parseTimeHHMMInLocation(window.Start, loc).Minute())*time.Minute)

		step := 30 * time.Minute
		genStart := windowStart.Add(-2 * time.Hour)
		genEnd := windowStart.Add(2 * time.Hour)
		for t := genStart; t.Before(genEnd) || t.Equal(genEnd); t = t.Add(step) {
			if t.Before(availStart) || t.After(availEnd) {
				continue
			}
			if t.Add(duration).After(availEnd) {
				continue
			}
			candidates = append(candidates, candidateSlot{startTime: t, endTime: t.Add(duration)})
		}
	}

	return candidates
}

func (e *CLPEngine) hasConflict(ctx context.Context, teacherID int, startTime, endTime time.Time) (bool, error) {
	conflicts, err := e.repo.FindConflictingBookings(ctx, teacherID, startTime, endTime)
	if err != nil {
		return false, err
	}
	return len(conflicts) > 0, nil
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
