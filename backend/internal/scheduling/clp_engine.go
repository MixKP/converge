package scheduling

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type CLPEngine struct {
	bookingStore   BookingStore
	teacherRoster  TeacherRoster
	scorer         Scorer
	commute        CommuteProvider
	branchCapacity BranchCapacityCheck
	logger         *slog.Logger
}

func NewCLPEngine(bookingStore BookingStore, teacherRoster TeacherRoster, scorer Scorer, commute CommuteProvider, branchCapacity BranchCapacityCheck, logger *slog.Logger) *CLPEngine {
	if branchCapacity == nil {
		panic("scheduling: NewCLPEngine requires a non-nil BranchCapacityCheck; pass a no-op implementation if capacity should not be enforced")
	}
	return &CLPEngine{
		bookingStore:   bookingStore,
		teacherRoster:  teacherRoster,
		scorer:         scorer,
		commute:        commute,
		branchCapacity: branchCapacity,
		logger:         logger,
	}
}

type prefetchedTeacherData struct {
	slots     []shared.WeeklySlot
	conflicts []Booking
}

func (e *CLPEngine) FindAlternativesForSlot(ctx context.Context, req BookingRequest, window shared.WeeklySlot) ([]BookingAlternative, error) {
	teachers, err := e.teacherRoster.TeachersBySubject(ctx, req.SubjectID)
	if err != nil {
		return nil, err
	}

	filtered := make([]TeacherInfo, 0, len(teachers))
	for _, t := range teachers {
		if t.Gender == req.RequiredGender {
			filtered = append(filtered, t)
		}
	}
	teachers = filtered

	e.logger.Debug("finding alternatives",
		"request_id", shared.RequestIDFromContext(ctx),
		"op", "CLPEngine.FindAlternativesForSlot",
		"subject_id", req.SubjectID,
		"branch_id", req.BranchID,
		"teacher_count", len(teachers),
	)

	if len(teachers) == 0 {
		return nil, nil
	}

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

	// The commute pad is a single global value: a booking at a different
	// branch than the request needs this much travel time, one at the same
	// branch needs none. Fetch it once (I/O hoisted out of the constraint,
	// which must stay pure). A lookup error fails the request rather than
	// silently emitting suggestions with no travel time — the exact case this
	// feature must reject (matches CommuteConflict). No commute source wired
	// ⇒ pad 0, behavior unchanged.
	var commutePad time.Duration
	if e.commute != nil {
		d, err := e.commute.DefaultCommute(ctx)
		if err != nil {
			return nil, err
		}
		commutePad = d
	}

	// Widen the conflict-fetch window by the commute pad so a different-branch
	// booking just outside the candidate range still surfaces and can block an
	// edge candidate once padded (mirrors CommuteConflict's widened lookup).
	prefetched := make(map[int]prefetchedTeacherData, len(teachers))
	conflictStart := prefStart.Add(-CandidateLookbehind).Add(-commutePad)
	conflictEnd := prefStart.Add(CandidateLookahead).Add(duration).Add(commutePad)
	for _, t := range teachers {
		slots, err := e.teacherRoster.TeacherAvailability(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		conflicts, err := e.bookingStore.FindConflictingBookings(ctx, t.ID, conflictStart, conflictEnd)
		if err != nil {
			return nil, err
		}
		prefetched[t.ID] = prefetchedTeacherData{slots: slots, conflicts: conflicts}
	}

	branchCapacity := 0
	branchOverlapByOffset := make(map[timeWindow]int, len(offsets))
	capacity, capErr := e.branchCapacity.GetCapacity(ctx, req.BranchID)
	if capErr != nil {
		e.logger.Warn("branch capacity lookup failed, skipping capacity enforcement",
			"request_id", shared.RequestIDFromContext(ctx),
			"op", "CLPEngine.FindAlternativesForSlot",
			"branch_id", req.BranchID,
			"error", capErr,
		)
	} else {
		branchCapacity = capacity
	}

	if branchCapacity > 0 {
		branchBookings, bookErr := e.bookingStore.FindBookingsByBranch(ctx, req.BranchID, conflictStart, conflictEnd)
		if bookErr != nil {
			e.logger.Warn("branch bookings lookup failed, skipping capacity enforcement",
				"request_id", shared.RequestIDFromContext(ctx),
				"op", "CLPEngine.FindAlternativesForSlot",
				"branch_id", req.BranchID,
				"error", bookErr,
			)
			branchCapacity = 0
		} else {
			for _, o := range offsets {
				count := 0
				for _, b := range branchBookings {
					if overlaps(o.start, o.end, b.StartTime, b.EndTime) {
						count++
					}
				}
				branchOverlapByOffset[o] = count
			}
		}
	}

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
		data := prefetched[t.ID]

		a.Metadata["slots"] = data.slots
		if !fitsAvailability(data.slots, o.start, o.end) {
			return false, nil
		}

		for _, c := range data.conflicts {
			buf := padFor(c.BranchID, req.BranchID, commutePad)
			if overlaps(o.start, o.end, c.StartTime.Add(-buf), c.EndTime.Add(buf)) {
				return false, nil
			}
		}

		if branchCapacity > 0 && branchOverlapByOffset[o] >= branchCapacity {
			return false, nil
		}
		return true, nil
	})

	model.SetObjective(func(a Assignment) (int, []string, error) {
		t := a.Value("teacher").(TeacherInfo)
		o := a.Value("offset").(timeWindow)
		slots := a.Metadata["slots"].([]shared.WeeklySlot)
		result := e.scorer.Score(ctx, ScorableCandidate{
			Teacher:           t,
			StartTime:         o.start,
			EndTime:           o.end,
			Request:           req,
			AvailabilitySlots: slots,
		})
		// Rank primarily by how little the slot shifts from the requested
		// time; the 0-100 quality only breaks ties between equally-close
		// slots. Keep the raw quality in metadata for display.
		shift := o.start.Sub(prefStart)
		if shift < 0 {
			shift = -shift
		}
		steps := int(shift / CandidateStepDuration)
		a.Metadata["quality"] = result.Score
		return result.Score - steps*ProximityWeight, result.Reasons, nil
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
		quality, _ := a.Metadata["quality"].(int)
		alt := BookingAlternative{
			TeacherID:   t.ID,
			TeacherName: t.Name,
			BranchID:    req.BranchID,
			SubjectID:   req.SubjectID,
			StartTime:   o.start,
			EndTime:     o.end,
			Score:       quality,
			Reasons:     a.Reasons,
		}
		alt = enrichWithCommute(alt, o, req.BranchID, prefetched[t.ID].conflicts, commutePad)
		alts = append(alts, alt)
	}
	return alts, nil
}

