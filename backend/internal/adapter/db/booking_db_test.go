//go:build integration

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/scheduling"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const postgresImage = "postgres:16-alpine"

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	ctx := context.Background()
	container, err := postgres.Run(ctx,
		postgresImage,
		postgres.WithDatabase("converge_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	database, err := InitDBWithConnString(connStr)
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

	require.NoError(t, AutoMigrate(database))
	return database
}

func seedBookingParents(t *testing.T, db *sql.DB) (teacherID, branchID, subjectID int) {
	t.Helper()
	require.NoError(t, db.QueryRow(`INSERT INTO teachers (id, name, email, gender, status) VALUES (1, 'Test Teacher', 'test@test.com', 'male', 'active') RETURNING id`).Scan(&teacherID))
	require.NoError(t, db.QueryRow(`INSERT INTO branches (id, name) VALUES (1, 'Test Branch') RETURNING id`).Scan(&branchID))
	require.NoError(t, db.QueryRow(`INSERT INTO subjects (id, name) VALUES (1, 'Test Subject') RETURNING id`).Scan(&subjectID))
	return
}

func seedStudent(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	var id int
	require.NoError(t, db.QueryRow(
		`INSERT INTO users (name, password_hash, role) VALUES ($1, 'x', 'student') RETURNING id`, name,
	).Scan(&id))
	return id
}

// seedCappedBranchWithTwoTeachers seeds a branch with the given capacity and
// two active teachers of different genders, for tests that need two
// independent teachers competing for the same branch/time slot.
func seedCappedBranchWithTwoTeachers(t *testing.T, db *sql.DB, capacity int) (teacherAID, teacherBID, branchID, subjectID int) {
	t.Helper()
	require.NoError(t, db.QueryRow(`INSERT INTO teachers (id, name, email, gender, status) VALUES (1, 'Teacher A', 'a@test.com', 'male', 'active') RETURNING id`).Scan(&teacherAID))
	require.NoError(t, db.QueryRow(`INSERT INTO teachers (id, name, email, gender, status) VALUES (2, 'Teacher B', 'b@test.com', 'female', 'active') RETURNING id`).Scan(&teacherBID))
	require.NoError(t, db.QueryRow(`INSERT INTO branches (id, name, capacity) VALUES (1, 'Capped Branch', $1) RETURNING id`, capacity).Scan(&branchID))
	require.NoError(t, db.QueryRow(`INSERT INTO subjects (id, name) VALUES (1, 'Test Subject') RETURNING id`).Scan(&subjectID))
	return
}

func TestBookingRepoCreateBooking(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)
	studentID := seedStudent(t, db, "John Doe")

	booking, err := repo.CreateBooking(context.Background(), scheduling.ConfirmBookingRequest{
		TeacherID: teacherID,
		BranchID:  branchID,
		SubjectID: subjectID,
		StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StudentID: studentID,
	})
	require.NoError(t, err)
	assert.Equal(t, teacherID, booking.TeacherID)
	assert.Equal(t, branchID, booking.BranchID)
	assert.Equal(t, subjectID, booking.SubjectID)
	assert.Equal(t, studentID, booking.StudentID)
	assert.Equal(t, "John Doe", booking.StudentName)
	assert.NotZero(t, booking.ID)
	assert.NotZero(t, booking.CreatedAt)
}

func TestBookingRepoCreateBooking_RejectsOverlap(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)
	studentID := seedStudent(t, db, "John Doe")

	_, err := repo.CreateBooking(context.Background(), scheduling.ConfirmBookingRequest{
		TeacherID: teacherID, BranchID: branchID, SubjectID: subjectID,
		StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StudentID: studentID,
	})
	require.NoError(t, err)

	_, err = repo.CreateBooking(context.Background(), scheduling.ConfirmBookingRequest{
		TeacherID: teacherID, BranchID: branchID, SubjectID: subjectID,
		StartTime: time.Date(2026, 6, 1, 9, 30, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC),
		StudentID: studentID,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, scheduling.ErrBookingConflict)
}

