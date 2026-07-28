//go:build integration

package db

import (
	"context"
	"errors"
	"testing"

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

func TestBranchRepoSetCapacity_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBranchRepository(db)

	_, err := db.Exec(`INSERT INTO branches (id, name, capacity) VALUES (1, 'Main Campus', 10)`)
	require.NoError(t, err)

	require.NoError(t, repo.SetCapacity(context.Background(), 1, 25))

	b, err := repo.GetBranchByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 25, b.Capacity)
}

func TestBranchRepoSetCapacity_Zero_Allowed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBranchRepository(db)

	_, err := db.Exec(`INSERT INTO branches (id, name, capacity) VALUES (1, 'Main Campus', 10)`)
	require.NoError(t, err)

	require.NoError(t, repo.SetCapacity(context.Background(), 1, 0))

	b, err := repo.GetBranchByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 0, b.Capacity)
}

func TestBranchRepoSetCapacity_Negative_ReturnsValidationError(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBranchRepository(db)

	_, err := db.Exec(`INSERT INTO branches (id, name, capacity) VALUES (1, 'Main Campus', 10)`)
	require.NoError(t, err)

	err = repo.SetCapacity(context.Background(), 1, -1)
	require.Error(t, err)
	var valErr *shared.ValidationError
	assert.True(t, errors.As(err, &valErr))
}

func TestBranchRepoSetCapacity_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBranchRepository(db)

	err := repo.SetCapacity(context.Background(), 99, 10)
	require.Error(t, err)
	var notFoundErr *shared.NotFoundError
	assert.True(t, errors.As(err, &notFoundErr))
}
