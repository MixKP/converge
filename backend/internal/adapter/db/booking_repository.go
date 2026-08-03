package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

func (r *BookingRepo) FindBookingsByBranch(ctx context.Context, branchID int, startTime, endTime time.Time) ([]scheduling.Booking, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, teacher_id, branch_id, subject_id, start_time, end_time
		FROM bookings
		WHERE branch_id = $1
		  AND tstzrange(start_time, end_time) && tstzrange($2, $3)`,
		branchID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOverlappingBookings(rows)
}

// CreateBooking enforces branch capacity atomically when a branch has one
// configured: it takes a per-branch advisory lock, counts overlapping
// bookings, and inserts within a single transaction, so concurrent confirms
// for the same branch cannot both pass the capacity check before either has
// committed. Branches with no capacity configured (capacity <= 0) skip the
// lock entirely, since there is nothing to serialize against.
func (r *BookingRepo) CreateBooking(ctx context.Context, req scheduling.ConfirmBookingRequest) (*scheduling.Booking, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var studentRole string
	if err := tx.QueryRowContext(ctx, `SELECT role FROM users WHERE id = $1`, req.StudentID).Scan(&studentRole); err != nil {
		if err == sql.ErrNoRows {
			return nil, &shared.ValidationError{Msg: fmt.Sprintf("student %d not found", req.StudentID)}
		}
		return nil, err
	}
	if studentRole != "student" {
		return nil, &shared.ValidationError{Msg: fmt.Sprintf("user %d is not a student", req.StudentID)}
	}

	var capacity int
	if err := tx.QueryRowContext(ctx, `SELECT capacity FROM branches WHERE id = $1`, req.BranchID).Scan(&capacity); err != nil {
		if err == sql.ErrNoRows {
			return nil, &shared.NotFoundError{Msg: fmt.Sprintf("branch %d not found", req.BranchID)}
		}
		return nil, err
	}

	if capacity > 0 {
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1::bigint)`, req.BranchID); err != nil {
			return nil, err
		}

		var overlapping int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM bookings
			WHERE branch_id = $1
			  AND tstzrange(start_time, end_time) && tstzrange($2, $3)`,
			req.BranchID, req.StartTime, req.EndTime,
		).Scan(&overlapping); err != nil {
			return nil, err
		}
		if overlapping >= capacity {
			return nil, scheduling.ErrBranchCapacityExceeded
		}
	}

	row := tx.QueryRowContext(ctx, `
		WITH ins AS (
			INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, student_id)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, teacher_id, branch_id, subject_id, start_time, end_time, student_id, created_at
		)
		SELECT ins.id, ins.teacher_id, ins.branch_id, ins.subject_id, ins.start_time, ins.end_time,
		       COALESCE(ins.student_id, 0), COALESCE(u.name, ''), ins.created_at
		FROM ins LEFT JOIN users u ON u.id = ins.student_id`,
		req.TeacherID, req.BranchID, req.SubjectID, req.StartTime, req.EndTime, req.StudentID,
	)

	var b scheduling.Booking
	if err := row.Scan(&b.ID, &b.TeacherID, &b.BranchID, &b.SubjectID,
		&b.StartTime, &b.EndTime, &b.StudentID, &b.StudentName, &b.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgExclusionViolation {
			return nil, scheduling.ErrBookingConflict
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &b, nil
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
	SELECT b.id, b.teacher_id, b.branch_id, b.subject_id, b.start_time, b.end_time,
	       COALESCE(b.student_id, 0), COALESCE(u.name, ''), b.created_at
	FROM bookings b
	LEFT JOIN users u ON u.id = b.student_id`

func scanStudentBookings(rows *sql.Rows) ([]scheduling.Booking, error) {
	bookings := []scheduling.Booking{}
	for rows.Next() {
		var b scheduling.Booking
		if err := rows.Scan(&b.ID, &b.TeacherID, &b.BranchID, &b.SubjectID,
			&b.StartTime, &b.EndTime, &b.StudentID, &b.StudentName, &b.CreatedAt); err != nil {
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
		return []scheduling.Booking{}, nil
	}
	placeholders := make([]string, len(studentIDs))
	args := make([]any, len(studentIDs))
	for i, id := range studentIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := bookingWithStudentSelect +
		fmt.Sprintf(` WHERE b.student_id IN (%s) ORDER BY b.created_at DESC`, strings.Join(placeholders, ","))

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStudentBookings(rows)
}
