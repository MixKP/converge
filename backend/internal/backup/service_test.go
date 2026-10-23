package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBackupPreservesLastVerifiedPointOnFailure(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.run = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "psql" {
			return []byte("160015\n"), nil
		}
		if len(args) == 1 && args[0] == "--version" {
			return []byte(name + " (PostgreSQL) 16.15 (Homebrew)\n"), nil
		}
		if name == "pg_dump" {
			return nil, os.WriteFile(strings.TrimPrefix(args[1], "--file="), []byte("archive"), 0600)
		}
		return nil, nil
	}
	require.NoError(t, s.backup(context.Background()))
	first, err := os.ReadFile(filepath.Join(dir, "latest"))
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, strings.TrimSpace(string(first))))
	require.FileExists(t, filepath.Join(dir, strings.TrimSpace(string(first))+".sha256"))
	s.run = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "psql" {
			return []byte("160015\n"), nil
		}
		if len(args) == 1 && args[0] == "--version" {
			return []byte(name + " (PostgreSQL) 16.15 (Homebrew)\n"), nil
		}
		if name == "pg_dump" {
			return nil, os.WriteFile(strings.TrimPrefix(args[1], "--file="), []byte("invalid"), 0600)
		}
		return nil, errors.New("verification failed")
	}
	require.ErrorContains(t, s.backup(context.Background()), "verification failed")
	latest, err := os.ReadFile(filepath.Join(dir, "latest"))
	require.NoError(t, err)
	require.Equal(t, first, latest)
	partials, err := filepath.Glob(filepath.Join(dir, ".*.partial"))
	require.NoError(t, err)
	require.Empty(t, partials)
}

func TestManualQueueAndPersistedStatus(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, s.Request())
	require.ErrorIs(t, s.Request(), ErrBusy)
	state, err := s.Status()
	require.NoError(t, err)
	require.True(t, state.Queued)
	require.NoError(t, writeTime(filepath.Join(dir, "last_success")))
	restarted := New(dir, time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	state, err = restarted.Status()
	require.NoError(t, err)
	require.NotNil(t, state.LastSuccess)
}

func TestSchedulerRunsAutomaticAndManualBackup(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	var dumps atomic.Int32
	s.run = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "psql" {
			return []byte("160015\n"), nil
		}
		if len(args) == 1 && args[0] == "--version" {
			return []byte(name + " (PostgreSQL) 16.15 (Homebrew)\n"), nil
		}
		if name == "pg_dump" {
			dumps.Add(1)
			return nil, os.WriteFile(strings.TrimPrefix(args[1], "--file="), []byte("archive"), 0600)
		}
		return nil, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	require.Eventually(t, func() bool { state, err := s.Status(); return err == nil && state.LastSuccess != nil && !state.Running }, 3*time.Second, 10*time.Millisecond)
	require.EqualValues(t, 1, dumps.Load())
	require.NoError(t, s.Request())
	require.Eventually(t, func() bool {
		state, err := s.Status()
		return err == nil && dumps.Load() == 2 && !state.Running && !state.Queued
	}, 3*time.Second, 10*time.Millisecond)
}

func TestBackupRejectsMismatchedClientVersion(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.run = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "psql" {
			return []byte("160015\n"), nil
		}
		if name == "pg_dump" && len(args) == 1 && args[0] == "--version" {
			return []byte("pg_dump (PostgreSQL) 18.1\n"), nil
		}
		t.Fatalf("unexpected backup command: %s", name)
		return nil, nil
	}
	require.ErrorContains(t, s.backup(context.Background()), "must match PostgreSQL server")
	require.NoFileExists(t, filepath.Join(dir, "latest"))
}

func TestLiveRestoreBlocksRequestsAndRecoversAfterSafetyBackupFailure(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "latest"), []byte("point.dump\n"), 0600))
	s := New(dir, time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	started := make(chan struct{})
	release := make(chan struct{})
	s.run = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "psql" {
			return []byte("160015\n"), nil
		}
		if len(args) == 1 && args[0] == "--version" {
			return []byte(name + " (PostgreSQL) 16.15\n"), nil
		}
		if name == "pg_dump" {
			close(started)
			<-release
			return nil, errors.New("safety backup failed")
		}
		return nil, nil
	}
	require.ErrorContains(t, s.RequestLiveRestore("point.dump", "wrong"), "confirmation")
	require.True(t, s.EnterRequest())
	s.LeaveRequest()
	require.NoError(t, s.RequestLiveRestore("point.dump", "RESTORE LIVE DATA from point.dump"))
	<-started
	require.False(t, s.EnterRequest())
	require.ErrorIs(t, s.Request(), ErrBusy)
	close(release)
	require.Eventually(t, func() bool {
		state, err := s.Status()
		return err == nil && !state.Restoring && strings.Contains(state.RestoreError, "safety backup failed")
	}, time.Second, time.Millisecond)
	require.True(t, s.EnterRequest())
	s.LeaveRequest()
}

func TestSavePreviewArchiveVerifiesTransfer(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.run = func(_ context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "pg_restore", name)
		require.Equal(t, "--list", args[0])
		return nil, nil
	}
	data := []byte("recovered archive")
	digest := sha256.Sum256(data)
	fetch := func(_ context.Context, dst io.Writer) (string, error) {
		_, err := dst.Write(data)
		return hex.EncodeToString(digest[:]), err
	}
	require.NoError(t, s.savePreviewArchive(context.Background(), "wal-preview-1.dump", fetch))
	archive, err := verifiedArchive(dir, "wal-preview-1.dump")
	require.NoError(t, err)
	require.Equal(t, data, mustReadFile(t, archive))

	badFetch := func(_ context.Context, dst io.Writer) (string, error) {
		_, err := dst.Write(data[:5])
		return hex.EncodeToString(digest[:]), err
	}
	require.ErrorContains(t, s.savePreviewArchive(context.Background(), "wal-preview-2.dump", badFetch), "checksum")
	require.NoFileExists(t, filepath.Join(dir, "wal-preview-2.dump"))
	require.ErrorContains(t, s.RequestPreviewRestore("bad", "APPLY WAL PREVIEW bad", fetch), "valid recovery preview")
}

func TestPreviewRestoreFailureClearsBusyStateAndTemporaryArchive(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	started := make(chan struct{})
	release := make(chan struct{})
	fetch := func(_ context.Context, dst io.Writer) (string, error) {
		close(started)
		<-release
		_, err := dst.Write([]byte("partial archive"))
		if err != nil {
			return "", err
		}
		return "", errors.New("worker connection lost")
	}
	require.ErrorContains(t, s.RequestPreviewRestore("123", "wrong", fetch), "confirmation")
	require.NoError(t, s.RequestPreviewRestore("123", "APPLY WAL PREVIEW 123", fetch))
	<-started
	require.False(t, s.EnterRequest())
	require.ErrorIs(t, s.Request(), ErrBusy)
	close(release)
	require.Eventually(t, func() bool {
		state, err := s.Status()
		return err == nil && !state.Restoring && strings.Contains(state.RestoreError, "worker connection lost")
	}, time.Second, time.Millisecond)
	require.True(t, s.EnterRequest())
	s.LeaveRequest()
	require.NoFileExists(t, filepath.Join(dir, "wal-preview-123.dump"))
	require.NoFileExists(t, filepath.Join(dir, "wal-preview-123.dump.sha256"))
	partials, err := filepath.Glob(filepath.Join(dir, ".wal-preview-*.partial"))
	require.NoError(t, err)
	require.Empty(t, partials)
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}
