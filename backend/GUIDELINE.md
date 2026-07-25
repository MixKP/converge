# Coding Guidelines

## 1. Result — Error Wrapping at Boundaries

All code that crosses a boundary (repository → service, service → handler) MUST return domain errors, not infrastructure errors. Use `Result[T]` to transform errors at the boundary without nested `if err != nil` chains in callers.

```go
// shared/result.go
package shared

type Result[T any] struct {
	value T
	err   error
}

func Ok[T any](val T) Result[T]    { return Result[T]{value: val} }
func Fail[T any](err error) Result[T] { return Result[T]{err: err} }

func (r Result[T]) Unwrap() (T, error)     { return r.value, r.err }
func (r Result[T]) Value() (T, bool)       { return r.value, r.err == nil }
func (r Result[T]) Error() error           { return r.err }
func (r Result[T]) IsOk() bool             { return r.err == nil }

func Map[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.err != nil {
		return Fail[U](r.err)
	}
	return Ok(fn(r.value))
}

func (r Result[T]) MapErr(fn func(error) error) Result[T] {
	if r.err != nil {
		r.err = fn(r.err)
	}
	return r
}
```

### Repository boundary

Bad — domain learns about `database/sql`:
```go
func (r *BookingRepo) DeleteBooking(ctx context.Context, id int) error {
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SchedulingService) Cancel(ctx context.Context, id int) error {
	err := s.bookingStore.DeleteBooking(ctx, id)
	if errors.Is(err, sql.ErrNoRows) { // domain imports database/sql
		...
	}
}
```

Good — domain stays infrastructure-free:
```go
func (r *BookingRepo) DeleteBooking(ctx context.Context, id int) error {
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return &NotFoundError{Msg: fmt.Sprintf("booking %d not found", id)}
	}
	return nil
}

func (s *SchedulingService) Cancel(ctx context.Context, id int) error {
	err := s.bookingStore.DeleteBooking(ctx, id)
	var notFound *NotFoundError
	if errors.As(err, &notFound) {
		...
	}
}
```

Good — using Result to map errors at the boundary:
```go
func (r *BookingRepo) Create(ctx context.Context, req ConfirmBookingRequest) Result[Booking] {
	var b Booking
	err := r.DB.QueryRowContext(ctx, query, ...).Scan(&b.ID, ...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
			return Fail[Booking](ErrBookingConflict)
		}
		return Fail[Booking](err)
	}
	return Ok(b)
}

func (s *SchedulingService) Confirm(ctx context.Context, req ConfirmBookingRequest) (*Booking, error) {
	return s.bookingStore.Create(ctx, req).
		MapErr(func(err error) error {
			if errors.Is(err, ErrBookingConflict) {
				return &ConflictError{Msg: "Teacher already has a booking in this time range"}
			}
			return err
		}).
		Unwrap()
}
```

---

## 2. Option — Explicit Nil Semantics

Fields that can be absent MUST use `Option[T]` instead of `*T`. The nil check happens at the API boundary (HTTP handler), never in domain logic.

```go
// shared/option.go
package shared

import "encoding/json"

type Option[T any] struct {
	value   T
	present bool
}

func Some[T any](val T) Option[T] { return Option[T]{value: val, present: true} }
func None[T any]() Option[T]      { return Option[T]{} }

func (o Option[T]) Value() (T, bool)      { return o.value, o.present }
func (o Option[T]) IsSome() bool          { return o.present }
func (o Option[T]) IsNone() bool          { return !o.present }
func (o Option[T]) Or(fallback T) T       { if o.present { return o.value }; return fallback }

func (o Option[T]) IfSome(fn func(T))     { if o.present { fn(o.value) } }

func (o Option[T]) MarshalJSON() ([]byte, error) {
	if !o.present { return json.Marshal(nil) }
	return json.Marshal(o.value)
}

func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" { return nil }
	o.present = true
	return json.Unmarshal(data, &o.value)
}
```

### Usage

```go
// Before
type BookingRequest struct {
	PreferredTeacherID *int `json:"preferred_teacher_id,omitempty"`
}
if req.PreferredTeacherID != nil && *req.PreferredTeacherID == teacherID { ... }

// After
type BookingRequest struct {
	PreferredTeacherID Option[int] `json:"preferred_teacher_id,omitempty"`
}

func (s *WeightedScorer) scoreTeacherPreference(candidate ScorableCandidate) (int, string) {
	prefID, ok := candidate.Request.PreferredTeacherID.Value()
	if !ok {
		return s.TeacherWeight / 2, "Teaches this subject"
	}
	if prefID == candidate.Teacher.ID {
		return s.TeacherWeight, "Your preferred teacher"
	}
	return 0, "Can also teach this subject"
}
```

---

## 3. Validation — Composable Functions

Validation MUST happen at the service boundary (first line of every public method). Reject invalid input before any work begins.

Use composable validators instead of inline `if field <= 0` blocks.

```go
// shared/validator.go
package shared

import "fmt"

type Validator[T any] func(T) error

func ValidateAll[T any](subject T, validators ...Validator[T]) error {
	for _, v := range validators {
		if err := v(subject); err != nil {
			return err
		}
	}
	return nil
}

func PositiveInt[T any](field string, get func(T) int) Validator[T] {
	return func(subject T) error {
		if get(subject) <= 0 {
			return &ValidationError{Msg: field + " must be positive"}
		}
		return nil
	}
}

func NonEmpty[T any](field string, get func(T) string) Validator[T] {
	return func(subject T) error {
		if get(subject) == "" {
			return &ValidationError{Msg: field + " must not be empty"}
		}
		return nil
	}
}

func DayOfWeekRange[T any](field string, get func(T) int) Validator[T] {
	return func(subject T) error {
		d := get(subject)
		if d < 0 || d > 6 {
			return &ValidationError{Msg: fmt.Sprintf("%s must be between 0 and 6, got %d", field, d)}
		}
		return nil
	}
}

func SliceNonEmpty[T any](field string, get func(T) int) Validator[T] {
	return func(subject T) error {
		if get(subject) == 0 {
			return &ValidationError{Msg: field + " must not be empty"}
		}
		return nil
	}
}
```