func TestBookingRepoCreateBooking_ExactCommuteGapAccepted(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchAID, subjectID := seedBookingParents(t, db)
	studentID := seedStudent(t, db, "Commute Student")

	var branchBID int
	require.NoError(t, db.QueryRow(`INSERT INTO branches (id, name) VALUES (2, 'Second Branch') RETURNING id`).Scan(&branchBID))

	_, err := repo.CreateBooking(context.Background(), scheduling.ConfirmBookingRequest{
		TeacherID: teacherID, BranchID: branchAID, SubjectID: subjectID,
		StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StudentID: studentID,
	})
	require.NoError(t, err)

	_, err = repo.CreateBooking(context.Background(), scheduling.ConfirmBookingRequest{
		TeacherID: teacherID, BranchID: branchBID, SubjectID: subjectID,
		StartTime: time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 11, 30, 0, 0, time.UTC),
		StudentID: studentID,
	})
	require.NoError(t, err, "an exact 30-minute commute gap should be accepted")
}

func TestBookingRepoCreateBooking_ConcurrentCommuteConflict(t *testing.T) {
	db := setupTestDB(t)
	db.SetMaxOpenConns(10)
	repo := NewBookingRepository(db)
	teacherID, branchAID, subjectID := seedBookingParents(t, db)
	studentID := seedStudent(t, db, "Commute Student")

	var branchBID int
	require.NoError(t, db.QueryRow(`INSERT INTO branches (id, name) VALUES (2, 'Second Branch') RETURNING id`).Scan(&branchBID))

	dayTwo := time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)
	requests := []scheduling.ConfirmBookingRequest{
		{
			TeacherID: teacherID, BranchID: branchAID, SubjectID: subjectID,
			StartTime: dayTwo, EndTime: dayTwo.Add(time.Hour), StudentID: studentID,
		},
		{
			TeacherID: teacherID, BranchID: branchBID, SubjectID: subjectID,
			StartTime: dayTwo.Add(75 * time.Minute), EndTime: dayTwo.Add(135 * time.Minute), StudentID: studentID,
		},
	}

	var wg sync.WaitGroup
	errs := make([]error, len(requests))
	for i := range requests {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = repo.CreateBooking(context.Background(), requests[i])
		}(i)
	}
	wg.Wait()

	successCount, commuteErrCount := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, scheduling.ErrCommuteConflict):
			commuteErrCount++
		}
	}
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 1, commuteErrCount)

	var count int
	require.NoError(t, db.QueryRow(`
		SELECT COUNT(*) FROM bookings
		WHERE teacher_id = $1 AND start_time >= $2 AND start_time < $3`,
		teacherID, dayTwo, dayTwo.Add(24*time.Hour),
	).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestBookingRepoCreateBooking_RejectsMissingStudent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)

	_, err := repo.CreateBooking(context.Background(), scheduling.ConfirmBookingRequest{
		TeacherID: teacherID, BranchID: branchID, SubjectID: subjectID,
		StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StudentID: 99999,
	})
	require.Error(t, err)
	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.EqualError(t, err, "student 99999 not found")
}

func TestBookingRepoCreateBooking_RejectsNonStudentUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)

	var parentID int
	require.NoError(t, db.QueryRow(
		`INSERT INTO users (name, password_hash, role) VALUES ('p', 'x', 'parent') RETURNING id`).Scan(&parentID))

	_, err := repo.CreateBooking(context.Background(), scheduling.ConfirmBookingRequest{
		TeacherID: teacherID, BranchID: branchID, SubjectID: subjectID,
		StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StudentID: parentID,
	})
	require.Error(t, err)
	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.EqualError(t, err, fmt.Sprintf("user %d is not a student", parentID))
}

