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

func TestCleanupRetainsLatestSafetyAndRecentArchives(t *testing.T) {
	t.Setenv("POSTGRES_DB", "cleanup_test")
	dir := t.TempDir()
	service := New(dir, time.Hour, 24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	now := time.Now()
	for _, file := range []struct {
		name string
		age  time.Duration
	}{
		{"cleanup_test_expired.dump", 48 * time.Hour},
		{"cleanup_test_recent.dump", time.Hour},
		{"cleanup_test_latest.dump", 48 * time.Hour},
		{"cleanup_test_safety.dump", 48 * time.Hour},
		{"unrelated.dump", 48 * time.Hour},
	} {
		for _, suffix := range []string{"", ".sha256"} {
			path := filepath.Join(dir, file.name+suffix)
			require.NoError(t, os.WriteFile(path, []byte("fixture"), 0600))
			require.NoError(t, os.Chtimes(path, now.Add(-file.age), now.Add(-file.age)))
		}
	}
	service.cleanup("cleanup_test_latest.dump", "cleanup_test_safety.dump")
	for _, name := range []string{"cleanup_test_recent.dump", "cleanup_test_latest.dump", "cleanup_test_safety.dump", "unrelated.dump"} {
		require.FileExists(t, filepath.Join(dir, name))
		require.FileExists(t, filepath.Join(dir, name+".sha256"))
	}
	require.NoFileExists(t, filepath.Join(dir, "cleanup_test_expired.dump"))
	require.NoFileExists(t, filepath.Join(dir, "cleanup_test_expired.dump.sha256"))
}

func fakeCommands(t *testing.T, scripts map[string]string) {
	t.Helper()
	bin := t.TempDir()
	for name, body := range scripts {
		require.NoError(t, os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0700))
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestNewStartsIdle(t *testing.T) {
	s := New(t.TempDir(), time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	state, err := s.Status()
	require.NoError(t, err)
	require.Equal(t, Status{}, state)
	require.True(t, s.EnterRequest())
	s.LeaveRequest()
}

func TestRestoreLiveArchiveRunsSetupAfterRestore(t *testing.T) {
	dir := t.TempDir()
	data := []byte("verified archive")
	sum := sha256.Sum256(data)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "converge_test.dump"), data, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "converge_test.dump.sha256"), []byte(hex.EncodeToString(sum[:])+"  converge_test.dump\n"), 0600))
	t.Setenv("POSTGRES_PASSWORD", "test")
	fakeCommands(t, map[string]string{
		"psql":       `case "$*" in *server_version_num*) echo 160015;; *) echo 1;; esac`,
		"pg_restore": `case "$1" in --version) echo "pg_restore (PostgreSQL) 16.15";; --list) echo "1; 0 0 TABLE public branches converge";; esac`,
	})
	s := New(dir, time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	var hookCalls atomic.Int32
	var hookErr error
	s.SetAfterRestore(func() error {
		hookCalls.Add(1)
		return hookErr
	})

	require.NoError(t, s.restoreLiveArchive(context.Background(), "converge_test.dump"))
	require.Equal(t, int32(1), hookCalls.Load())

	hookErr = errors.New("trigger repair failed")
	require.EqualError(t, s.restoreLiveArchive(context.Background(), "converge_test.dump"), "live data was replaced but post-restore setup failed: trigger repair failed")
	require.Equal(t, int32(2), hookCalls.Load())

	err := s.restoreLiveArchive(context.Background(), "missing.dump")
	require.EqualError(t, err, "read backup checksum: open "+filepath.Join(dir, "missing.dump.sha256")+": no such file or directory")
	require.Equal(t, int32(2), hookCalls.Load())
}

func TestEnterRequest(t *testing.T) {
	newService := func() *Service {
		return New(t.TempDir(), time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
	t.Run("Idle", func(t *testing.T) {
		s := newService()
		require.True(t, s.EnterRequest())
		s.LeaveRequest()
	})
	t.Run("Restoring", func(t *testing.T) {
		s := newService()
		s.restoring.Store(true)
		require.False(t, s.EnterRequest())
	})
	t.Run("GateHeldByRestore", func(t *testing.T) {
		s := newService()
		s.gate.Lock()
		defer s.gate.Unlock()
		require.False(t, s.EnterRequest())
	})
}

func TestLeaveRequest(t *testing.T) {
	newService := func() *Service {
		return New(t.TempDir(), time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
	t.Run("ReleasesGate", func(t *testing.T) {
		s := newService()
		require.True(t, s.EnterRequest())
		require.False(t, s.gate.TryLock())
		s.LeaveRequest()
		require.True(t, s.gate.TryLock())
		s.gate.Unlock()
	})
	t.Run("KeepsOtherRequests", func(t *testing.T) {
		s := newService()
		require.True(t, s.EnterRequest())
		require.True(t, s.EnterRequest())
		s.LeaveRequest()
		require.False(t, s.gate.TryLock())
		s.LeaveRequest()
		require.True(t, s.gate.TryLock())
		s.gate.Unlock()
	})
}
