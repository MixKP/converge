## Generic CLP core (`model.go`, `solver.go`)

```
STRUCTURE Model:
    Variables    : list of (Name, Domain[])
    Constraints  : list of predicate(Assignment) -> bool
    Objective    : function(Assignment) -> (score, reasons)
    DedupKey     : function(Assignment) -> string

ALGORITHM Solve(Model, topN):
    candidates ← CartesianProduct(Model.Variables)      // every combination

    valid ← empty list
    FOR EACH a IN candidates:
        FOR EACH c IN Model.Constraints:
            IF NOT c(a) THEN skip a and continue to next candidate
        (a.Score, a.Reasons) ← Model.Objective(a)
        append a to valid

    STABLE SORT valid BY a.Score DESCENDING

    IF Model.DedupKey exists:
        seen ← empty set
        keep only the first a for each DedupKey(a)      // best-per-key survives
                                                        // because list is sorted
    RETURN first topN of valid


ALGORITHM CartesianProduct(vars):                       // recursive
    IF vars is empty: RETURN [ empty assignment ]
    tail ← CartesianProduct(vars[1..])
    RETURN { tail-assignment + (vars[0].Name ← v)
             | v IN vars[0].Domain, tail-assignment IN tail }
```

## Domain model construction (`clp_engine.go`)

```
ALGORITHM FindAlternativesForSlot(request, window):

    ── 1. Build the teacher domain ──────────────────────────────
    teachers ← TeachersBySubject(request.SubjectID)
    teachers ← FILTER teachers WHERE teacher.Gender = request.RequiredGender
    IF teachers is empty: RETURN no alternatives

    ── 2. Build the time-offset domain ──────────────────────────
    duration ← request.DurationMinutes
               OR (window.End − window.Start) if unspecified

    anchor    ← calendar date matching window.DayOfWeek
    prefStart ← anchor + window.Start

    offsets ← { (t, t + duration)
                | t = prefStart − LOOKBEHIND, stepping by STEP,
                  while t ≤ prefStart + LOOKAHEAD }
              // LOOKBEHIND = LOOKAHEAD = 2h, STEP = 30min ⇒ 9 windows

    ── 3. Hoist all I/O out of the constraints (they must stay pure) ──
    commutePad ← DefaultCommute()  , or 0 if no commute source
                 // on error: FAIL the request — never suggest a slot
                 // with unverified travel time

    // widen the fetch window by the pad so a booking just outside the
    // candidate range can still block an edge candidate once padded
    fetchFrom ← prefStart − LOOKBEHIND − commutePad
    fetchTo   ← prefStart + LOOKAHEAD + duration + commutePad

    FOR EACH t IN teachers:
        prefetched[t] ← ( TeacherAvailability(t),
                          ConflictingBookings(t, fetchFrom, fetchTo) )

    capacity ← GetCapacity(request.BranchID)     // on error: warn, capacity ← 0
    IF capacity > 0:
        branchBookings ← BookingsByBranch(request.BranchID, fetchFrom, fetchTo)
                         // on error: warn, capacity ← 0 (skip enforcement)
        FOR EACH o IN offsets:
            occupancy[o] ← COUNT of branchBookings overlapping o

    ── 4. Declare the model ─────────────────────────────────────
    model.AddVariable("teacher", teachers)
    model.AddVariable("offset",  offsets)

    model.AddConstraint( λ(teacher, offset):
        (slots, conflicts) ← prefetched[teacher]

        // C1 — the slot must sit wholly inside one weekly availability window
        IF NOT FitsAvailability(slots, offset): RETURN false

        // C2 — no collision with the teacher's existing bookings, each padded
        //      by travel time when it is at a different branch
        FOR EACH c IN conflicts:
            pad ← (c.BranchID = request.BranchID) ? 0 : commutePad
            IF Overlaps(offset, [c.Start − pad, c.End + pad]): RETURN false

        // C3 — the branch must have a room free for that window
        IF capacity > 0 AND occupancy[offset] ≥ capacity: RETURN false

        RETURN true )

    model.SetObjective( λ(teacher, offset):
        quality ← Scorer.Score(teacher, offset, request, slots)   // 0..100
        steps   ← |offset.Start − prefStart| / STEP
        // proximity dominates: PROXIMITY_WEIGHT (1000) ≫ any quality score,
        // so the nearest valid slot always wins and quality only breaks ties
        RETURN ( quality − steps × PROXIMITY_WEIGHT , quality.Reasons ) )

    model.SetDedupKey( λ(teacher, offset): teacher.ID )
        // at most one suggestion per teacher — their best-ranked window

    ── 5. Solve and present ─────────────────────────────────────
    assignments ← Solve(model, MAX_ALTERNATIVES = 3)

    RETURN [ Alternative{ teacher, offset, quality, reasons,
                          CommuteMinutes ← EnrichWithCommute(...) }
             for each assignment ]
```

## Supporting predicates

```
FUNCTION FitsAvailability(weeklySlots, candidate):
    FOR EACH slot IN weeklySlots WHERE slot.DayOfWeek = candidate.Weekday:
        IF candidate.StartTimeOfDay ≥ slot.Start
           AND candidate.EndTimeOfDay ≤ slot.End:
            RETURN true
    RETURN false

FUNCTION Overlaps(a, b):
    RETURN a.Start < b.End AND b.Start < a.End

FUNCTION EnrichWithCommute(alt, offset, conflicts, commutePad):
    // report travel time only when it is the *binding* constraint on this slot
    maxPad ← 0
    FOR EACH c IN conflicts at a different branch:
        gap ← time between c and the candidate window (skip if overlapping)
        IF 0 ≤ gap < commutePad + STEP:      // rounded up onto the 30-min grid
            maxPad ← max(maxPad, commutePad)
    RETURN maxPad (if > 0)
```

## Scoring (`scorer.go`)

```
FUNCTION Score(candidate) -> (0..100, reasons):
    teacherScore ← 50  if candidate.Teacher = request.PreferredTeacher
                   25  if no preferred teacher was given
                    0  otherwise
    fitScore     ← f(buffer / duration), where buffer is the smaller margin
                   between the candidate window and the enclosing availability
                   window: 50 (ratio ≥ 2), 25 (≥ 1), 15 (≥ 0.5), else 5
    RETURN teacherScore + fitScore
```

## Layer 3 Orchestration (`service.go`)

```
ALGORITHM Evaluate(request):
    VALIDATE subject, branch, gender ∈ {male, female, lgbtq+},
             each slot has 0 ≤ dayOfWeek ≤ 6 and start < end

    FOR EACH slot IN request.PreferredSlots:
        match ← FindExactMatch(subject, branch, slot, duration, teacher, gender)

        IF match exists:
            IF NOT CommuteConflict(match.teacher, branch, match.times):
                emit ExactMatch(score = 100)   and continue to next slot
            // else: teacher can't physically get there — fall through

        emit Alternatives ← FindAlternativesForSlot(request, slot)

    RETURN all slot results
```
