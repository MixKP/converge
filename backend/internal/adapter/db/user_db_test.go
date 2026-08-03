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

func TestUserRepoCreateUser_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	u, err := repo.CreateUser(context.Background(), "alice", "hashed-secret", "student")
	require.NoError(t, err)
	assert.NotZero(t, u.ID)
	assert.Equal(t, "alice", u.Name)
	assert.Equal(t, "student", u.Role)

	var storedHash string
	require.NoError(t, db.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, u.ID).Scan(&storedHash))
	assert.Equal(t, "hashed-secret", storedHash)
}

func TestUserRepoCreateUser_DuplicateName_ReturnsConflict(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	_, err := repo.CreateUser(context.Background(), "alice", "hash-1", "student")
	require.NoError(t, err)

	_, err = repo.CreateUser(context.Background(), "alice", "hash-2", "student")
	require.Error(t, err)
	var confErr *shared.ConflictError
	assert.True(t, errors.As(err, &confErr))
}

func TestUserRepoGetCredentialByName_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	created, err := repo.CreateUser(context.Background(), "alice", "hashed-secret", "parent")
	require.NoError(t, err)

	id, hash, role, err := repo.GetCredentialByName(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, created.ID, id)
	assert.Equal(t, "hashed-secret", hash)
	assert.Equal(t, "parent", role)
}

func TestUserRepoGetCredentialByName_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	_, _, _, err := repo.GetCredentialByName(context.Background(), "ghost")
	require.Error(t, err)
	var notFoundErr *shared.NotFoundError
	assert.True(t, errors.As(err, &notFoundErr))
}

func TestUserRepoCreateParent_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	s1 := seedStudent(t, db, "student-a")
	s2 := seedStudent(t, db, "student-b")

	p, err := repo.CreateParent(context.Background(), "mom", "hash", []int{s1, s2})
	require.NoError(t, err)
	assert.NotZero(t, p.ID)
	assert.Equal(t, "parent", p.Role)

	var links int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM parent_students WHERE parent_id = $1`, p.ID).Scan(&links))
	assert.Equal(t, 2, links)
}

func TestUserRepoCreateParent_DuplicateStudentIDs_LinksOnce(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	s1 := seedStudent(t, db, "student-a")

	p, err := repo.CreateParent(context.Background(), "mom", "hash", []int{s1, s1})
	require.NoError(t, err)

	var links int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM parent_students WHERE parent_id = $1`, p.ID).Scan(&links))
	assert.Equal(t, 1, links)
}

func TestUserRepoCreateParent_InvalidStudent_RollsBack(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	s1 := seedStudent(t, db, "student-a")

	_, err := repo.CreateParent(context.Background(), "mom", "hash", []int{s1, 99999})
	require.Error(t, err)
	var valErr *shared.ValidationError
	assert.True(t, errors.As(err, &valErr))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM users WHERE name = 'mom'`).Scan(&count))
	assert.Equal(t, 0, count)
}
