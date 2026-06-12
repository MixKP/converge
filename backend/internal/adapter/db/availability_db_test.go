//go:build integration

package db

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepoGetActiveTeachers(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, status) VALUES (1, 'Test Teacher', 'test@teacher.com', 'active')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teachers (id, name, status) VALUES (2, 'Inactive Teacher', 'deactivated')`)
	require.NoError(t, err)

	teachers, err := repo.GetActiveTeachers(context.Background())
	require.NoError(t, err)
	assert.Len(t, teachers, 1)
	assert.Equal(t, "Test Teacher", teachers[0].Name)
}

func TestPostgresRepoReplaceWeeklyAvailability(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, status) VALUES (1, 'Test Teacher', 'test@teacher.com', 'active')`)
	require.NoError(t, err)

	slots := []shared.WeeklySlot{
		{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")},
		{DayOfWeek: 0, Start: shared.TimeHHMM("11:00"), End: shared.TimeHHMM("12:00")},
	}
	err = repo.ReplaceWeeklyAvailability(context.Background(), 1, slots)
	require.NoError(t, err)

	var count int
	err = db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM teacher_availability WHERE teacher_id = 1`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	replaced := []shared.WeeklySlot{
		{DayOfWeek: 2, Start: shared.TimeHHMM("14:00"), End: shared.TimeHHMM("16:00")},
	}
	err = repo.ReplaceWeeklyAvailability(context.Background(), 1, replaced)
	require.NoError(t, err)

	err = db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM teacher_availability WHERE teacher_id = 1`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPostgresRepoGetTeachersBySubject_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, status) VALUES (1, 'Alice', 'alice@test.com', 'active')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teachers (id, name, email, status) VALUES (2, 'Bob', 'bob@test.com', 'active')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teachers (id, name, status) VALUES (3, 'Inactive Teacher', 'deactivated')`)
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

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, status) VALUES (1, 'Alice', 'alice@test.com', 'active')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teachers (id, name, email, status) VALUES (2, 'Bob', 'bob@test.com', 'active')`)
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
	assert.Len(t, result[0].Weekly, 2)
	assert.Equal(t, 0, result[0].Weekly[0].DayOfWeek)
	assert.Equal(t, shared.TimeHHMM("09:00"), result[0].Weekly[0].Start)
	assert.Equal(t, 2, result[0].Weekly[1].DayOfWeek)
	assert.Equal(t, shared.TimeHHMM("10:00"), result[0].Weekly[1].Start)

	assert.Equal(t, "Bob", result[1].Teacher.Name)
	assert.Len(t, result[1].Weekly, 1)
	assert.Equal(t, 1, result[1].Weekly[0].DayOfWeek)
}

func TestPostgresRepoSaveRawSubmission(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO teachers (id, name, email, status) VALUES (1, 'Test Teacher', 'test@teacher.com', 'active')`)
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
