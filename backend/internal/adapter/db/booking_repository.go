package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/ports"
)

type BookingRepo struct {
	DB *sql.DB
}

func NewBookingRepository(database *sql.DB) *BookingRepo {
	return &BookingRepo{DB: database}
}

func (r *BookingRepo) FindExactMatch(ctx context.Context, req models.BookingRequest) (*ports.BookingMatch, error) {
	preferredEnd := req.PreferredStart.Add(time.Duration(req.DurationMinutes) * time.Minute)
	startTime := req.PreferredStart.Format("15:04")
	endTime := preferredEnd.Format("15:04")

	var teacherIDVal interface{}
	if req.PreferredTeacherID != nil && *req.PreferredTeacherID > 0 {
		teacherIDVal = *req.PreferredTeacherID
	}

	row := r.DB.QueryRowContext(ctx, `
		SELECT 0, t.id, $2::int, $1::int, $4::timestamptz, $5::timestamptz, t.name
		FROM teachers t
		JOIN teacher_subjects ts ON t.id = ts.teacher_id AND ts.subject_id = $1
		JOIN teacher_availability ta ON t.id = ta.teacher_id
		WHERE t.status = 'active'
		  AND ta.day_of_week = (EXTRACT(DOW FROM $4::timestamptz)::int + 6) % 7
		  AND ta.start_time <= $6::time
		  AND ta.end_time >= $7::time
		  AND ($3::int IS NULL OR t.id = $3)
		  AND NOT EXISTS (
		    SELECT 1 FROM bookings b
		    WHERE b.teacher_id = t.id
		      AND tstzrange(b.start_time, b.end_time) && tstzrange($4::timestamptz, $5::timestamptz)
		  )
		ORDER BY t.id
		LIMIT 1`,
		req.SubjectID, req.BranchID, teacherIDVal, req.PreferredStart, preferredEnd, startTime, endTime,
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
