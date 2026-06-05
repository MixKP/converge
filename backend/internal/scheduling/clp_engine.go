package scheduling

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type CLPEngine struct {
	bookingStore  BookingStore
	teacherRoster TeacherRoster
	scorer        Scorer
	commute       CommuteEstimate
	room          RoomCheck
	logger        *slog.Logger
}

func NewCLPEngine(bookingStore BookingStore, teacherRoster TeacherRoster, scorer Scorer, commute CommuteEstimate, room RoomCheck, logger *slog.Logger) *CLPEngine {
	return &CLPEngine{
		bookingStore:  bookingStore,
		teacherRoster: teacherRoster,
		scorer:        scorer,
		commute:       commute,
		room:          room,
		logger:        logger,
	}
}

func (e *CLPEngine) FindAlternativesForSlot(ctx context.Context, req BookingRequest, window shared.WeeklySlot) ([]BookingAlternative, error) {
	teachers, err := e.teacherRoster.TeachersBySubject(ctx, req.SubjectID)
	if err != nil {
		return nil, err
	}

	e.logger.Debug("finding alternatives",
		"request_id", shared.RequestIDFromContext(ctx),
		"op", "CLPEngine.FindAlternativesForSlot",
		"subject_id", req.SubjectID,
		"branch_id", req.BranchID,
		"teacher_count", len(teachers),
	)

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

			alt = e.enrichWithCommute(ctx, alt, req.BranchID, slot.startTime)
			alt = e.enrichWithRoom(ctx, alt, req.BranchID, slot.startTime, slot.endTime)

			windowCandidates = append(windowCandidates, alt)
		}
	}

	return rankAndLimit(windowCandidates), nil
}

func (e *CLPEngine) enrichWithCommute(ctx context.Context, alt BookingAlternative, branchID int, t time.Time) BookingAlternative {
	if e.commute == nil {
		return alt
	}
	commuteDur, err := e.commute.Estimate(ctx, branchID, branchID, t)
	if err != nil {
		e.logger.Warn("commune estimate failed",
			"request_id", shared.RequestIDFromContext(ctx),
			"op", "CLPEngine.enrichWithCommute",
			"branch_id", branchID,
			"error", err,
		)
		return alt
	}
	if commuteDur > 0 {
		mins := int(commuteDur.Minutes())
		alt.CommuteMinutes = &mins
	}
	return alt
}

func (e *CLPEngine) enrichWithRoom(ctx context.Context, alt BookingAlternative, branchID int, start, end time.Time) BookingAlternative {
	if e.room == nil {
		return alt
	}
	available, err := e.room.CheckAvailability(ctx, branchID, start, end)
	if err != nil {
		e.logger.Warn("room check failed",
			"request_id", shared.RequestIDFromContext(ctx),
			"op", "CLPEngine.enrichWithRoom",
			"branch_id", branchID,
			"error", err,
		)
		return alt
	}
	alt.RoomAvailable = &available
	return alt
}

type candidateSlot struct {
	startTime time.Time
	endTime   time.Time
}

func (e *CLPEngine) generateCandidateSlots(req BookingRequest, availSlots []shared.WeeklySlot, window shared.WeeklySlot) []candidateSlot {
	duration := time.Duration(req.DurationMinutes) * time.Minute
	if duration == 0 {
		winStart := shared.ParseTimeHHMM(window.Start)
		winEnd := shared.ParseTimeHHMM(window.End)
		duration = winEnd.Sub(winStart)
	}
	var candidates []candidateSlot

	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		e.logger.Warn("failed to load timezone, falling back to UTC",
			"op", "CLPEngine.generateCandidateSlots",
			"error", err,
		)
		loc = time.UTC
	}
	anchorDate := shared.AnchorDateForDay(window.DayOfWeek, loc)

	for _, weekSlot := range availSlots {
		if weekSlot.DayOfWeek != window.DayOfWeek {
			continue
		}

		availStart := anchorDate.Add(time.Duration(shared.ParseTimeHHMM(weekSlot.Start).Hour())*time.Hour + time.Duration(shared.ParseTimeHHMM(weekSlot.Start).Minute())*time.Minute)
		availEnd := anchorDate.Add(time.Duration(shared.ParseTimeHHMM(weekSlot.End).Hour())*time.Hour + time.Duration(shared.ParseTimeHHMM(weekSlot.End).Minute())*time.Minute)

		if availEnd.Sub(availStart) < duration {
			continue
		}

		windowStart := anchorDate.Add(time.Duration(shared.ParseTimeHHMM(window.Start).Hour())*time.Hour + time.Duration(shared.ParseTimeHHMM(window.Start).Minute())*time.Minute)

		for t := windowStart.Add(-CandidateLookbehind); t.Before(windowStart.Add(CandidateLookahead)) || t.Equal(windowStart.Add(CandidateLookahead)); t = t.Add(CandidateStepDuration) {
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

func rankAndLimit(candidates []BookingAlternative) []BookingAlternative {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	candidates = deduplicateByTeacher(candidates)
	if len(candidates) > MaxAlternatives {
		candidates = candidates[:MaxAlternatives]
	}
	return candidates
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
