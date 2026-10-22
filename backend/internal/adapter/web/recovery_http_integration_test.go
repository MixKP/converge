//go:build integration

package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/adapter/db"
	"github.com/RinZ5/converge/backend/internal/adapter/token"
	"github.com/RinZ5/converge/backend/internal/backup"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestRecoveryHTTPWithPostgres(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("converge_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	connection, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	database, err := db.InitDBWithConnString(connection)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	require.NoError(t, db.AutoMigrate(database))

	_, err = database.Exec(`INSERT INTO branches (name, capacity) VALUES ('recoverable', 10)`)
	require.NoError(t, err)
	_, err = database.Exec(`UPDATE branches SET capacity = 20 WHERE name = 'recoverable'`)
	require.NoError(t, err)
	tx, err := database.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(`INSERT INTO branches (name, capacity) VALUES ('rolled back', 30)`)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	var eventID int64
	var eventAt time.Time
	require.NoError(t, database.QueryRow(`SELECT id, occurred_at FROM admin_audit_event ORDER BY id DESC LIMIT 1`).Scan(&eventID, &eventAt))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := backup.New(t.TempDir(), time.Hour, 24*time.Hour, logger)
	service.SetAfterRestore(func() error { return db.RecoverAfterLiveRestore(database) })
	handler := NewRecoveryTimelineHandler(database, service, logger)
	handler.url = "http://worker.test"
	handler.token = "worker-secret"
	workerAvailable := true
	var forwarded []struct {
		Target   time.Time `json:"target"`
		EventID  int64     `json:"event_id"`
		Position string    `json:"position"`
	}
	archiveCalls := 0
	handler.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer worker-secret", request.Header.Get("Authorization"))
		code, body := http.StatusOK, `{"backups":[],"archive_segment":"segment-1"}`
		switch request.URL.Path {
		case "/status":
			if !workerAvailable {
				code, body = http.StatusServiceUnavailable, "unavailable"
			}
		case "/previews":
			if request.Method == http.MethodPost {
				var preview struct {
					Target   time.Time `json:"target"`
					EventID  int64     `json:"event_id"`
					Position string    `json:"position"`
				}
				require.NoError(t, json.NewDecoder(request.Body).Decode(&preview))
				forwarded = append(forwarded, preview)
				code, body = http.StatusAccepted, `{"id":"77","state":"running"}`
			} else {
				body = `{"id":"77","state":"ready","event_id":1,"selection_matched":true}`
			}
		case "/previews/archive":
			archiveCalls++
			code, body = http.StatusInternalServerError, "unexpected archive request"
		default:
			t.Fatalf("unexpected worker path %s", request.URL.Path)
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	jwt := token.NewJWT([]byte("integration-test-secret"), time.Hour)
	adminToken, err := jwt.Issue(shared.Principal{UserID: 1, Role: shared.RoleAdmin})
	require.NoError(t, err)
	studentToken, err := jwt.Issue(shared.Principal{UserID: 2, Role: shared.RoleStudent})
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api", AuthMiddleware(jwt), RequireRole(shared.RoleAdmin))
	admin.GET("/recovery/timeline", handler.Timeline)
	admin.POST("/recovery/previews", handler.Preview)
	admin.POST("/recovery/previews/apply", handler.ApplyPreview)
	request := func(method, path, body, bearer string) *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		httpRequest := httptest.NewRequest(method, path, strings.NewReader(body))
		if bearer != "" {
			httpRequest.Header.Set("Authorization", "Bearer "+bearer)
		}
		httpRequest.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, httpRequest)
		return response
	}

	require.Equal(t, http.StatusUnauthorized, request(http.MethodGet, "/api/recovery/timeline", "", "").Code)
	require.Equal(t, http.StatusForbidden, request(http.MethodGet, "/api/recovery/timeline", "", studentToken).Code)
	response := request(http.MethodGet, "/api/recovery/timeline", "", adminToken)
	require.Equal(t, http.StatusOK, response.Code)
	var timeline struct {
		Events []struct {
			ID        int64  `json:"id"`
			Operation string `json:"operation"`
		} `json:"events"`
		Physical json.RawMessage `json:"physical"`
		Error    string          `json:"physical_error"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &timeline))
	require.Len(t, timeline.Events, 2)
	require.Equal(t, eventID, timeline.Events[0].ID)
	require.Equal(t, "update", timeline.Events[0].Operation)
	require.Greater(t, timeline.Events[0].ID, timeline.Events[1].ID)
	require.JSONEq(t, `{"backups":[],"archive_segment":"segment-1"}`, string(timeline.Physical))
	require.Empty(t, timeline.Error)

	require.Eventually(t, func() bool { return time.Now().After(eventAt.Add(time.Second)) }, 3*time.Second, 10*time.Millisecond)
	for _, position := range []string{"before", "after"} {
		response = request(http.MethodPost, "/api/recovery/previews", `{"event_id":`+strconv.FormatInt(eventID, 10)+`,"position":"`+position+`"}`, adminToken)
		require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	}
	require.Len(t, forwarded, 2)
	require.Equal(t, eventID, forwarded[0].EventID)
	require.Equal(t, "before", forwarded[0].Position)
	require.Equal(t, eventAt.Add(-time.Microsecond), forwarded[0].Target)
	require.Equal(t, eventAt.Add(time.Second), forwarded[1].Target)

	require.Equal(t, http.StatusNotFound, request(http.MethodPost, "/api/recovery/previews", `{"event_id":999999,"position":"after"}`, adminToken).Code)
	require.Equal(t, http.StatusBadRequest, request(http.MethodPost, "/api/recovery/previews", `{"target":"2099-01-01T00:00:00Z"}`, adminToken).Code)
	require.Len(t, forwarded, 2)

	workerAvailable = false
	response = request(http.MethodGet, "/api/recovery/timeline", "", adminToken)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"physical_error":"Physical backup status is unavailable"`)
	workerAvailable = true

	wrongConfirmation := `{"id":"77","confirmation":"wrong"}`
	require.Equal(t, http.StatusBadRequest, request(http.MethodPost, "/api/recovery/previews/apply", wrongConfirmation, adminToken).Code)
	require.Equal(t, http.StatusConflict, request(http.MethodPost, "/api/recovery/previews/apply", `{"id":"78","confirmation":"APPLY WAL PREVIEW 78"}`, adminToken).Code)
	require.Zero(t, archiveCalls)
	var capacity int
	require.NoError(t, database.QueryRow(`SELECT capacity FROM branches WHERE name = 'recoverable'`).Scan(&capacity))
	require.Equal(t, 20, capacity)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	t.Setenv("DB_HOST", host)
	t.Setenv("DB_PORT", port.Port())
	t.Setenv("POSTGRES_USER", "test")
	t.Setenv("POSTGRES_PASSWORD", "test")
	t.Setenv("POSTGRES_DB", "converge_test")
	t.Setenv("PGHOST", host)
	t.Setenv("PGPORT", port.Port())
	t.Setenv("PGUSER", "test")
	t.Setenv("PGPASSWORD", "test")
	t.Setenv("PGDATABASE", "converge_test")
	archive := filepath.Join(t.TempDir(), "preview.dump")
	output, err := exec.Command("pg_dump", "--format=custom", "--file="+archive).CombinedOutput()
	require.NoError(t, err, string(output))
	archiveBytes, err := os.ReadFile(archive)
	require.NoError(t, err)
	digest := sha256.Sum256(archiveBytes)
	_, err = database.Exec(`UPDATE branches SET capacity = 30 WHERE name = 'recoverable'`)
	require.NoError(t, err)
	var downloaded atomic.Bool
	workerServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer worker-secret" || request.URL.Path != "/previews/archive" || request.URL.Query().Get("id") != "77" {
			http.Error(writer, "unexpected archive request", http.StatusBadRequest)
			return
		}
		downloaded.Store(true)
		writer.Header().Set("X-Archive-Sha256", hex.EncodeToString(digest[:]))
		_, _ = writer.Write(archiveBytes)
	}))
	t.Cleanup(workerServer.Close)
	handler.url = workerServer.URL
	response = request(http.MethodPost, "/api/recovery/previews/apply", `{"id":"77","confirmation":"APPLY WAL PREVIEW 77"}`, adminToken)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	require.Eventually(t, func() bool {
		status, err := service.Status()
		return err == nil && !status.Restoring
	}, 2*time.Minute, 100*time.Millisecond)
	status, err := service.Status()
	require.NoError(t, err)
	require.Empty(t, status.RestoreError)
	require.NotNil(t, status.RestoredAt)
	require.True(t, downloaded.Load())
	require.NoError(t, database.QueryRow(`SELECT capacity FROM branches WHERE name = 'recoverable'`).Scan(&capacity))
	require.Equal(t, 20, capacity)
}
