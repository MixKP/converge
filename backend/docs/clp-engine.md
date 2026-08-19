## Generic CLP core (`model.go`, `solver.go`)

```
STRUCTURE Variable:
    Name    : string
    Domain  : list of value

STRUCTURE Assignment:
    Values    : map from variable name to value
    Metadata  : map from string to value   // scratch space shared by the
                                           // constraints and the objective
    Score     : integer
    Reasons   : list of string

STRUCTURE Model:
    Variables    : list of Variable
    Constraints  : list of function(Assignment) -> (bool, error)
    Objective    : function(Assignment) -> (score, reasons, error), may be NIL
    DedupKey     : function(Assignment) -> string, may be NIL


ALGORITHM Solve
    INPUT   model : Model, topN : integer
    OUTPUT  list of Assignment, error

    all ← CartesianProduct(model.Variables)

    valid ← empty list
    FOR EACH a IN all DO
        passed ← TRUE
        FOR EACH c IN model.Constraints DO
            ⟨ok, err⟩ ← c(a)
            IF err ≠ NIL THEN
                FAIL WITH err                  // one bad constraint aborts the
            END IF                             // whole solve
            IF NOT ok THEN
                passed ← FALSE
                BREAK
            END IF
        END FOR

        IF NOT passed THEN
            CONTINUE
        END IF

        IF model.Objective ≠ NIL THEN
            ⟨score, reasons, err⟩ ← model.Objective(a)
            IF err ≠ NIL THEN
                FAIL WITH err
            END IF
            a.Score ← score
            a.Reasons ← reasons
        END IF

        APPEND a TO valid
    END FOR

    STABLE SORT valid BY a.Score DESCENDING

    IF model.DedupKey ≠ NIL THEN
        seen ← empty set of string
        deduped ← empty list
        FOR EACH a IN valid DO                 // valid is already sorted, so the
            key ← model.DedupKey(a)            // first hit for a key is its best
            IF key ∉ seen THEN
                INSERT key INTO seen
                APPEND a TO deduped
            END IF
        END FOR
        valid ← deduped
    END IF

    IF |valid| > topN THEN
        valid ← valid[0 .. topN−1]
    END IF
    RETURN ⟨valid, NIL⟩
END ALGORITHM


ALGORITHM CartesianProduct                     // recursive
    INPUT   vars : list of Variable
    OUTPUT  list of Assignment

    IF vars is empty THEN
        RETURN [ Assignment with empty Values and empty Metadata ]
    END IF

    head ← vars[0]
    tail ← CartesianProduct(vars[1 ..])

    result ← empty list
    FOR EACH v IN head.Domain DO
        FOR EACH t IN tail DO
            a ← copy of t with fresh empty Metadata
            a.Values[head.Name] ← v
            APPEND a TO result
        END FOR
    END FOR
    RETURN result
END ALGORITHM
```

## Domain model construction (`clp_engine.go`)