### Usage

```go
// Before — repetitive inline checks
func (s *SchedulingService) Confirm(ctx context.Context, req ConfirmBookingRequest) (*Booking, error) {
	if req.TeacherID <= 0 { return nil, &ValidationError{Msg: "teacher_id must be positive"} }
	if req.BranchID <= 0  { return nil, &ValidationError{Msg: "branch_id must be positive"} }
	if req.SubjectID <= 0 { return nil, &ValidationError{Msg: "subject_id must be positive"} }
	if req.ClientName == "" { return nil, &ValidationError{Msg: "client_name must not be empty"} }
	...
}

// After
func (s *SchedulingService) Confirm(ctx context.Context, req ConfirmBookingRequest) (*Booking, error) {
	if err := ValidateAll(req,
		PositiveInt("teacher_id", func(r ConfirmBookingRequest) int { return r.TeacherID }),
		PositiveInt("branch_id",  func(r ConfirmBookingRequest) int { return r.BranchID }),
		PositiveInt("subject_id", func(r ConfirmBookingRequest) int { return r.SubjectID }),
		NonEmpty("client_name",   func(r ConfirmBookingRequest) string { return r.ClientName }),
	); err != nil {
		return nil, err
	}
	...
}
```

---

## 4. Errors — Domain Types, Strongly Typed

Error semantics MUST be communicated by type, not by inspecting error strings.

Defined once in `shared/errors.go`:
```go
type ValidationError struct{ Msg string }
func (e *ValidationError) Error() string { return e.Msg }

type ConflictError struct{ Msg string }
func (e *ConflictError) Error() string { return e.Msg }

type NotFoundError struct{ Msg string }
func (e *NotFoundError) Error() string { return e.Msg }
```

All domain packages re-export via type aliases:
```go
type ValidationError = shared.ValidationError
type ConflictError   = shared.ConflictError
type NotFoundError   = shared.NotFoundError
```

### Repository error contract

Repositories MUST return domain errors for domain-expected failure modes. The caller checks with `errors.Is` / `errors.As`. Never with `sql.ErrNoRows` or any infrastructure type.

### Handler dispatch

```go
var valErr *scheduling.ValidationError
var confErr *scheduling.ConflictError
var notFoundErr *scheduling.NotFoundError

switch {
case errors.As(err, &valErr):
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
case errors.As(err, &confErr):
	c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
case errors.As(err, &notFoundErr):
	c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
default:
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}
```

---

## 5. Resource Cleanup — Defer at Acquisition

Any resource acquired (rows, files, contexts with cancel) MUST be deferred for cleanup on the immediate next line.

```go
// Correct
rows, err := r.DB.QueryContext(ctx, query, teacherID)
if err != nil {
	return nil, err
}
defer rows.Close()

// Wrong — deferred after error handling opens a window for leaks
rows, err := r.DB.QueryContext(ctx, query, teacherID)
if err != nil { return nil, err }
// ... intermediate code ...
defer rows.Close()
```

---

## 6. Immutability — Copy, Never Mutate

Functions that transform data MUST return a new value. Never mutate a pointer parameter or struct field from a child function.

```go
// Good — returns new copy
func (e *CLPEngine) enrichWithCommute(ctx context.Context, alt BookingAlternative, ...) BookingAlternative {
	alt.CommuteMinutes = &mins
	return alt
}

// Bad — mutates through pointer
func (e *CLPEngine) enrichWithCommute(ctx context.Context, alt *BookingAlternative, ...) {
	alt.CommuteMinutes = &mins
}
```

### CSP constraint constraints

Constraints and objectives receive `Assignment` by value. They MUST NOT mutate `a.Values` as a side-channel. If data must be shared between constraint and objective, restructure the model to carry metadata explicitly.

```go
// Current — side-channel mutation
model.AddConstraint(func(a Assignment) (bool, error) {
	slots, _ := e.teacherRoster.TeacherAvailability(ctx, t.ID)
	a.Values["slots"] = slots  // writing for objective
	...
})
```

---

## 7. Minimal Constructors — Accept Interfaces, Return Structs

All types with dependencies MUST expose a constructor that accepts interface types (ports), enabling test substitution.

```go
func NewCLPEngine(
	bookingStore  BookingStore,    // interface
	teacherRoster TeacherRoster,   // interface
	scorer        Scorer,          // interface
	commute       CommuteEstimate, // interface
	room          RoomCheck,       // interface
	logger        *slog.Logger,
) *CLPEngine {
	return &CLPEngine{...}
}
```

Web handlers MUST define their dependency as a file-private interface naming only the methods they actually call.

```go
type bookingService interface {
	Evaluate(ctx context.Context, req BookingRequest) (*BookingResponse, error)
	Confirm(ctx context.Context, req ConfirmBookingRequest) (*Booking, error)
	Cancel(ctx context.Context, bookingID int) error
	ListAll(ctx context.Context) ([]Booking, error)
}
```
