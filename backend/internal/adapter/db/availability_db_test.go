//go:build integration

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepoGetActiveTeachers(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, gender, status) VALUES (1, 'Test Teacher', 'test@teacher.com', 'male', 'active')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teachers (id, name, gender, status) VALUES (2, 'Inactive Teacher', 'female', 'deactivated')`)
	require.NoError(t, err)

	teachers, err := repo.GetActiveTeachers(context.Background())
	require.NoError(t, err)
	assert.Len(t, teachers, 1)
	assert.Equal(t, "Test Teacher", teachers[0].Name)
}

func TestPostgresRepoGetAllTeachers(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, gender, status) VALUES (1, 'Active Teacher', 'active@teacher.com', 'male', 'active')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teachers (id, name, gender, status) VALUES (2, 'Inactive Teacher', 'female', 'deactivated')`)
	require.NoError(t, err)

	teachers, err := repo.GetAllTeachers(context.Background())
	require.NoError(t, err)
	assert.Len(t, teachers, 2)

	byName := map[string]string{}
	for _, tt := range teachers {
		byName[tt.Name] = tt.Status
	}
	assert.Equal(t, "active", byName["Active Teacher"])
	assert.Equal(t, "deactivated", byName["Inactive Teacher"])
}

func TestPostgresRepoGetInactiveTeachers(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, gender, status) VALUES (1, 'Active Teacher', 'active@teacher.com', 'male', 'active')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teachers (id, name, gender, status) VALUES (2, 'Inactive Teacher', 'female', 'deactivated')`)
	require.NoError(t, err)

	teachers, err := repo.GetInactiveTeachers(context.Background())
	require.NoError(t, err)
	assert.Len(t, teachers, 1)
	assert.Equal(t, "Inactive Teacher", teachers[0].Name)
	assert.Equal(t, "deactivated", teachers[0].Status)
}

func countTeacherSlots(t *testing.T, db *sql.DB, teacherID int) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM teacher_availability WHERE teacher_id = $1`, teacherID).Scan(&count))
	return count
}

