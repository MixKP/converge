package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/RinZ5/converge/backend/internal/scheduling"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/jackc/pgx/v5/pgconn"
)

const pgExclusionViolation = "23P01"

type BookingRepo struct {
	DB *sql.DB
}

func NewBookingRepository(database *sql.DB) *BookingRepo {
	return &BookingRepo{DB: database}
}

func (r *BookingRepo) FindExactMatch(ctx context.Context, subjectID, branchID int, slot shared.WeeklySlot, durationMinutes int, teacherID shared.Option[int], gender string) (*scheduling.BookingMatch, error) {
	loc := shared.LoadLocation()
	duration := time.Duration(durationMinutes) * time.Minute

	windowEnd := slot.End
	if duration > 0 {
		parsedStart, err := time.ParseInLocation("15:04", string(slot.Start), loc)
		if err != nil {
			return nil, fmt.Errorf("invalid slot start time format: %w", err)
		}
		windowEnd = shared.TimeHHMM(parsedStart.Add(duration).Format("15:04"))
	}

	anchorDate := shared.AnchorDateForDay(slot.DayOfWeek, loc)
	parsedStart, err := time.Parse("15:04", string(slot.Start))
	if err != nil {
		return nil, fmt.Errorf("invalid slot start time format: %w", err)
	}
	startTS := time.Date(anchorDate.Year(), anchorDate.Month(), anchorDate.Day(), parsedStart.Hour(), parsedStart.Minute(), 0, 0, loc)
	endTS := startTS
	if duration > 0 {
		endTS = endTS.Add(duration)
	} else {
		parsedEnd, err := time.ParseInLocation("15:04", string(slot.End), loc)
		if err != nil {
			return nil, fmt.Errorf("invalid slot end time format: %w", err)
		}
		endTS = time.Date(anchorDate.Year(), anchorDate.Month(), anchorDate.Day(), parsedEnd.Hour(), parsedEnd.Minute(), 0, 0, loc)
	}

	row := r.DB.QueryRowContext(ctx, `
		SELECT 0, t.id, $2::int, $1::int, $5::timestamptz, $6::timestamptz, t.name
		FROM teachers t
		JOIN teacher_subjects ts ON t.id = ts.teacher_id AND ts.subject_id = $1
		JOIN teacher_availability ta ON t.id = ta.teacher_id
		WHERE t.status = 'active'
		  AND ta.day_of_week = $3
		  AND ta.start_time <= $4::time
		  AND ta.end_time >= $7::time
		  AND ($8::int IS NULL OR t.id = $8)
		  AND t.gender = $9
		  AND NOT EXISTS (
		    SELECT 1 FROM bookings b
		    WHERE b.teacher_id = t.id
		      AND tstzrange(b.start_time, b.end_time) && tstzrange($5::timestamptz, $6::timestamptz)
		  )
		LIMIT 1`,
		subjectID, branchID, slot.DayOfWeek, string(slot.Start), startTS, endTS, string(windowEnd), teacherID.SQL(), gender,
	)

	var booking scheduling.Booking
	var name string
	var bid int
	if err := row.Scan(&bid, &booking.TeacherID, &booking.BranchID, &booking.SubjectID,
		&booking.StartTime, &booking.EndTime, &name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	booking.ID = bid
	return &scheduling.BookingMatch{Booking: booking, TeacherName: name}, nil
}

// scanOverlappingBookings scans rows shaped like the overlap queries below
// (id, teacher_id, branch_id, subject_id, start_time, end_time).
func scanOverlappingBookings(rows *sql.Rows) ([]scheduling.Booking, error) {
	var bookings []scheduling.Booking
	for rows.Next() {
		var b scheduling.Booking
		if err := rows.Scan(&b.ID, &b.TeacherID, &b.BranchID, &b.SubjectID,
			&b.StartTime, &b.EndTime); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *BookingRepo) FindConflictingBookings(ctx context.Context, teacherID int, startTime, endTime time.Time) ([]scheduling.Booking, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, teacher_id, branch_id, subject_id, start_time, end_time
		FROM bookings
		WHERE teacher_id = $1
		  AND tstzrange(start_time, end_time) && tstzrange($2, $3)`,
		teacherID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOverlappingBookings(rows)
}

func (r *BookingRepo) CreateBooking(ctx context.Context, req scheduling.ConfirmBookingRequest) (*scheduling.Booking, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := requireStudent(ctx, tx, req.StudentID); err != nil {
		return nil, err
	}
	if err := requireActiveBranch(ctx, tx, req.BranchID); err != nil {
		return nil, err
	}
	if err := lockTeacher(ctx, tx, req.TeacherID); err != nil {
		return nil, err
	}
	commuteMinutes, err := readCommuteMinutes(ctx, tx)
	if err != nil {
		return nil, err
	}
	if err := checkCommute(ctx, tx, req.TeacherID, req.BranchID, req.StartTime, req.EndTime, commuteMinutes); err != nil {
		return nil, err
	}

	row := tx.QueryRowContext(ctx, `
		WITH ins AS (
			INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, student_id)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, teacher_id, branch_id, subject_id, start_time, end_time, student_id, created_at
		)
		SELECT ins.id, ins.teacher_id, ins.branch_id, ins.subject_id, ins.start_time, ins.end_time,
		       ins.student_id, u.name, ins.created_at
		FROM ins JOIN users u ON u.id = ins.student_id`,
		req.TeacherID, req.BranchID, req.SubjectID, req.StartTime, req.EndTime, req.StudentID,
	)

	var b scheduling.Booking
	if err := row.Scan(&b.ID, &b.TeacherID, &b.BranchID, &b.SubjectID,
		&b.StartTime, &b.EndTime, &b.StudentID, &b.StudentName, &b.CreatedAt); err != nil {
		if isExclusionViolation(err) {
			return nil, scheduling.ErrBookingConflict
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &b, nil
}

// RescheduleBooking moves one booking under the same per-teacher lock as
// CreateBooking. The teacher is read before the lock and the row is locked
// after it, the order every other writer of a teacher's bookings uses.
func (r *BookingRepo) RescheduleBooking(ctx context.Context, bookingID int, start, end, now time.Time) (*scheduling.Booking, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	notFound := &shared.NotFoundError{Msg: fmt.Sprintf("booking %d not found", bookingID)}

	var teacherID int
	if err := tx.QueryRowContext(ctx, `SELECT teacher_id FROM bookings WHERE id = $1`, bookingID).Scan(&teacherID); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound
		}
		return nil, err
	}
	if err := lockTeacher(ctx, tx, teacherID); err != nil {
		return nil, err
	}

	var branchID int
	var currentStart time.Time
	if err := tx.QueryRowContext(ctx,
		`SELECT branch_id, start_time FROM bookings WHERE id = $1 FOR UPDATE`, bookingID,
	).Scan(&branchID, &currentStart); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound
		}
		return nil, err
	}

	if !currentStart.After(now) {
		return nil, scheduling.ErrBookingStarted
	}
	if !start.After(now) {
		return nil, scheduling.ErrSessionInPast
	}
	if err := requireActiveBranch(ctx, tx, branchID); err != nil {
		return nil, err
	}
	inside, err := withinAvailability(ctx, tx, teacherID, start, end)
	if err != nil {
		return nil, err
	}
	if !inside {
		return nil, scheduling.ErrOutsideAvailability
	}
	commuteMinutes, err := readCommuteMinutes(ctx, tx)
	if err != nil {
		return nil, err
	}
	if err := checkCommute(ctx, tx, teacherID, branchID, start, end, commuteMinutes); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE bookings SET start_time = $2, end_time = $3 WHERE id = $1`, bookingID, start, end,
	); err != nil {
		if isExclusionViolation(err) {
			return nil, scheduling.ErrBookingConflict
		}
		return nil, err
	}

	rows, err := tx.QueryContext(ctx, bookingWithStudentSelect+` WHERE b.id = $1`, bookingID)
	if err != nil {
		return nil, err
	}
	moved, err := scanStudentBookings(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(moved) != 1 {
		return nil, notFound
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &moved[0], nil
}

func isExclusionViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgExclusionViolation
}

func requireStudent(ctx context.Context, tx *sql.Tx, studentID int) error {
	var studentRole string
	if err := tx.QueryRowContext(ctx, `SELECT role FROM users WHERE id = $1`, studentID).Scan(&studentRole); err != nil {
		if err == sql.ErrNoRows {
			return &shared.ValidationError{Msg: fmt.Sprintf("student %d not found", studentID)}
		}
		return err
	}
	if studentRole != "student" {
		return &shared.ValidationError{Msg: fmt.Sprintf("user %d is not a student", studentID)}
	}
	return nil
}

func requireActiveBranch(ctx context.Context, tx *sql.Tx, branchID int) error {
	var branchStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM branches WHERE id = $1`, branchID).Scan(&branchStatus); err != nil {
		if err == sql.ErrNoRows {
			return &shared.NotFoundError{Msg: fmt.Sprintf("branch %d not found", branchID)}
		}
		return err
	}
	if branchStatus != "active" {
		return &shared.ValidationError{Msg: fmt.Sprintf("branch %d is deactivated", branchID)}
	}
	return nil
}

func lockTeacher(ctx context.Context, tx *sql.Tx, teacherID int) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(1, $1::int)`, teacherID)
	return err
}

