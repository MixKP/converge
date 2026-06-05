//go:build integration

package db

import (
	"context"
	"database/sql"
	"fmt"
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
	require.NoError(t, db.QueryRow(`INSERT INTO teachers (id, name, email, status) VALUES (1, 'Test Teacher', 'test@test.com', 'active') RETURNING id`).Scan(&teacherID))
	require.NoError(t, db.QueryRow(`INSERT INTO branches (id, name) VALUES (1, 'Test Branch') RETURNING id`).Scan(&branchID))
	require.NoError(t, db.QueryRow(`INSERT INTO subjects (id, name) VALUES (1, 'Test Subject') RETURNING id`).Scan(&subjectID))
	return
}

func TestBookingRepoCreateBooking(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)

	start := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	req := scheduling.ConfirmBookingRequest{
		TeacherID:  teacherID,
		BranchID:   branchID,
		SubjectID:  subjectID,
		StartTime:  start,
		EndTime:    end,
		ClientName: "John Doe",
	}

	booking, err := repo.CreateBooking(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, teacherID, booking.TeacherID)
	assert.Equal(t, branchID, booking.BranchID)
	assert.Equal(t, subjectID, booking.SubjectID)
	assert.Equal(t, "John Doe", booking.ClientName)
	assert.NotZero(t, booking.ID)
	assert.NotZero(t, booking.CreatedAt)

	overlap := scheduling.ConfirmBookingRequest{
		TeacherID:  teacherID,
		BranchID:   branchID,
		SubjectID:  subjectID,
		StartTime:  start.Add(30 * time.Minute),
		EndTime:    end.Add(30 * time.Minute),
		ClientName: "Jane Doe",
	}
	_, err = repo.CreateBooking(context.Background(), overlap)
	require.Error(t, err)
	assert.ErrorIs(t, err, scheduling.ErrBookingConflict)
}

func TestBookingRepoFindConflictingBookings(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)

	_, err := db.Exec(`
		INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, client_name)
		VALUES ($1, $2, $3, '2026-06-01T09:00:00Z', '2026-06-01T10:00:00Z', 'Booking 1')`, teacherID, branchID, subjectID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, client_name)
		VALUES ($1, $2, $3, '2026-06-01T11:00:00Z', '2026-06-01T12:00:00Z', 'Booking 2')`, teacherID, branchID, subjectID)
	require.NoError(t, err)

	conflicts, err := repo.FindConflictingBookings(context.Background(), teacherID,
		time.Date(2026, 6, 1, 9, 30, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Len(t, conflicts, 1)

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

	var bookingID int
	require.NoError(t, db.QueryRow(`
		INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, client_name)
		VALUES ($1, $2, $3, '2026-06-01T09:00:00Z', '2026-06-01T10:00:00Z', 'John')
		RETURNING id`, teacherID, branchID, subjectID).Scan(&bookingID))

	err := repo.DeleteBooking(context.Background(), bookingID)
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE id = $1`, bookingID).Scan(&count))
	assert.Equal(t, 0, count)

	err = repo.DeleteBooking(context.Background(), 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestBookingRepoFindAllBookings(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, branchID, subjectID := seedBookingParents(t, db)

	for i := range 3 {
		_, err := db.Exec(`
			INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, client_name)
			VALUES ($1, $2, $3, $4, $5, $6)`, teacherID, branchID, subjectID,
			fmt.Sprintf("2026-06-0%dT09:00:00Z", i+1),
			fmt.Sprintf("2026-06-0%dT10:00:00Z", i+1),
			fmt.Sprintf("Booking %d", i+1))
		require.NoError(t, err)
	}

	bookings, err := repo.FindAllBookings(context.Background())
	require.NoError(t, err)
	assert.Len(t, bookings, 3)
}

func TestBookingRepoFindTeacherAvailability(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBookingRepository(db)
	teacherID, _, _ := seedBookingParents(t, db)

	slots, err := repo.FindTeacherAvailability(context.Background(), teacherID)
	require.NoError(t, err)
	assert.Empty(t, slots)

	avail := []shared.WeeklySlot{
		{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")},
	}
	availRepo := NewPostgresRepo(db)
	require.NoError(t, availRepo.ReplaceWeeklyAvailability(context.Background(), teacherID, avail))

	slots, err = repo.FindTeacherAvailability(context.Background(), teacherID)
	require.NoError(t, err)
	assert.Len(t, slots, 1)
	assert.Equal(t, "09:00", string(slots[0].Start))
}
