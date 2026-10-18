package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApplyPreviewToIsolatedDatabase(t *testing.T) {
	if os.Getenv("CONVERGE_PREVIEW_APPLY_DRILL") != "1" {
		t.Skip("set CONVERGE_PREVIEW_APPLY_DRILL=1 for the isolated database drill")
	}
	if os.Getenv("DB_HOST") != "127.0.0.1" || os.Getenv("DB_PORT") != "5434" {
		t.Fatal("drill requires the isolated database on 127.0.0.1:5434")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	run := func(name string, args ...string) string {
		t.Helper()
		output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
		require.NoError(t, err, string(output))
		return strings.TrimSpace(string(output))
	}
	run("psql", "-v", "ON_ERROR_STOP=1", "-c", "CREATE TABLE pitr_apply_probe (value text); INSERT INTO pitr_apply_probe VALUES ('before')")
	t.Cleanup(func() {
		output, err := exec.Command("psql", "-v", "ON_ERROR_STOP=1", "-c", "DROP TABLE IF EXISTS pitr_apply_probe").CombinedOutput()
		require.NoError(t, err, string(output))
	})
	archive := filepath.Join(t.TempDir(), "preview.dump")
	run("pg_dump", "--format=custom", "--file="+archive)
	run("psql", "-v", "ON_ERROR_STOP=1", "-c", "UPDATE pitr_apply_probe SET value='after'")
	data, err := os.ReadFile(archive)
	require.NoError(t, err)
	digest := sha256.Sum256(data)
	s := New(t.TempDir(), time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	fetch := func(_ context.Context, dst io.Writer) (string, error) {
		_, err := dst.Write(data)
		return hex.EncodeToString(digest[:]), err
	}
	require.NoError(t, s.RequestPreviewRestore("123", "APPLY WAL PREVIEW 123", fetch))
	require.Eventually(t, func() bool {
		status, err := s.Status()
		return err == nil && !status.Restoring
	}, time.Minute, 100*time.Millisecond)
	status, err := s.Status()
	require.NoError(t, err)
	require.Empty(t, status.RestoreError)
	require.NotNil(t, status.RestoredAt)
	require.Equal(t, "before", run("psql", "-AtX", "-c", "SELECT value FROM pitr_apply_probe"))
}
