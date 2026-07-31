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

func TestCommuteConfigRepoGetCommuteMinutes_DefaultSeeded(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCommuteConfigRepository(db)

	minutes, err := repo.GetCommuteMinutes(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 30, minutes)
}

func TestCommuteConfigRepoSetCommuteMinutes_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCommuteConfigRepository(db)

	require.NoError(t, repo.SetCommuteMinutes(context.Background(), 45))

	minutes, err := repo.GetCommuteMinutes(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 45, minutes)
}

func TestCommuteConfigRepoSetCommuteMinutes_Zero_Allowed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCommuteConfigRepository(db)

	require.NoError(t, repo.SetCommuteMinutes(context.Background(), 0))

	minutes, err := repo.GetCommuteMinutes(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, minutes)
}

func TestCommuteConfigRepoSetCommuteMinutes_Negative_ReturnsValidationError(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCommuteConfigRepository(db)

	err := repo.SetCommuteMinutes(context.Background(), -1)
	require.Error(t, err)
	var valErr *shared.ValidationError
	assert.True(t, errors.As(err, &valErr))
}
