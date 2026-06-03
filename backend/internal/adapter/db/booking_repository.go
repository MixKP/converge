package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/ports"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrBookingConflict = errors.New("booking conflict: teacher already has a booking in this time range")

type BookingRepo struct {
	DB *sql.DB
}

func NewBookingRepository(database *sql.DB) *BookingRepo {
	return &BookingRepo{DB: database}
}

func (r *BookingRepo) FindExactMatch(ctx context.Context, subjectID, branchID int, slot models.WeeklySlot, durationMinutes int, teacherIDVal interface{}) (*ports.BookingMatch, error) {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone: %w", err)
	}

	duration := time.Duration(durationMinutes) * time.Minute

	windowEnd := slot.End
	if duration > 0 {
		parsedStart, err := time.ParseInLocation("15:04", string(slot.Start), loc)
		if err != nil {
			return nil, fmt.Errorf("invalid slot start time format: %w", err)
		}
		windowEnd = models.TimeHHMM(parsedStart.Add(duration).Format("15:04"))
	}
	anchorDate := time.Now().In(loc).Truncate(24 * time.Hour)
	desiredWeekday := time.Weekday(slot.DayOfWeek)
	for anchorDate.Weekday() != desiredWeekday {
		anchorDate = anchorDate.Add(24 * time.Hour)
	}
	parsedStart, err := time.ParseInLocation("15:04", string(slot.Start), loc)
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
		  AND NOT EXISTS (
		    SELECT 1 FROM bookings b
		    WHERE b.teacher_id = t.id
		      AND tstzrange(b.start_time, b.end_time) && tstzrange($5::timestamptz, $6::timestamptz)
		  )
		ORDER BY t.id
		LIMIT 1`,
		subjectID, branchID, slot.DayOfWeek, string(slot.Start), startTS, endTS, string(windowEnd), teacherIDVal,
	)

	var booking models.Booking
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
	return &ports.BookingMatch{Booking: booking, TeacherName: name}, nil
}

func (r *BookingRepo) FindTeachersBySubject(ctx context.Context, subjectID int) ([]models.Teacher, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT t.id, t.name, t.email
		FROM teachers t
		JOIN teacher_subjects ts ON t.id = ts.teacher_id
		WHERE ts.subject_id = $1 AND t.status = 'active'
		ORDER BY t.name`, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teachers []models.Teacher
	for rows.Next() {
		var tc models.Teacher
		if err := rows.Scan(&tc.ID, &tc.Name, &tc.Email); err != nil {
			return nil, err
		}
		teachers = append(teachers, tc)
	}
	return teachers, rows.Err()
}

func (r *BookingRepo) FindConflictingBookings(ctx context.Context, teacherID int, startTime, endTime time.Time) ([]models.Booking, error) {
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

	var bookings []models.Booking
	for rows.Next() {
		var b models.Booking
		if err := rows.Scan(&b.ID, &b.TeacherID, &b.BranchID, &b.SubjectID,
			&b.StartTime, &b.EndTime); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *BookingRepo) FindTeacherAvailability(ctx context.Context, teacherID int) ([]models.WeeklySlot, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT day_of_week, to_char(start_time, 'HH24:MI') AS start_time, to_char(end_time, 'HH24:MI') AS end_time
		FROM teacher_availability
		WHERE teacher_id = $1
		ORDER BY day_of_week, start_time`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []models.WeeklySlot
	for rows.Next() {
		var s models.WeeklySlot
		if err := rows.Scan(&s.DayOfWeek, &s.Start, &s.End); err != nil {
			return nil, err
		}
		slots = append(slots, s)
	}
	return slots, rows.Err()
}

func (r *BookingRepo) CreateBooking(ctx context.Context, req models.ConfirmBookingRequest) (*models.Booking, error) {
	row := r.DB.QueryRowContext(ctx, `
		INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, client_name)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, teacher_id, branch_id, subject_id, start_time, end_time, client_name, created_at`,
		req.TeacherID, req.BranchID, req.SubjectID, req.StartTime, req.EndTime, req.ClientName,
	)

	var b models.Booking
	if err := row.Scan(&b.ID, &b.TeacherID, &b.BranchID, &b.SubjectID,
		&b.StartTime, &b.EndTime, &b.ClientName, &b.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
			return nil, ErrBookingConflict
		}
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
		return sql.ErrNoRows
	}
	return nil
}

func (r *BookingRepo) FindAllBookings(ctx context.Context) ([]models.Booking, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, teacher_id, branch_id, subject_id, start_time, end_time, client_name, created_at
		FROM bookings
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []models.Booking
	for rows.Next() {
		var b models.Booking
		if err := rows.Scan(&b.ID, &b.TeacherID, &b.BranchID, &b.SubjectID,
			&b.StartTime, &b.EndTime, &b.ClientName, &b.CreatedAt); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}
