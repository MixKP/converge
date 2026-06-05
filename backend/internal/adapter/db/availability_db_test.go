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