func readCommuteMinutes(ctx context.Context, tx *sql.Tx) (int, error) {
	var commuteMinutes int
	err := tx.QueryRowContext(ctx, `SELECT commute_minutes FROM commute_config WHERE id = 1 FOR SHARE`).Scan(&commuteMinutes)
	return commuteMinutes, err
}

func checkCommute(ctx context.Context, tx *sql.Tx, teacherID, branchID int, start, end time.Time, commuteMinutes int) error {
	if commuteMinutes <= 0 {
		return nil
	}
	var commuteConflict bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM bookings
			WHERE teacher_id = $1
			  AND branch_id <> $2
			  AND tstzrange(
				start_time - ($5::int * interval '1 minute'),
				end_time + ($5::int * interval '1 minute'),
				'[)'
			  ) && tstzrange($3, $4, '[)')
		)`,
		teacherID, branchID, start, end, commuteMinutes,
	).Scan(&commuteConflict); err != nil {
		return err
	}
	if commuteConflict {
		return scheduling.ErrCommuteConflict
	}
	return nil
}

// withinAvailability reads the weekly slots inside the transaction, so it
// serialises with ReplaceWeeklyAvailabilityAndDeleteBookingsFrom through the
// shared teacher lock. Slots are wall-clock times, so the new time is compared
// in the business timezone.
func withinAvailability(ctx context.Context, tx *sql.Tx, teacherID int, start, end time.Time) (bool, error) {
	var inside bool
	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM teacher_availability
			WHERE teacher_id = $1
			  AND ($2::timestamptz AT TIME ZONE $4)::date = ($3::timestamptz AT TIME ZONE $4)::date
			  AND day_of_week = EXTRACT(DOW FROM $2::timestamptz AT TIME ZONE $4)
			  AND start_time <= ($2::timestamptz AT TIME ZONE $4)::time
			  AND end_time >= ($3::timestamptz AT TIME ZONE $4)::time
		)`,
		teacherID, start, end, shared.LoadLocation().String(),
	).Scan(&inside)
	return inside, err
}

