package scheduling

import (
	"context"
	"sort"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type solverCandidate struct {
	Teacher           TeacherInfo
	Start             time.Time
	End               time.Time
	AvailabilitySlots []shared.WeeklySlot
	Score             int
	Reasons           []string
}

type Solver struct{}

func (e *CLPEngine) newSolver() *Solver {
	return &Solver{}
}

type solverObjectiveFn func(teacher TeacherInfo, start, end time.Time, availSlots []shared.WeeklySlot, req BookingRequest) (int, []string)

func (s *Solver) Solve(
	ctx context.Context,
	teachers []TeacherInfo,
	req BookingRequest,
	window shared.WeeklySlot,
	scorer solverObjectiveFn,
	roster TeacherRoster,
	bookingStore BookingStore,
) ([]solverCandidate, error) {
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

	var valid []solverCandidate
	for _, teacher := range teachers {
		availSlots, err := roster.TeacherAvailability(ctx, teacher.ID)
		if err != nil {
			return nil, err
		}

		for _, off := range offsets {
			if !fitsAvailability(availSlots, off.start, off.end) {
				continue
			}
			conflicts, err := bookingStore.FindConflictingBookings(ctx, teacher.ID, off.start, off.end)
			if err != nil {
				return nil, err
			}
			if len(conflicts) > 0 {
				continue
			}

			score, reasons := scorer(teacher, off.start, off.end, availSlots, req)
			valid = append(valid, solverCandidate{
				Teacher:           teacher,
				Start:             off.start,
				End:               off.end,
				AvailabilitySlots: availSlots,
				Score:             score,
				Reasons:           reasons,
			})
		}
	}

	sort.Slice(valid, func(i, j int) bool {
		if valid[i].Score != valid[j].Score {
			return valid[i].Score > valid[j].Score
		}
		return valid[i].Teacher.ID < valid[j].Teacher.ID
	})
	valid = deduplicateByTeacherSolver(valid)
	if len(valid) > MaxAlternatives {
		valid = valid[:MaxAlternatives]
	}
	return valid, nil
}

func fitsAvailability(slots []shared.WeeklySlot, start, end time.Time) bool {
	candStart := timeOfDay(start)
	candEnd := timeOfDay(end)
	for _, slot := range slots {
		slotWeekday := time.Weekday((slot.DayOfWeek + 1) % 7)
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

func deduplicateByTeacherSolver(candidates []solverCandidate) []solverCandidate {
	seen := make(map[int]bool)
	var result []solverCandidate
	for _, c := range candidates {
		if seen[c.Teacher.ID] {
			continue
		}
		seen[c.Teacher.ID] = true
		result = append(result, c)
	}
	return result
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

type timeWindow struct {
	start time.Time
	end   time.Time
}