func TestPostgresRepoReplaceWeeklyAvailability_FirstSubmission(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, gender, status) VALUES (1, 'Test Teacher', 'test@teacher.com', 'male', 'active')`)
	require.NoError(t, err)

	err = repo.ReplaceWeeklyAvailability(context.Background(), 1, []shared.WeeklySlot{
		{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")},
		{DayOfWeek: 0, Start: shared.TimeHHMM("11:00"), End: shared.TimeHHMM("12:00")},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, countTeacherSlots(t, db, 1))
}

func TestPostgresRepoReplaceWeeklyAvailability_ReplacesPrevious(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, gender, status) VALUES (1, 'Test Teacher', 'test@teacher.com', 'male', 'active')`)
	require.NoError(t, err)
	require.NoError(t, repo.ReplaceWeeklyAvailability(context.Background(), 1, []shared.WeeklySlot{
		{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")},
		{DayOfWeek: 0, Start: shared.TimeHHMM("11:00"), End: shared.TimeHHMM("12:00")},
	}))

	err = repo.ReplaceWeeklyAvailability(context.Background(), 1, []shared.WeeklySlot{
		{DayOfWeek: 2, Start: shared.TimeHHMM("14:00"), End: shared.TimeHHMM("16:00")},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, countTeacherSlots(t, db, 1))
}

func TestPostgresRepoReplaceWeeklyAvailabilityAndDeleteBookingsFrom(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	teacherAID, teacherBID, branchID, subjectID := seedCappedBranchWithTwoTeachers(t, db, -1)
	studentID := seedStudent(t, db, "Student")
	insertBooking := func(teacherID int, start time.Time) int {
		var id int
		require.NoError(t, db.QueryRow(`
			INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, student_id)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			teacherID, branchID, subjectID, start, start.Add(time.Hour), studentID).Scan(&id))
		return id
	}
	weekStart := time.Date(2026, 10, 18, 0, 0, 0, 0, shared.LoadLocation())
	lastWeekID := insertBooking(teacherAID, weekStart.Add(-time.Hour))
	insertBooking(teacherAID, weekStart)
	insertBooking(teacherAID, weekStart.Add(3*24*time.Hour))
	insertBooking(teacherAID, weekStart.Add(10*24*time.Hour))
	otherTeacherID := insertBooking(teacherBID, weekStart.Add(24*time.Hour))

	deleted, err := repo.ReplaceWeeklyAvailabilityAndDeleteBookingsFrom(context.Background(), teacherAID, []shared.WeeklySlot{
		{DayOfWeek: 2, Start: shared.TimeHHMM("14:00"), End: shared.TimeHHMM("16:00")},
	}, weekStart)
	require.NoError(t, err)
	assert.Equal(t, 3, deleted)
	assert.Equal(t, 1, countTeacherSlots(t, db, teacherAID))

	rows, err := db.Query(`SELECT id FROM bookings ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	var remaining []int
	for rows.Next() {
		var id int
		require.NoError(t, rows.Scan(&id))
		remaining = append(remaining, id)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []int{lastWeekID, otherTeacherID}, remaining)
}

func TestPostgresRepoReplaceWeeklyAvailabilityAndDeleteBookingsFrom_InvalidSlotKeepsBookings(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	teacherID, branchID, subjectID := seedBookingParents(t, db)
	studentID := seedStudent(t, db, "Student")
	weekStart := time.Date(2026, 10, 18, 0, 0, 0, 0, shared.LoadLocation())
	start := weekStart.Add(24 * time.Hour)
	_, err := db.Exec(`
		INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, student_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		teacherID, branchID, subjectID, start, start.Add(time.Hour), studentID)
	require.NoError(t, err)
	require.NoError(t, repo.ReplaceWeeklyAvailability(context.Background(), teacherID, []shared.WeeklySlot{
		{DayOfWeek: 1, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")},
	}))

	_, err = repo.ReplaceWeeklyAvailabilityAndDeleteBookingsFrom(context.Background(), teacherID, []shared.WeeklySlot{
		{DayOfWeek: 9, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")},
	}, weekStart)
	require.Error(t, err)

	var bookings int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE teacher_id = $1`, teacherID).Scan(&bookings))
	assert.Equal(t, 1, bookings)
	assert.Equal(t, 1, countTeacherSlots(t, db, teacherID))
}

func TestPostgresRepoReplaceWeeklyAvailabilityAndDeleteBookingsFrom_TeacherNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := repo.ReplaceWeeklyAvailabilityAndDeleteBookingsFrom(context.Background(), 99, []shared.WeeklySlot{
		{DayOfWeek: 1, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")},
	}, time.Now())

	var notFound *shared.NotFoundError
	require.ErrorAs(t, err, &notFound)
}

func TestPostgresRepoGetTeachersBySubject_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, gender, status) VALUES (1, 'Alice', 'alice@test.com', 'female', 'active')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teachers (id, name, email, gender, status) VALUES (2, 'Bob', 'bob@test.com', 'male', 'active')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teachers (id, name, gender, status) VALUES (3, 'Inactive Teacher', 'lgbtq+', 'deactivated')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO subjects (id, name) VALUES (1, 'Math')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO subjects (id, name) VALUES (2, 'Physics')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teacher_subjects (teacher_id, subject_id) VALUES (1, 1), (2, 1), (3, 1), (1, 2)`)
	require.NoError(t, err)

	teachers, err := repo.GetTeachersBySubject(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, teachers, 2)
	assert.Equal(t, "Alice", teachers[0].Name)
	assert.Equal(t, "Bob", teachers[1].Name)
}

func TestPostgresRepoGetAllAvailability_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, gender, status) VALUES (1, 'Alice', 'alice@test.com', 'female', 'active')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teachers (id, name, email, gender, status) VALUES (2, 'Bob', 'bob@test.com', 'male', 'active')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teacher_availability (teacher_id, day_of_week, start_time, end_time)
		VALUES (1, 0, '09:00', '12:00'), (1, 2, '10:00', '14:00')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teacher_availability (teacher_id, day_of_week, start_time, end_time)
		VALUES (2, 1, '10:00', '15:00')`)
	require.NoError(t, err)

	result, err := repo.GetAllAvailability(context.Background())
	require.NoError(t, err)
	require.Len(t, result, 2)

	assert.Equal(t, "Alice", result[0].Teacher.Name)
	assert.Equal(t, "female", result[0].Teacher.Gender)
	assert.Len(t, result[0].Weekly, 2)
	assert.Equal(t, 0, result[0].Weekly[0].DayOfWeek)
	assert.Equal(t, shared.TimeHHMM("09:00"), result[0].Weekly[0].Start)
	assert.Equal(t, 2, result[0].Weekly[1].DayOfWeek)
	assert.Equal(t, shared.TimeHHMM("10:00"), result[0].Weekly[1].Start)

	assert.Equal(t, "Bob", result[1].Teacher.Name)
	assert.Equal(t, "male", result[1].Teacher.Gender)
	assert.Len(t, result[1].Weekly, 1)
	assert.Equal(t, 1, result[1].Weekly[0].DayOfWeek)
}

func TestPostgresRepoSaveRawSubmission(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, gender, status) VALUES (1, 'Test Teacher', 'test@teacher.com', 'male', 'active')`)
	require.NoError(t, err)

	rawJSON, _ := json.Marshal(map[string]interface{}{
		"teacher_id": 1,
		"weekly":     []string{"slot1"},
	})
	err = repo.SaveRawSubmission(context.Background(), 1, rawJSON)
	require.NoError(t, err)

	var count int
	err = db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM form_submission WHERE teacher_id = 1`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCheckViolationError_MatchingCode_ReturnsValidationError(t *testing.T) {
	err := &pgconn.PgError{Code: pgCheckViolation}

	result := checkViolationError(err, "invalid gender")

	var valErr *shared.ValidationError
	require.ErrorAs(t, result, &valErr)
	assert.Equal(t, "invalid gender", valErr.Msg)
}

func TestCheckViolationError_DifferentPgCode_ReturnsNil(t *testing.T) {
	err := &pgconn.PgError{Code: pgExclusionViolation}

	result := checkViolationError(err, "invalid gender")

	assert.Nil(t, result)
}

func TestCheckViolationError_NonPgError_ReturnsNil(t *testing.T) {
	result := checkViolationError(errors.New("connection refused"), "invalid gender")

	assert.Nil(t, result)
}