func TestBookingRepoCreateBooking_BranchCapacityIsInformational(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherAID, teacherBID, branchID, subjectID := seedCappedBranchWithTwoTeachers(t, db, 1)
	studentID := seedStudent(t, db, "Capacity Student")

	start := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

	_, err := repo.CreateBooking(context.Background(), scheduling.ConfirmBookingRequest{
		TeacherID: teacherAID, BranchID: branchID, SubjectID: subjectID,
		StartTime: start, EndTime: end, StudentID: studentID,
	})
	require.NoError(t, err)

	_, err = repo.CreateBooking(context.Background(), scheduling.ConfirmBookingRequest{
		TeacherID: teacherBID, BranchID: branchID, SubjectID: subjectID,
		StartTime: start, EndTime: end, StudentID: studentID,
	})
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE branch_id = $1`, branchID).Scan(&count))
	assert.Equal(t, 2, count)
}

func seedTwoBookings(t *testing.T, db *sql.DB) (teacherID int) {
	t.Helper()
	teacherID, branchID, subjectID := seedBookingParents(t, db)
	studentID := seedStudent(t, db, "Student")
	for _, window := range [][2]string{{"2026-06-01T09:00:00Z", "2026-06-01T10:00:00Z"}, {"2026-06-01T11:00:00Z", "2026-06-01T12:00:00Z"}} {
		_, err := db.Exec(`
			INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, student_id)
			VALUES ($1, $2, $3, $4, $5, $6)`, teacherID, branchID, subjectID, window[0], window[1], studentID)
		require.NoError(t, err)
	}
	return teacherID
}

func TestBookingRepoFindConflictingBookings_Overlapping(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID := seedTwoBookings(t, db)

	conflicts, err := repo.FindConflictingBookings(context.Background(), teacherID,
		time.Date(2026, 6, 1, 9, 30, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Len(t, conflicts, 1)
}

func TestBookingRepoFindConflictingBookings_NoOverlap(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID := seedTwoBookings(t, db)

	noConflicts, err := repo.FindConflictingBookings(context.Background(), teacherID,
		time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 15, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Empty(t, noConflicts)
}

func TestBookingRepoDeleteBooking(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)
	studentID := seedStudent(t, db, "Student")

	var bookingID int
	require.NoError(t, db.QueryRow(`
		INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, student_id)
		VALUES ($1, $2, $3, '2026-06-01T09:00:00Z', '2026-06-01T10:00:00Z', $4)
		RETURNING id`, teacherID, branchID, subjectID, studentID).Scan(&bookingID))

	err := repo.DeleteBooking(context.Background(), bookingID)
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE id = $1`, bookingID).Scan(&count))
	assert.Equal(t, 0, count)
}

func TestBookingRepoDeleteBooking_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)

	err := repo.DeleteBooking(context.Background(), 999)
	require.Error(t, err)
	var notFound *shared.NotFoundError
	assert.ErrorAs(t, err, &notFound)
	assert.EqualError(t, err, "booking 999 not found")
}

func TestBookingRepoFindAllBookings(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)
	studentID := seedStudent(t, db, "Student")

	for i := range 3 {
		_, err := db.Exec(`
			INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, student_id)
			VALUES ($1, $2, $3, $4, $5, $6)`, teacherID, branchID, subjectID,
			fmt.Sprintf("2026-06-0%dT09:00:00Z", i+1),
			fmt.Sprintf("2026-06-0%dT10:00:00Z", i+1), studentID)
		require.NoError(t, err)
	}

	bookings, err := repo.FindAllBookings(context.Background())
	require.NoError(t, err)
	assert.Len(t, bookings, 3)
}

func seedStudentBookings(t *testing.T, db *sql.DB) (s1, s2 int) {
	t.Helper()
	teacherID, branchID, subjectID := seedBookingParents(t, db)
	s1 = seedStudent(t, db, "student-a")
	s2 = seedStudent(t, db, "student-b")
	insert := func(studentID, hour int) {
		_, err := db.Exec(`
			INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, student_id)
			VALUES ($1, $2, $3, $4, $5, $6)`, teacherID, branchID, subjectID,
			time.Date(2026, 6, 1, hour, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 1, hour+1, 0, 0, 0, time.UTC), studentID)
		require.NoError(t, err)
	}
	insert(s1, 9)
	insert(s1, 11)
	insert(s2, 13)
	return s1, s2
}

