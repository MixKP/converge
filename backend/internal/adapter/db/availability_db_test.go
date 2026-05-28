//go:build integration

package db

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *PostgresRepo {
	t.Helper()

	database, err := InitDB()
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

	require.NoError(t, AutoMigrate(database))

	database.Exec(`DELETE FROM teacher_availability`)
	database.Exec(`DELETE FROM form_submission`)
	database.Exec(`DELETE FROM teachers`)

	_, err = database.Exec(`INSERT INTO teachers (id, name, status) VALUES (1, 'Test Teacher', 'active')`)
	require.NoError(t, err)
	_, err = database.Exec(`INSERT INTO teachers (id, name, status) VALUES (2, 'Inactive Teacher', 'deactivated')`)
	require.NoError(t, err)

	return NewPostgresRepo(database)
}

func TestPostgresRepoGetActiveTeachers(t *testing.T) {
	repo := setupTestDB(t)

	teachers, err := repo.GetActiveTeachers(context.Background())
	require.NoError(t, err)
	assert.Len(t, teachers, 1)
	assert.Equal(t, "Test Teacher", teachers[0].Name)
}

func TestPostgresRepoReplaceWeeklyAvailability(t *testing.T) {
	repo := setupTestDB(t)

	slots := []models.WeeklySlot{
		{DayOfWeek: 1, Start: "09:00", End: "10:00"},
		{DayOfWeek: 1, Start: "11:00", End: "12:00"},
	}
	err := repo.ReplaceWeeklyAvailability(context.Background(), 1, slots)
	require.NoError(t, err)

	var count int
	err = repo.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM teacher_availability WHERE teacher_id = 1`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	replaced := []models.WeeklySlot{
		{DayOfWeek: 3, Start: "14:00", End: "16:00"},
	}
	err = repo.ReplaceWeeklyAvailability(context.Background(), 1, replaced)
	require.NoError(t, err)

	err = repo.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM teacher_availability WHERE teacher_id = 1`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPostgresRepoSaveRawSubmission(t *testing.T) {
	repo := setupTestDB(t)

	rawJSON, _ := json.Marshal(map[string]interface{}{
		"teacher_id": 1,
		"weekly":     []string{"slot1"},
	})
	err := repo.SaveRawSubmission(context.Background(), 1, rawJSON)
	require.NoError(t, err)

	var count int
	err = repo.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM form_submission WHERE teacher_id = 1`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
