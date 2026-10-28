//go:build integration

package db

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestBootstrapAdminPreservesExistingData(t *testing.T) {
	database := setupTestDB(t)
	t.Setenv("ADMIN_USERNAME", "operator")
	t.Setenv("ADMIN_PASSWORD", "initial-admin-password")
	_, err := database.Exec(`INSERT INTO branches(name, capacity) VALUES ('Existing branch', 10)`)
	require.NoError(t, err)
	require.NoError(t, BootstrapAdmin(database))

	var originalHash, role string
	require.NoError(t, database.QueryRow(`SELECT password_hash, role FROM users WHERE name = 'operator'`).Scan(&originalHash, &role))
	require.Equal(t, "admin", role)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(originalHash), []byte("initial-admin-password")))

	t.Setenv("ADMIN_PASSWORD", "different-password-on-restart")
	require.NoError(t, BootstrapAdmin(database))
	var hash string
	require.NoError(t, database.QueryRow(`SELECT password_hash FROM users WHERE name = 'operator'`).Scan(&hash))
	require.Equal(t, originalHash, hash)
	var count int
	require.NoError(t, database.QueryRow(`SELECT count(*) FROM branches WHERE name = 'Existing branch' AND capacity = 10`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, database.QueryRow(`SELECT count(*) FROM users WHERE role = 'admin'`).Scan(&count))
	require.Equal(t, 1, count)

	t.Setenv("ADMIN_PASSWORD", "")
	require.ErrorContains(t, BootstrapAdmin(database), "ADMIN_PASSWORD is required")
	t.Setenv("ADMIN_PASSWORD", "valid-password")
	t.Setenv("ADMIN_USERNAME", "student")
	_, err = database.Exec(`INSERT INTO users(name, password_hash, role) VALUES ('student', 'existing-hash', 'student')`)
	require.NoError(t, err)
	require.ErrorContains(t, BootstrapAdmin(database), "non-admin account")
	require.NoError(t, database.QueryRow(`SELECT password_hash, role FROM users WHERE name = 'student'`).Scan(&hash, &role))
	require.Equal(t, "existing-hash", hash)
	require.Equal(t, "student", role)
}