```
ALGORITHM FindAlternativesForSlot
    INPUT   request : BookingRequest, window : WeeklySlot
    OUTPUT  list of Alternative, error

    ── 1. Build the teacher domain ──────────────────────────────
    ⟨teachers, err⟩ ← TeachersBySubject(request.SubjectID)
    IF err ≠ NIL THEN
        FAIL WITH err
    END IF
    teachers ← FILTER teachers WHERE teacher.Gender = request.RequiredGender
    IF teachers is empty THEN
        RETURN ⟨empty list, NIL⟩               // no alternatives, not an error
    END IF

    ── 2. Build the time-offset domain ──────────────────────────
    duration ← request.DurationMinutes
    IF duration = 0 THEN
        duration ← window.End − window.Start
    END IF

    anchor    ← calendar date matching window.DayOfWeek
    prefStart ← anchor + time-of-day(window.Start)

    offsets ← empty list
    t ← prefStart − LOOKBEHIND
    WHILE t ≤ prefStart + LOOKAHEAD DO
        APPEND ⟨t, t + duration⟩ TO offsets
        t ← t + STEP
    END WHILE
    // LOOKBEHIND = LOOKAHEAD = 2h, STEP = 30min ⇒ 9 candidate windows

    ── 3. Hoist all I/O out of the constraint and the objective ──
    // Neither may perform I/O: Solve calls them once per candidate, so every
    // lookup they need is fetched here and captured by the closures below.
    commutePad ← 0
    IF a commute source is wired THEN
        ⟨commutePad, err⟩ ← DefaultCommute()
        IF err ≠ NIL THEN
            FAIL WITH err                      // never suggest a slot whose
        END IF                                 // travel time is unverified
    END IF

    // Widen the fetch window by the pad so a booking just outside the candidate
    // range can still block an edge candidate once padded.
    fetchFrom ← prefStart − LOOKBEHIND − commutePad
    fetchTo   ← prefStart + LOOKAHEAD + duration + commutePad

    FOR EACH t IN teachers DO
        ⟨slots, err⟩ ← TeacherAvailability(t.ID)
        IF err ≠ NIL THEN
            FAIL WITH err
        END IF
        ⟨conflicts, err⟩ ← ConflictingBookings(t.ID, fetchFrom, fetchTo)
        IF err ≠ NIL THEN
            FAIL WITH err
        END IF
        prefetched[t.ID] ← ⟨slots, conflicts⟩
    END FOR

    capacity ← 0
    occupancy ← empty map from offset to integer
    ⟨c, err⟩ ← GetCapacity(request.BranchID)
    IF err ≠ NIL THEN
        WARN "capacity lookup failed"          // degrade, do not fail: capacity
    ELSE                                       // stays 0 ⇒ C3 is not enforced
        capacity ← c
    END IF

    IF capacity > 0 THEN
        ⟨branchBookings, err⟩ ← BookingsByBranch(request.BranchID, fetchFrom, fetchTo)
        IF err ≠ NIL THEN
            WARN "branch bookings lookup failed"
            capacity ← 0                       // skip enforcement
        ELSE
            FOR EACH o IN offsets DO
                occupancy[o] ← COUNT of b IN branchBookings
                                WHERE Overlaps(o, ⟨b.Start, b.End⟩)
            END FOR
        END IF
    END IF

    ── 4. Declare the model ─────────────────────────────────────
    model.AddVariable("teacher", teachers)
    model.AddVariable("offset",  offsets)

    model.AddConstraint( FUNCTION(a : Assignment) -> (bool, error)
        teacher ← a.Values["teacher"]
        offset  ← a.Values["offset"]
        ⟨slots, conflicts⟩ ← prefetched[teacher.ID]

        // Hand the availability slots to the objective, which is keyed on the
        // same Assignment and would otherwise have to re-read prefetched.
        a.Metadata["slots"] ← slots

        // C1 — the slot must sit wholly inside one weekly availability window
        IF NOT FitsAvailability(slots, offset) THEN
            RETURN ⟨FALSE, NIL⟩
        END IF

        // C2 — no collision with the teacher's existing bookings, each padded
        //      by travel time when it is at a different branch
        FOR EACH c IN conflicts DO
            pad ← PadFor(c.BranchID, request.BranchID, commutePad)
            IF Overlaps(offset, ⟨c.Start − pad, c.End + pad⟩) THEN
                RETURN ⟨FALSE, NIL⟩
            END IF
        END FOR

        // C3 — the branch must have a room free for that window
        IF capacity > 0 AND occupancy[offset] ≥ capacity THEN
            RETURN ⟨FALSE, NIL⟩
        END IF

        RETURN ⟨TRUE, NIL⟩
    END FUNCTION )

    model.SetObjective( FUNCTION(a : Assignment) -> (score, reasons, error)
        teacher ← a.Values["teacher"]
        offset  ← a.Values["offset"]
        slots   ← a.Metadata["slots"]          // written by the constraint above
        ⟨quality, reasons⟩ ← Score(teacher, offset, request, slots)   // 0..100

        steps ← |offset.Start − prefStart| / STEP

        // Proximity dominates: PROXIMITY_WEIGHT (1000) ≫ any quality score, so
        // the nearest valid slot always wins and quality only breaks ties.
        // The ranking score below is internal; the raw quality is stashed in
        // metadata because that — not the ranking score — is what the caller
        // reports as Alternative.Score.
        a.Metadata["quality"] ← quality
        RETURN ⟨ quality − steps × PROXIMITY_WEIGHT , reasons , NIL ⟩
    END FUNCTION )

    model.SetDedupKey( FUNCTION(a : Assignment) -> string
        RETURN string(a.Values["teacher"].ID)  // at most one suggestion per
    END FUNCTION )                             // teacher: their best window

    ── 5. Solve and present ─────────────────────────────────────
    ⟨assignments, err⟩ ← Solve(model, MAX_ALTERNATIVES = 3)
    IF err ≠ NIL THEN
        FAIL WITH err
    END IF

    alternatives ← empty list
    FOR EACH a IN assignments DO
        teacher ← a.Values["teacher"]
        offset  ← a.Values["offset"]
        alt ← Alternative{
            Teacher   ← teacher,
            Branch    ← request.BranchID,
            Subject   ← request.SubjectID,
            Start     ← offset.Start,
            End       ← offset.End,
            Score     ← a.Metadata["quality"],   // raw 0..100, not a.Score
            Reasons   ← a.Reasons
        }
        ⟨_, conflicts⟩ ← prefetched[teacher.ID]
        alt ← EnrichWithCommute(alt, offset, request.BranchID, conflicts, commutePad)
        APPEND alt TO alternatives
    END FOR
    RETURN ⟨alternatives, NIL⟩
END ALGORITHM
```

