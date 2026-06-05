package scheduling

import (
	"context"
	"log/slog"
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

	solver := e.newSolver()
	candidates, err := solver.Solve(ctx, teachers, req, window,
		func(teacher TeacherInfo, start, end time.Time, availSlots []shared.WeeklySlot, req BookingRequest) (int, []string) {
			result := e.scorer.Score(ctx, ScorableCandidate{
				Teacher:           teacher,
				StartTime:         start,
				EndTime:           end,
				Request:           req,
				AvailabilitySlots: availSlots,
			})
			return result.Score, result.Reasons
		},
		e.teacherRoster,
		e.bookingStore,
	)
	if err != nil {
		return nil, err
	}

	alts := make([]BookingAlternative, len(candidates))
	for i, c := range candidates {
		alt := BookingAlternative{
			TeacherID:   c.Teacher.ID,
			TeacherName: c.Teacher.Name,
			BranchID:    req.BranchID,
			SubjectID:   req.SubjectID,
			StartTime:   c.Start,
			EndTime:     c.End,
			Score:       c.Score,
			Reasons:     c.Reasons,
		}
		alt = e.enrichWithCommute(ctx, alt, req.BranchID, c.Start)
		alt = e.enrichWithRoom(ctx, alt, req.BranchID, c.Start, c.End)
		alts[i] = alt
	}
	return alts, nil
}

func (e *CLPEngine) checkAvailability(ctx context.Context, teacher TeacherInfo, start, end time.Time) (bool, error) {
	slots, err := e.teacherRoster.TeacherAvailability(ctx, teacher.ID)
	if err != nil {
		return false, err
	}
	for _, slot := range slots {
		slotWeekday := time.Weekday((slot.DayOfWeek + 1) % 7)
		if start.Weekday() != slotWeekday {
			continue
		}
		availStart := shared.ParseTimeHHMM(slot.Start)
		availEnd := shared.ParseTimeHHMM(slot.End)
		candStart := timeOfDay(start)
		candEnd := timeOfDay(end)
		if (candStart.Equal(availStart) || candStart.After(availStart)) &&
			(candEnd.Equal(availEnd) || candEnd.Before(availEnd)) {
			return true, nil
		}
	}
	return false, nil
}

func (e *CLPEngine) checkConflict(ctx context.Context, teacher TeacherInfo, start, end time.Time) (bool, error) {
	conflicts, err := e.bookingStore.FindConflictingBookings(ctx, teacher.ID, start, end)
	if err != nil {
		return false, err
	}
	return len(conflicts) == 0, nil
}

func (e *CLPEngine) enrichWithCommute(ctx context.Context, alt BookingAlternative, branchID int, t time.Time) BookingAlternative {
	if e.commute == nil {
		return alt
	}
	commuteDur, err := e.commute.Estimate(ctx, branchID, branchID, t)
	if err != nil {
		e.logger.Warn("commute estimate failed",
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
