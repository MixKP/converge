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

	var teacherID *int
	if req.PreferredTeacherID != nil && *req.PreferredTeacherID > 0 {
		teacherID = req.PreferredTeacherID
	}

	var row *sql.Row
	if teacherID != nil {
		row = r.DB.QueryRowContext(ctx, `
			SELECT b.id, b.teacher_id, b.branch_id, b.subject_id, b.start_time, b.end_time, t.name
			FROM bookings b
			JOIN teachers t ON t.id = b.teacher_id
			JOIN teacher_subjects ts ON t.id = ts.teacher_id AND ts.subject_id = $1
			JOIN teacher_availability ta ON t.id = ta.teacher_id
			WHERE b.subject_id = $1
			  AND b.branch_id = $2
			  AND b.teacher_id = $3
			  AND b.start_time = $4
			  AND b.end_time = $5
			  AND t.status = 'active'
			ORDER BY b.id
			LIMIT 1`,
			req.SubjectID, req.BranchID, *teacherID, req.PreferredStart, preferredEnd,
		)
	} else {
		row = r.DB.QueryRowContext(ctx, `
			SELECT t.id, 0, $2, $1, $4::timestamptz, $5::timestamptz, t.name
			FROM teachers t
			JOIN teacher_subjects ts ON t.id = ts.teacher_id AND ts.subject_id = $1
			JOIN teacher_availability ta ON t.id = ta.teacher_id
			WHERE t.status = 'active'
			  AND ta.day_of_week = (EXTRACT(DOW FROM $4::timestamptz)::int + 6) % 7
			  AND ta.start_time <= $3::time
			  AND ta.end_time >= $6::time
			  AND NOT EXISTS (
			    SELECT 1 FROM bookings b
			    WHERE b.teacher_id = t.id
			      AND tstzrange(b.start_time, b.end_time) && tstzrange($4::timestamptz, $5::timestamptz)
			  )
			ORDER BY t.id
			LIMIT 1`,
			req.SubjectID, req.BranchID, startTime, req.PreferredStart, preferredEnd, endTime,
		)
	}

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
