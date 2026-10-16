package web

import (
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