func TestBookingRepoFindBookingsByStudentIDs_OneStudent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	s1, _ := seedStudentBookings(t, db)

	own, err := repo.FindBookingsByStudentIDs(context.Background(), []int{s1})
	require.NoError(t, err)
	assert.Len(t, own, 2)
	assert.Equal(t, "student-a", own[0].StudentName)
	assert.Equal(t, "Test Teacher", own[0].TeacherName)
	assert.Equal(t, "Test Branch", own[0].BranchName)
	assert.Equal(t, "Test Subject", own[0].SubjectName)
}

func TestBookingRepoFindBookingsByStudentIDs_SeveralStudents(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	s1, s2 := seedStudentBookings(t, db)

	both, err := repo.FindBookingsByStudentIDs(context.Background(), []int{s1, s2})
	require.NoError(t, err)
	assert.Len(t, both, 3)
}

func TestBookingRepoFindBookingsByStudentIDs_EmptyList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	seedStudentBookings(t, db)

	none, err := repo.FindBookingsByStudentIDs(context.Background(), []int{})
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestBookingRepoFindExactMatch_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)

	_, err := db.Exec(`INSERT INTO teacher_subjects (teacher_id, subject_id) VALUES ($1, $2)`, teacherID, subjectID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO teacher_availability (teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, 0, '09:00', '17:00')`, teacherID)
	require.NoError(t, err)

	slot := shared.WeeklySlot{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")}
	match, err := repo.FindExactMatch(context.Background(), subjectID, branchID, slot, 60, shared.None[int](), "male")
	require.NoError(t, err)
	require.NotNil(t, match)
	assert.Equal(t, "Test Teacher", match.TeacherName)
	assert.Equal(t, teacherID, match.Booking.TeacherID)
	assert.NotZero(t, match.Booking.StartTime)
	assert.NotZero(t, match.Booking.EndTime)
}

func TestBookingRepoFindExactMatch_GenderMismatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)

	_, err := db.Exec(`INSERT INTO teacher_subjects (teacher_id, subject_id) VALUES ($1, $2)`, teacherID, subjectID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO teacher_availability (teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, 0, '09:00', '17:00')`, teacherID)
	require.NoError(t, err)

	slot := shared.WeeklySlot{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")}
	match, err := repo.FindExactMatch(context.Background(), subjectID, branchID, slot, 60, shared.None[int](), "female")
	require.NoError(t, err)
	assert.Nil(t, match, "teacher is male, requesting female should exclude the exact match")
}