## Supporting predicates

```
FUNCTION FitsAvailability
    INPUT   weeklySlots : list of WeeklySlot, candidate : offset
    OUTPUT  bool

    FOR EACH slot IN weeklySlots DO
        IF slot.DayOfWeek ≠ candidate.Weekday THEN
            CONTINUE
        END IF
        IF candidate.StartTimeOfDay ≥ slot.Start
           AND candidate.EndTimeOfDay ≤ slot.End THEN
            RETURN TRUE
        END IF
    END FOR
    RETURN FALSE
END FUNCTION


FUNCTION Overlaps
    INPUT   a : interval, b : interval
    OUTPUT  bool

    RETURN a.Start < b.End AND b.Start < a.End
END FUNCTION


FUNCTION PadFor
    INPUT   bookingBranchID : integer, requestBranchID : integer,
            pad : duration
    OUTPUT  duration

    // A booking at the requested branch needs no travel time; any other branch
    // costs the single global pad.
    IF bookingBranchID = requestBranchID THEN
        RETURN 0
    END IF
    RETURN pad
END FUNCTION


FUNCTION EnrichWithCommute
    INPUT   alt : Alternative, offset : offset, requestBranchID : integer,
            conflicts : list of Booking, commutePad : duration
    OUTPUT  Alternative

    // Report travel time only when it is the *binding* constraint on this slot.
    IF commutePad ≤ 0 THEN
        RETURN alt
    END IF

    maxPad ← 0
    FOR EACH c IN conflicts DO
        pad ← PadFor(c.BranchID, requestBranchID, commutePad)
        IF pad ≤ 0 THEN
            CONTINUE                           // same branch: no travel needed
        END IF

        IF c.End ≤ offset.Start THEN
            gap ← offset.Start − c.End         // booking is before the slot
        ELSE IF offset.End ≤ c.Start THEN
            gap ← c.Start − offset.End         // booking is after the slot
        ELSE
            CONTINUE                           // overlapping: not a valid slot
        END IF

        // Candidates land on a STEP grid, so a commute that is not a multiple
        // of STEP rounds the binding slot's gap up into [pad, pad + STEP).
        IF 0 ≤ gap < pad + STEP AND pad > maxPad THEN
            maxPad ← pad
        END IF
    END FOR

    IF maxPad > 0 THEN
        alt.CommuteMinutes ← maxPad in minutes
    END IF
    RETURN alt
END FUNCTION
```

## Scoring (`scorer.go`)

```
FUNCTION Score
    INPUT   teacher, candidateWindow, request, availabilitySlots
    OUTPUT  score in 0..100, reasons : list of string

    ⟨teacherScore, teacherReason⟩ ← ScoreTeacherPreference(teacher, request)
    ⟨fitScore, fitReason⟩ ← ScoreAvailabilityFit(candidateWindow, availabilitySlots)
    RETURN ⟨ teacherScore + fitScore , [teacherReason, fitReason] ⟩
END FUNCTION


FUNCTION ScoreTeacherPreference                // TEACHER_WEIGHT = 50
    INPUT   teacher, request
    OUTPUT  score, reason

    IF request has no PreferredTeacher THEN
        RETURN ⟨ TEACHER_WEIGHT / 2 , "Teaches this subject" ⟩
    END IF
    IF request.PreferredTeacher = teacher.ID THEN
        RETURN ⟨ TEACHER_WEIGHT , "Your preferred teacher" ⟩
    END IF
    RETURN ⟨ 0 , "Can also teach this subject" ⟩
END FUNCTION


FUNCTION ScoreAvailabilityFit                  // FIT_WEIGHT = 50
    INPUT   candidateWindow, availabilitySlots
    OUTPUT  score, reason

    IF candidateWindow.Start or candidateWindow.End is unset THEN
        RETURN ⟨ FIT_WEIGHT / 2 , "Schedule unavailable" ⟩
    END IF
    IF availabilitySlots is empty THEN
        RETURN ⟨ FIT_WEIGHT / 2 , "Schedule unavailable" ⟩
    END IF

    FOR EACH slot IN availabilitySlots DO
        IF slot.DayOfWeek ≠ candidateWindow.Weekday THEN
            CONTINUE
        END IF
        IF NOT (candidateWindow.StartTimeOfDay ≥ slot.Start
                AND candidateWindow.EndTimeOfDay ≤ slot.End) THEN
            CONTINUE
        END IF

        // buffer is the smaller of the two margins between the candidate window
        // and the enclosing availability window
        buffer   ← MIN( candidateWindow.Start − slot.Start ,
                        slot.End − candidateWindow.End )
        duration ← candidateWindow.End − candidateWindow.Start
        ratio    ← buffer / duration

        IF ratio ≥ 2 THEN
            RETURN ⟨ FIT_WEIGHT , "Plenty of availability" ⟩
        ELSE IF ratio ≥ 1 THEN
            RETURN ⟨ 25 , "Well within schedule" ⟩
        ELSE IF ratio ≥ 0.5 THEN
            RETURN ⟨ 15 , "Fits available time" ⟩
        ELSE IF buffer < 1 minute THEN
            RETURN ⟨ 5 , "Time adjusted to match availability" ⟩
        ELSE
            RETURN ⟨ 5 , "Tightly fits schedule" ⟩
        END IF
    END FOR

    RETURN ⟨ 5 , "Tightly fits schedule" ⟩     // no availability window matched
END FUNCTION
```

