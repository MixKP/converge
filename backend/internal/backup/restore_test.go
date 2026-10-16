package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func restoreFixture(t *testing.T) RestoreOptions {
	t.Helper()
	dir := t.TempDir()
	name := "converge_test.dump"
	data := []byte("verified archive")
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0600))
	sum := sha256.Sum256(data)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name+".sha256"), []byte(hex.EncodeToString(sum[:])+"  "+name+"\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "latest"), []byte(name+"\n"), 0600))
	return RestoreOptions{BackupDir: dir, BackupName: "latest", SourceHost: "localhost", SourcePort: "5432", TargetHost: "127.0.0.1", TargetPort: "5433", TargetDB: "converge_restore", TargetUser: "converge", TargetPassword: "test", Confirmation: "RESTORE to 127.0.0.1:5433/converge_restore"}
}

func TestRestoreChecksArchiveAndIsolatedTarget(t *testing.T) {
	options := restoreFixture(t)
	var commands []string
	run := func(_ context.Context, name string, args, _ []string) ([]byte, error) {
		commands = append(commands, name+" "+strings.Join(args, " "))
		if name == "psql" && strings.Contains(strings.Join(args, " "), "server_version_num") {
			return []byte("160015\n"), nil
		}
		if name == "pg_restore" && len(args) == 1 && args[0] == "--version" {
			return []byte("pg_restore (PostgreSQL) 16.15 (Homebrew)\n"), nil
		}
		if name == "psql" && strings.Contains(strings.Join(args, " "), "pg_tables") {
			return []byte("0\n"), nil
		}
		return []byte("1\n"), nil
	}
	require.NoError(t, restore(context.Background(), options, false, run))
	require.Len(t, commands, 9)
	require.Contains(t, commands[2], "pg_restore --list")
	require.Contains(t, commands[4], "--single-transaction")

	options.TargetPort = "5432"
	options.Confirmation = "RESTORE to 127.0.0.1:5432/converge_restore"
	require.ErrorContains(t, restore(context.Background(), options, false, run), "production database server")
	require.Len(t, commands, 9)
}

func TestRestoreRejectsBadChecksumAndNonemptyTarget(t *testing.T) {
	options := restoreFixture(t)
	var calls int
	run := func(_ context.Context, name string, args, _ []string) ([]byte, error) {
		calls++
		if name == "psql" && strings.Contains(strings.Join(args, " "), "server_version_num") {
			return []byte("160015\n"), nil
		}
		if name == "pg_restore" && len(args) == 1 && args[0] == "--version" {
			return []byte("pg_restore (PostgreSQL) 16.15 (Homebrew)\n"), nil
		}
		if name == "psql" {
			return []byte("1\n"), nil
		}
		return nil, nil
	}
	require.ErrorContains(t, restore(context.Background(), options, false, run), "not empty")
	require.Equal(t, 4, calls)
	require.NoError(t, os.WriteFile(filepath.Join(options.BackupDir, "converge_test.dump"), []byte("damaged"), 0600))
	require.ErrorContains(t, restore(context.Background(), options, false, run), "checksum does not match")
	require.Equal(t, 4, calls)
}

func TestRestoreRejectsMismatchedClientVersion(t *testing.T) {
	options := restoreFixture(t)
	var calls int
	run := func(_ context.Context, name string, args, _ []string) ([]byte, error) {
		calls++
		if name == "psql" {
			return []byte("160015\n"), nil
		}
		if name == "pg_restore" && len(args) == 1 && args[0] == "--version" {
			return []byte("pg_restore (PostgreSQL) 18.1\n"), nil
		}
		t.Fatalf("unexpected command: %s", name)
		return nil, nil
	}
	require.ErrorContains(t, restore(context.Background(), options, false, run), "must match replacement PostgreSQL")
	require.Equal(t, 2, calls)
}

func TestLiveRestoreRequiresExactConfirmationAndCleansInOneTransaction(t *testing.T) {
	options := restoreFixture(t)
	options.TargetHost = options.SourceHost
	options.TargetPort = options.SourcePort
	options.TargetDB = "converge_db"
	options.BackupName = "converge_test.dump"
	var commands []string
	run := func(_ context.Context, name string, args, _ []string) ([]byte, error) {
		commands = append(commands, name+" "+strings.Join(args, " "))
		if name == "psql" && strings.Contains(strings.Join(args, " "), "server_version_num") {
			return []byte("160015\n"), nil
		}
		if name == "pg_restore" && len(args) == 1 && args[0] == "--version" {
			return []byte("pg_restore (PostgreSQL) 16.15\n"), nil
		}
		return []byte("1\n"), nil
	}
	require.ErrorContains(t, restore(context.Background(), options, true, run), "confirmation must be")
	require.Empty(t, commands)
	options.Confirmation = "RESTORE LIVE DATA from converge_test.dump"
	require.NoError(t, restore(context.Background(), options, true, run))
	require.Len(t, commands, 8)
	require.Contains(t, commands[3], "--single-transaction")
	require.Contains(t, commands[3], "--clean --if-exists")
	for _, cmd := range commands {
		require.NotContains(t, cmd, "pg_tables")
	}
}
