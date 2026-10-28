//go:build integration

package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuditRecordsCommittedChangesOnly(t *testing.T) {
	database := setupTestDB(t)
	tx, err := database.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(`INSERT INTO branches (name, capacity) VALUES ('discarded', 10)`)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	var count int
	require.NoError(t, database.QueryRow(`SELECT count(*) FROM admin_audit_event`).Scan(&count))
	require.Zero(t, count)

	_, err = database.Exec(`INSERT INTO branches (name, capacity) VALUES ('kept', 10)`)
	require.NoError(t, err)
	_, err = database.Exec(`UPDATE branches SET capacity = 20 WHERE name = 'kept'`)
	require.NoError(t, err)
	var operation, resource, field string
	require.NoError(t, database.QueryRow(`SELECT operation, resource, changed_fields[1] FROM admin_audit_event ORDER BY id DESC LIMIT 1`).Scan(&operation, &resource, &field))
	require.Equal(t, "update", operation)
	require.Equal(t, "branches", resource)
	require.Equal(t, "capacity", field)
}

func TestRecoverAfterLiveRestorePreservesHistoryAndRepairsTriggers(t *testing.T) {
	database := setupTestDB(t)
	_, err := database.Exec(`INSERT INTO branches (name, capacity) VALUES ('restored', 10)`)
	require.NoError(t, err)
	require.NoError(t, RecoverAfterLiveRestore(database))
	var count int
	require.NoError(t, database.QueryRow(`SELECT count(*) FROM admin_audit_event`).Scan(&count))
	require.Equal(t, 1, count)

	_, err = database.Exec(`DROP TRIGGER admin_data_change ON branches`)
	require.NoError(t, err)
	require.NoError(t, RecoverAfterLiveRestore(database))
	require.NoError(t, database.QueryRow(`SELECT count(*) FROM admin_audit_event`).Scan(&count))
	require.Equal(t, 1, count)

	_, err = database.Exec(`UPDATE branches SET capacity = 20 WHERE name = 'restored'`)
	require.NoError(t, err)
	require.NoError(t, database.QueryRow(`SELECT count(*) FROM admin_audit_event`).Scan(&count))
	require.Equal(t, 2, count)
}
