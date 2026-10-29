package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/RinZ5/converge/backend/internal/backup"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"time"
)

func TestRecoveryHandlerStatusAndManualRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "last_success"), []byte("2026-10-11T08:00:00Z\n"), 0600))
	handler := NewRecoveryHandler(backup.New(dir, time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil))), slog.New(slog.NewTextHandler(io.Discard, nil)))
	router := gin.New()
	router.GET("/recovery/backups", handler.Status)
	router.POST("/recovery/backups", handler.RequestBackup)

	status := httptest.NewRecorder()
	router.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/recovery/backups", nil))
	require.Equal(t, http.StatusOK, status.Code)
	require.JSONEq(t, `{"last_success":"2026-10-11T08:00:00Z","last_failure":null,"latest":"","queued":false,"running":false,"restoring":false,"restore_backup":"","restore_error":"","restored_at":null}`, status.Body.String())

	queued := httptest.NewRecorder()
	router.ServeHTTP(queued, httptest.NewRequest(http.MethodPost, "/recovery/backups", nil))
	require.Equal(t, http.StatusAccepted, queued.Code)
	status = httptest.NewRecorder()
	router.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/recovery/backups", nil))
	require.Contains(t, status.Body.String(), `"queued":true`)

	duplicate := httptest.NewRecorder()
	router.ServeHTTP(duplicate, httptest.NewRequest(http.MethodPost, "/recovery/backups", nil))
	require.Equal(t, http.StatusConflict, duplicate.Code)
}

func TestRecoveryHandlerUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewRecoveryHandler(backup.New("", time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil))), slog.New(slog.NewTextHandler(io.Discard, nil)))
	router := gin.New()
	router.GET("/recovery/backups", handler.Status)
	router.POST("/recovery/backups", handler.RequestBackup)

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, "/recovery/backups", nil))
		require.Equal(t, http.StatusServiceUnavailable, response.Code)
	}
}

func TestRecoveryHandlerMaintenance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := backup.New(t.TempDir(), time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewRecoveryHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	router := gin.New()
	router.Use(handler.Maintenance)
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	router.GET("/api/branches", ok)
	router.GET("/api/recovery/backups", ok)
	router.POST("/api/recovery/restores", ok)
	send := func(method, path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, nil))
		return response
	}

	require.Equal(t, http.StatusOK, send(http.MethodGet, "/api/branches").Code)

	started := make(chan struct{})
	release := make(chan struct{})
	fetch := func(_ context.Context, _ io.Writer) (string, error) {
		close(started)
		<-release
		return "", errors.New("download stopped")
	}
	require.NoError(t, service.RequestPreviewRestore("123", "APPLY WAL PREVIEW 123", fetch))
	<-started
	blocked := send(http.MethodGet, "/api/branches")
	require.Equal(t, http.StatusServiceUnavailable, blocked.Code)
	require.JSONEq(t, `{"error":"database restore is in progress"}`, blocked.Body.String())
	require.Equal(t, http.StatusOK, send(http.MethodGet, "/api/recovery/backups").Code)
	require.Equal(t, http.StatusOK, send(http.MethodPost, "/api/recovery/restores").Code)

	close(release)
	require.Eventually(t, func() bool {
		return send(http.MethodGet, "/api/branches").Code == http.StatusOK
	}, time.Second, 5*time.Millisecond)
}
