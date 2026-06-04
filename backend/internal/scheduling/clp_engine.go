package scheduling

import (
	"context"
	"log"
	"sort"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type CLPEngine struct {
	bookingStore BookingStore
	teacherRoster TeacherRoster
	scorer       Scorer
	commute      CommuteEstimate
	room         RoomCheck
}

func NewCLPEngine(bookingStore BookingStore, teacherRoster TeacherRoster, scorer Scorer, commute CommuteEstimate, room RoomCheck) *CLPEngine {
	return &CLPEngine{
		bookingStore:  bookingStore,
		teacherRoster: teacherRoster,
		scorer:        scorer,
		commute:       commute,
		room:          room,
	}
}

func (e *CLPEngine) FindAlternativesForSlot(ctx context.Context, req BookingRequest, window shared.WeeklySlot) ([]BookingAlternative, error) {
	teachers, err := e.teacherRoster.TeachersBySubject(ctx, req.SubjectID)
	if err != nil {
		return nil, err
	}

	var windowCandidates []BookingAlternative

	for _, teacher := range teachers {
		availSlots, err := e.teacherRoster.TeacherAvailability(ctx, teacher.ID)
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

			result := e.scorer.Score(ctx, ScorableCandidate{
				Teacher:           teacher,
				StartTime:         slot.startTime,
				EndTime:           slot.endTime,
				Request:           req,
				AvailabilitySlots: availSlots,
				MatchedSlot:       window,
			})

			alt := BookingAlternative{
				TeacherID:   teacher.ID,
				TeacherName: teacher.Name,
				BranchID:    req.BranchID,
				SubjectID:   req.SubjectID,
				StartTime:   slot.startTime,
				EndTime:     slot.endTime,
				Score:       result.Score,
				Reasons:     result.Reasons,
			}

			if e.commute != nil {
				commuteDur, err := e.commute.Estimate(ctx, req.BranchID, req.BranchID, slot.startTime)
				if err == nil && commuteDur > 0 {
					mins := int(commuteDur.Minutes())
					alt.CommuteMinutes = &mins
				}
			}

			if e.room != nil {
				available, err := e.room.CheckAvailability(ctx, req.BranchID, slot.startTime, slot.endTime)
				if err == nil {
					alt.RoomAvailable = &available
				}
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

func (e *CLPEngine) generateCandidateSlots(req BookingRequest, availSlots []shared.WeeklySlot, window shared.WeeklySlot) []candidateSlot {
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
	anchorDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
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
	conflicts, err := e.bookingStore.FindConflictingBookings(ctx, teacherID, startTime, endTime)
	if err != nil {
		return false, err
	}
	return len(conflicts) > 0, nil
}

func deduplicateByTeacher(alternatives []BookingAlternative) []BookingAlternative {
	seen := make(map[int]bool)
	var result []BookingAlternative
	for _, a := range alternatives {
		if seen[a.TeacherID] {
			continue
		}
		seen[a.TeacherID] = true
		result = append(result, a)
	}
	return result
}
