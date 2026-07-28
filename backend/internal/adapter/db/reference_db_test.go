//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepoGetSubjects(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPostgresRepo(db)

	_, err := db.Exec(`INSERT INTO subjects (id, name) VALUES (1, 'Mathematics'), (2, 'Physics')`)
	require.NoError(t, err)

	subjects, err := repo.GetSubjects(context.Background())
	require.NoError(t, err)
	assert.Len(t, subjects, 2)
	assert.Equal(t, 1, subjects[0].ID)
	assert.Equal(t, "Mathematics", subjects[0].Name)
	assert.Equal(t, 2, subjects[1].ID)
	assert.Equal(t, "Physics", subjects[1].Name)
}