func TestBookingRepoFindExactMatch_NoMatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)

	_, err := db.Exec(`INSERT INTO teacher_subjects (teacher_id, subject_id) VALUES ($1, $2)`, teacherID, subjectID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO teacher_availability (teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, 1, '09:00', '17:00')`, teacherID) // day=1 (Monday), but slot is day=0
	require.NoError(t, err)

	slot := shared.WeeklySlot{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")}
	match, err := repo.FindExactMatch(context.Background(), subjectID, branchID, slot, 60, shared.None[int](), "male")
	require.NoError(t, err)
	assert.Nil(t, match)
}

func TestBookingRepoFindExactMatch_DeactivatedTeacher(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)

	_, err := db.Exec(`UPDATE teachers SET status = 'deactivated' WHERE id = $1`, teacherID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO teacher_subjects (teacher_id, subject_id) VALUES ($1, $2)`, teacherID, subjectID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO teacher_availability (teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, 0, '09:00', '17:00')`, teacherID)
	require.NoError(t, err)

	slot := shared.WeeklySlot{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")}
	match, err := repo.FindExactMatch(context.Background(), subjectID, branchID, slot, 60, shared.None[int](), "male")
	require.NoError(t, err)
	assert.Nil(t, match)
}

func TestBookingRepoFindExactMatch_ConflictExcludes(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)

	_, err := db.Exec(`INSERT INTO teacher_subjects (teacher_id, subject_id) VALUES ($1, $2)`, teacherID, subjectID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO teacher_availability (teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, 0, '09:00', '17:00')`, teacherID)
	require.NoError(t, err)

	// Insert a booking that overlaps with the slot
	studentID := seedStudent(t, db, "Student")
	loc := shared.LoadLocation()
	anchor := shared.AnchorDateForDay(0, loc)
	conflictStart := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 9, 30, 0, 0, loc)
	conflictEnd := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 10, 30, 0, 0, loc)
	_, err = db.Exec(`INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, student_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		teacherID, branchID, subjectID, conflictStart, conflictEnd, studentID)
	require.NoError(t, err)

	slot := shared.WeeklySlot{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")}
	match, err := repo.FindExactMatch(context.Background(), subjectID, branchID, slot, 60, shared.None[int](), "male")
	require.NoError(t, err)
	assert.Nil(t, match)
}

func TestBookingRepoCreateBooking_RejectsDeactivatedBranch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)
	studentID := seedStudent(t, db, "Jane Doe")

	_, err := db.Exec(`UPDATE branches SET status = 'deactivated' WHERE id = $1`, branchID)
	require.NoError(t, err)

	_, err = repo.CreateBooking(context.Background(), scheduling.ConfirmBookingRequest{
		TeacherID: teacherID,
		BranchID:  branchID,
		SubjectID: subjectID,
		StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StudentID: studentID,
	})
	require.Error(t, err)
	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.EqualError(t, err, fmt.Sprintf("branch %d is deactivated", branchID))
}

func TestBookingRepoCreateBooking_AcceptsReactivatedBranch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)
	studentID := seedStudent(t, db, "Jane Doe")

	_, err := db.Exec(`UPDATE branches SET status = 'deactivated' WHERE id = $1`, branchID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE branches SET status = 'active' WHERE id = $1`, branchID)
	require.NoError(t, err)

	booking, err := repo.CreateBooking(context.Background(), scheduling.ConfirmBookingRequest{
		TeacherID: teacherID,
		BranchID:  branchID,
		SubjectID: subjectID,
		StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StudentID: studentID,
	})
	require.NoError(t, err)
	assert.Equal(t, branchID, booking.BranchID)
}

type rescheduleFixture struct {
	repo    *BookingRepo
	db      *sql.DB
	now     time.Time
	moveID  int
	pastID  int
	at      func(day, hour int) time.Time
	timesOf func(id int) (time.Time, time.Time)
}

