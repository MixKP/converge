//go:build integration

package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchRepoGetBranches(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBranchRepository(db)

	_, err := db.Exec(`INSERT INTO branches (id, name, capacity) VALUES (1, 'Main Campus', 30), (2, 'Downtown', 15)`)
	require.NoError(t, err)

	branches, err := repo.GetBranches(context.Background())
	require.NoError(t, err)
	assert.Len(t, branches, 2)
	assert.Equal(t, 1, branches[0].ID)
	assert.Equal(t, "Main Campus", branches[0].Name)
	assert.Equal(t, 30, branches[0].Capacity)
	assert.Equal(t, 2, branches[1].ID)
	assert.Equal(t, "Downtown", branches[1].Name)
	assert.Equal(t, 15, branches[1].Capacity)
}

func TestBranchRepoGetBranchByID_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBranchRepository(db)

	_, err := db.Exec(`INSERT INTO branches (id, name, capacity) VALUES (1, 'Main Campus', 30)`)
	require.NoError(t, err)

	b, err := repo.GetBranchByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1, b.ID)
	assert.Equal(t, "Main Campus", b.Name)
	assert.Equal(t, 30, b.Capacity)
}

func TestBranchRepoGetBranchByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBranchRepository(db)

	_, err := repo.GetBranchByID(context.Background(), 99)
	require.Error(t, err)
	var notFoundErr *shared.NotFoundError
	assert.True(t, errors.As(err, &notFoundErr))
}

func TestBranchRepoCountOverlappingBookings(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBranchRepository(db)

	require.NoError(t, db.QueryRow(`INSERT INTO branches (id, name, capacity) VALUES (1, 'Main Campus', 3) RETURNING id`).Scan(new(int)))
	require.NoError(t, db.QueryRow(`INSERT INTO subjects (id, name) VALUES (1, 'Test Subject') RETURNING id`).Scan(new(int)))
	require.NoError(t, db.QueryRow(`INSERT INTO teachers (id, name, email, gender, status) VALUES (1, 'Teacher A', 'a@test.com', 'male', 'active') RETURNING id`).Scan(new(int)))
	require.NoError(t, db.QueryRow(`INSERT INTO teachers (id, name, email, gender, status) VALUES (2, 'Teacher B', 'b@test.com', 'female', 'active') RETURNING id`).Scan(new(int)))

	start := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

	_, err := db.Exec(`
		INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, client_name)
		VALUES (1, 1, 1, $1, $2, 'John Doe')`, start, end)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, client_name)
		VALUES (2, 1, 1, $1, $2, 'Jane Doe')`, start.Add(30*time.Minute), end.Add(30*time.Minute))
	require.NoError(t, err)

	count, err := repo.CountOverlappingBookings(context.Background(), 1, start, end)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	outsideCount, err := repo.CountOverlappingBookings(context.Background(), 1, end.Add(time.Hour), end.Add(2*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, outsideCount)
}
