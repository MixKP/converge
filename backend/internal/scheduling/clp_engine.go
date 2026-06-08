package scheduling

import (
	"context"
	"log/slog"
	"strconv"
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

	duration := time.Duration(req.DurationMinutes) * time.Minute
	if duration == 0 {
		winStart := shared.ParseTimeHHMM(window.Start)
		winEnd := shared.ParseTimeHHMM(window.End)
		duration = winEnd.Sub(winStart)
	}

	loc := shared.LoadLocation()
	anchor := shared.AnchorDateForDay(window.DayOfWeek, loc)
	prefStart := anchor.Add(
		time.Duration(shared.ParseTimeHHMM(window.Start).Hour())*time.Hour +
			time.Duration(shared.ParseTimeHHMM(window.Start).Minute())*time.Minute,
	)
	offsets := generateOffsets(prefStart, duration)

	teacherDomain := make([]any, len(teachers))
	for i, t := range teachers {
		teacherDomain[i] = t
	}
	offsetDomain := make([]any, len(offsets))
	for i, o := range offsets {
		offsetDomain[i] = o
	}

	model := NewModel()
	model.AddVariable("teacher", teacherDomain)
	model.AddVariable("offset", offsetDomain)

	model.AddConstraint(func(a Assignment) (bool, error) {
		t := a.Value("teacher").(TeacherInfo)
		o := a.Value("offset").(timeWindow)

		slots, err := e.teacherRoster.TeacherAvailability(ctx, t.ID)
		if err != nil {
			return false, err
		}
		a.Values["slots"] = slots
		if !fitsAvailability(slots, o.start, o.end) {
			return false, nil
		}

		conflicts, err := e.bookingStore.FindConflictingBookings(ctx, t.ID, o.start, o.end)
		if err != nil {
			return false, err
		}
		return len(conflicts) == 0, nil
	})

	model.SetObjective(func(a Assignment) (int, []string, error) {
		t := a.Value("teacher").(TeacherInfo)
		o := a.Value("offset").(timeWindow)
		slots := a.Values["slots"].([]shared.WeeklySlot)
		result := e.scorer.Score(ctx, ScorableCandidate{
			Teacher:           t,
			StartTime:         o.start,
			EndTime:           o.end,
			Request:           req,
			AvailabilitySlots: slots,
		})
		return result.Score, result.Reasons, nil
	})

	model.SetDedupKey(func(a Assignment) string {
		t := a.Value("teacher").(TeacherInfo)
		return strconv.Itoa(t.ID)
	})

	solver := e.newSolver()
	assignments, err := solver.Solve(*model, MaxAlternatives)
	if err != nil {
		return nil, err
	}

	alts := make([]BookingAlternative, 0, len(assignments))
	for _, a := range assignments {
		t := a.Value("teacher").(TeacherInfo)
		o := a.Value("offset").(timeWindow)
		alt := BookingAlternative{
			TeacherID:   t.ID,
			TeacherName: t.Name,
			BranchID:    req.BranchID,
			SubjectID:   req.SubjectID,
			StartTime:   o.start,
			EndTime:     o.end,
			Score:       a.Score,
			Reasons:     a.Reasons,
		}
		alt = e.enrichWithCommute(ctx, alt, req.BranchID, o.start)
		alt = e.enrichWithRoom(ctx, alt, req.BranchID, o.start, o.end)
		alts = append(alts, alt)
	}
	return alts, nil
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

func fitsAvailability(slots []shared.WeeklySlot, start, end time.Time) bool {
	candStart := timeOfDay(start)
	candEnd := timeOfDay(end)
	for _, slot := range slots {
		slotWeekday := time.Weekday(slot.DayOfWeek) // 0=Sunday ... 6=Saturday (matches frontend)
		if start.Weekday() != slotWeekday {
			continue
		}
		availStart := shared.ParseTimeHHMM(slot.Start)
		availEnd := shared.ParseTimeHHMM(slot.End)
		if (candStart.Equal(availStart) || candStart.After(availStart)) &&
			(candEnd.Equal(availEnd) || candEnd.Before(availEnd)) {
			return true
		}
	}
	return false
}

type timeWindow struct {
	start time.Time
	end   time.Time
}

func generateOffsets(anchor time.Time, duration time.Duration) []timeWindow {
	lookbehind := 2 * time.Hour
	lookahead := 2 * time.Hour
	step := 30 * time.Minute

	var windows []timeWindow
	for t := anchor.Add(-lookbehind); !t.After(anchor.Add(lookahead)); t = t.Add(step) {
		windows = append(windows, timeWindow{start: t, end: t.Add(duration)})
	}
	return windows
}