## Layer 3 Orchestration (`service.go`)

```
ALGORITHM Evaluate
    INPUT   request : BookingRequest
    OUTPUT  list of SlotResult, error

    ── Validate ────────────────────────────────────────────────
    VALIDATE request.SubjectID > 0
    VALIDATE request.BranchID > 0
    VALIDATE request.PreferredSlots is not empty
    VALIDATE request.RequiredGender ∈ {male, female, lgbtq+}
    FOR EACH slot IN request.PreferredSlots DO
        VALIDATE 0 ≤ slot.DayOfWeek ≤ 6
        VALIDATE slot.Start ≠ "" AND slot.End ≠ ""
        VALIDATE slot.Start < slot.End
    END FOR
    // any failed VALIDATE fails the request with a validation error

    ── Resolve each requested slot ─────────────────────────────
    results ← empty list
    FOR EACH slot IN request.PreferredSlots DO
        ⟨match, err⟩ ← FindExactMatch(request.SubjectID, request.BranchID, slot,
                                      request.DurationMinutes,
                                      request.PreferredTeacherID,
                                      request.RequiredGender)
        IF err ≠ NIL THEN
            FAIL WITH err
        END IF

        IF match ≠ NIL THEN
            ⟨blocked, err⟩ ← CommuteConflict(match.TeacherID, request.BranchID,
                                             match.Start, match.End)
            IF err ≠ NIL THEN
                FAIL WITH err
            END IF
            IF NOT blocked THEN
                APPEND SlotResult{ Slot ← slot,
                                   ExactMatch ← match with Score ← 100
                                                and Reasons ← ["Exact match"],
                                   Message ← "Exact match found" } TO results
                CONTINUE                       // next slot
            END IF
            // else: the matched teacher cannot physically get there in time, so
            // this is not truly an exact match — fall through to alternatives
        END IF

        ⟨alternatives, err⟩ ← FindAlternativesForSlot(request, slot)
        IF err ≠ NIL THEN
            FAIL WITH err
        END IF
        APPEND SlotResult{ Slot ← slot,
                           Alternatives ← alternatives,
                           Message ← summary of |alternatives| } TO results
    END FOR

    RETURN ⟨results, NIL⟩
END ALGORITHM


ALGORITHM CommuteConflict
    INPUT   teacherID, branchID, start, end
    OUTPUT  bool, error

    IF no commute source is wired THEN
        RETURN ⟨FALSE, NIL⟩
    END IF
    ⟨pad, err⟩ ← DefaultCommute()
    IF err ≠ NIL THEN
        FAIL WITH err
    END IF

    // Widen the lookup by the pad, mirroring FindAlternativesForSlot.
    ⟨bookings, err⟩ ← ConflictingBookings(teacherID, start − pad, end + pad)
    IF err ≠ NIL THEN
        FAIL WITH err
    END IF

    // Same-branch time overlaps are already rejected by FindExactMatch, so only
    // different-branch bookings padded by travel time are considered here.
    FOR EACH b IN bookings DO
        buf ← PadFor(b.BranchID, branchID, pad)
        IF buf ≤ 0 THEN
            CONTINUE
        END IF
        IF Overlaps(⟨start, end⟩, ⟨b.Start − buf, b.End + buf⟩) THEN
            RETURN ⟨TRUE, NIL⟩
        END IF
    END FOR
    RETURN ⟨FALSE, NIL⟩
END ALGORITHM
```