// setupRescheduleFixture freezes "now" at Thu 22 Oct 2026 12:00 Bangkok.
// Teacher A is free Mondays 09:00-17:00 and Tuesdays 09:00-12:00. Student S has
// an upcoming Monday 09:00 class (the one moved) and a past Tuesday class.
// Another student holds Monday 13:00 with the same teacher, and the teacher
// teaches at a second branch on Monday 11:00-12:00 (commute is 30 minutes).
func setupRescheduleFixture(t *testing.T) rescheduleFixture {
	t.Helper()
	db := setupTestDB(t)
	teacherA, _, branchID, subjectID := seedCappedBranchWithTwoTeachers(t, db, -1)
	var secondBranch int
	require.NoError(t, db.QueryRow(`INSERT INTO branches (id, name) VALUES (2, 'Second Branch') RETURNING id`).Scan(&secondBranch))
	student := seedStudent(t, db, "S")
	other := seedStudent(t, db, "S2")
	_, err := db.Exec(`
		INSERT INTO teacher_availability (teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, 1, '09:00', '17:00'), ($1, 2, '09:00', '12:00')`, teacherA)
	require.NoError(t, err)

	loc := shared.LoadLocation()
	at := func(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, loc) }
	insert := func(studentID, branch int, start time.Time) int {
		var id int
		require.NoError(t, db.QueryRow(`
			INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, student_id)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			teacherA, branch, subjectID, start, start.Add(time.Hour), studentID).Scan(&id))
		return id
	}

	f := rescheduleFixture{
		repo:   NewBookingRepository(db),
		db:     db,
		now:    at(22, 12),
		moveID: insert(student, branchID, at(26, 9)),
		pastID: insert(student, branchID, at(20, 9)),
		at:     at,
	}
	insert(other, branchID, at(26, 13))
	insert(student, secondBranch, at(26, 11))
	f.timesOf = func(id int) (time.Time, time.Time) {
		var start, end time.Time
		require.NoError(t, db.QueryRow(`SELECT start_time, end_time FROM bookings WHERE id = $1`, id).Scan(&start, &end))
		return start.In(loc), end.In(loc)
	}
	return f
}

func TestBookingRepoRescheduleBooking_MovesWithinAvailability(t *testing.T) {
	f := setupRescheduleFixture(t)

	moved, err := f.repo.RescheduleBooking(context.Background(), f.moveID, f.at(27, 10), f.at(27, 11), f.now)

	require.NoError(t, err)
	assert.Equal(t, f.moveID, moved.ID, "the booking keeps its id")
	assert.True(t, moved.StartTime.Equal(f.at(27, 10)))
	assert.Equal(t, "S", moved.StudentName)
	assert.Equal(t, "Teacher A", moved.TeacherName, "the response carries the display names like the list does")
	start, end := f.timesOf(f.moveID)
	assert.Equal(t, f.at(27, 10), start)
	assert.Equal(t, f.at(27, 11), end)
}

func TestBookingRepoRescheduleBooking_Rejects(t *testing.T) {
	tests := []struct {
		name       string
		id         func(f rescheduleFixture) int
		start, end func(f rescheduleFixture) time.Time
		want       error
	}{
		{"a class that already happened",
			func(f rescheduleFixture) int { return f.pastID },
			func(f rescheduleFixture) time.Time { return f.at(26, 15) },
			func(f rescheduleFixture) time.Time { return f.at(26, 16) },
			scheduling.ErrBookingStarted},
		{"a new time in the past",
			func(f rescheduleFixture) int { return f.moveID },
			func(f rescheduleFixture) time.Time { return f.at(19, 9) },
			func(f rescheduleFixture) time.Time { return f.at(19, 10) },
			scheduling.ErrSessionInPast},
		{"past the end of the availability window",
			func(f rescheduleFixture) int { return f.moveID },
			func(f rescheduleFixture) time.Time { return f.at(26, 16) },
			func(f rescheduleFixture) time.Time { return f.at(26, 18) },
			scheduling.ErrOutsideAvailability},
		{"a day the teacher is not free",
			func(f rescheduleFixture) int { return f.moveID },
			func(f rescheduleFixture) time.Time { return f.at(28, 9) },
			func(f rescheduleFixture) time.Time { return f.at(28, 10) },
			scheduling.ErrOutsideAvailability},
		{"on top of another student's class",
			func(f rescheduleFixture) int { return f.moveID },
			func(f rescheduleFixture) time.Time { return f.at(26, 13) },
			func(f rescheduleFixture) time.Time { return f.at(26, 14) },
			scheduling.ErrBookingConflict},
		{"right after a class at another branch",
			func(f rescheduleFixture) int { return f.moveID },
			func(f rescheduleFixture) time.Time { return f.at(26, 12) },
			func(f rescheduleFixture) time.Time { return f.at(26, 13) },
			scheduling.ErrCommuteConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := setupRescheduleFixture(t)
			id := tt.id(f)
			beforeStart, beforeEnd := f.timesOf(id)

			_, err := f.repo.RescheduleBooking(context.Background(), id, tt.start(f), tt.end(f), f.now)

			require.ErrorIs(t, err, tt.want)
			afterStart, afterEnd := f.timesOf(id)
			assert.Equal(t, beforeStart, afterStart, "the booking is left where it was")
			assert.Equal(t, beforeEnd, afterEnd)
		})
	}
}

func TestBookingRepoRescheduleBooking_NotFound(t *testing.T) {
	f := setupRescheduleFixture(t)

	_, err := f.repo.RescheduleBooking(context.Background(), 9999, f.at(26, 15), f.at(26, 16), f.now)

	var notFound *shared.NotFoundError
	require.ErrorAs(t, err, &notFound)
}