func (r *BookingRepo) DeleteBooking(ctx context.Context, bookingID int) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM bookings WHERE id = $1`, bookingID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return &shared.NotFoundError{Msg: fmt.Sprintf("booking %d not found", bookingID)}
	}
	return nil
}

const bookingWithStudentSelect = `
	SELECT b.id, b.teacher_id, t.name, b.branch_id, br.name, b.subject_id, s.name,
	       b.start_time, b.end_time, b.student_id, u.name, b.created_at
	FROM bookings b
	JOIN users u ON u.id = b.student_id
	JOIN teachers t ON t.id = b.teacher_id
	JOIN branches br ON br.id = b.branch_id
	JOIN subjects s ON s.id = b.subject_id`

func scanStudentBookings(rows *sql.Rows) ([]scheduling.Booking, error) {
	var bookings []scheduling.Booking
	for rows.Next() {
		var b scheduling.Booking
		if err := rows.Scan(&b.ID, &b.TeacherID, &b.TeacherName, &b.BranchID, &b.BranchName,
			&b.SubjectID, &b.SubjectName, &b.StartTime, &b.EndTime,
			&b.StudentID, &b.StudentName, &b.CreatedAt); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *BookingRepo) FindAllBookings(ctx context.Context) ([]scheduling.Booking, error) {
	rows, err := r.DB.QueryContext(ctx, bookingWithStudentSelect+` ORDER BY b.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStudentBookings(rows)
}

func (r *BookingRepo) FindBookingsByStudentIDs(ctx context.Context, studentIDs []int) ([]scheduling.Booking, error) {
	if len(studentIDs) == 0 {
		return nil, nil
	}
	query := bookingWithStudentSelect + ` WHERE b.student_id = ANY($1) ORDER BY b.created_at DESC`

	rows, err := r.DB.QueryContext(ctx, query, studentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStudentBookings(rows)
}