// enrichWithCommute reports the travel time a chosen slot depends on: when the
// teacher has a different-branch booking whose gap to the slot is no larger
// than the commute pad (i.e. the travel time is the binding constraint), that
// pad is reported. Pure — commutePad is the single global travel time.
func enrichWithCommute(alt BookingAlternative, o timeWindow, reqBranchID int, conflicts []Booking, commutePad time.Duration) BookingAlternative {
	if commutePad <= 0 {
		return alt
	}
	var maxPad time.Duration
	for _, c := range conflicts {
		buf := padFor(c.BranchID, reqBranchID, commutePad)
		if buf <= 0 {
			continue
		}
		var gap time.Duration
		switch {
		case !c.EndTime.After(o.start): // booking is before the slot
			gap = o.start.Sub(c.EndTime)
		case !o.end.After(c.StartTime): // booking is after the slot
			gap = c.StartTime.Sub(o.end)
		default:
			continue // overlapping — not a valid slot
		}
		// Candidates land on a CandidateStepDuration grid, so a commute that
		// isn't a multiple of the step rounds the binding slot's gap up into
		// [buf, buf+step). Report the pad on that binding slot.
		if gap >= 0 && gap < buf+CandidateStepDuration && buf > maxPad {
			maxPad = buf
		}
	}
	if maxPad > 0 {
		alt.CommuteMinutes = shared.Some(int(maxPad.Minutes()))
	}
	return alt
}

// CommuteConflict reports whether placing a booking for teacherID at
// [start,end] in branchID leaves too little travel time given the teacher's
// existing different-branch bookings. Same-branch overlaps are assumed already
// handled by the caller (the exact-match query rejects time overlaps), so only
// different-branch bookings padded by their commute are considered. Returns
// false when no commute source is wired.
func (e *CLPEngine) CommuteConflict(ctx context.Context, teacherID, branchID int, start, end time.Time) (bool, error) {
	if e.commute == nil {
		return false, nil
	}
	buf, err := e.commute.DefaultCommute(ctx)
	if err != nil {
		return false, err
	}
	bookings, err := e.bookingStore.FindConflictingBookings(ctx, teacherID,
		start.Add(-buf), end.Add(buf))
	if err != nil {
		return false, err
	}
	for _, b := range bookings {
		pad := padFor(b.BranchID, branchID, buf)
		if pad <= 0 {
			continue
		}
		if overlaps(start, end, b.StartTime.Add(-pad), b.EndTime.Add(pad)) {
			return true, nil
		}
	}
	return false, nil
}

// padFor is the travel time a candidate must leave against an existing booking:
// zero when the booking is at the requested branch, otherwise the global pad.
func padFor(bookingBranchID, reqBranchID int, pad time.Duration) time.Duration {
	if bookingBranchID == reqBranchID {
		return 0
	}
	return pad
}

func overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
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
	lookbehind := CandidateLookbehind
	lookahead := CandidateLookahead
	step := CandidateStepDuration

	var windows []timeWindow
	for t := anchor.Add(-lookbehind); !t.After(anchor.Add(lookahead)); t = t.Add(step) {
		windows = append(windows, timeWindow{start: t, end: t.Add(duration)})
	}
	return windows
}
