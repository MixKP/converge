package web

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/backup"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRecoveryPreviewWorkerResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name       string
		workerCode int
		workerBody string
		wantCode   int
		wantBody   string
	}{
		{"accepted", http.StatusAccepted, `{"id":"preview-1","state":"running"}`, http.StatusAccepted, `"state":"running"`},
		{"outside backup coverage", http.StatusBadRequest, "Selected time is before the first physical backup. Choose a later time.", http.StatusBadRequest, "Selected time is before the first physical backup"},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost || r.URL.Path != "/previews" || r.Header.Get("Authorization") != "Bearer test-token" {
					return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader("unauthorized")), Header: make(http.Header)}, nil
				}
				return &http.Response{StatusCode: test.workerCode, Body: io.NopCloser(strings.NewReader(test.workerBody)), Header: make(http.Header)}, nil
			})}
			handler := &RecoveryTimelineHandler{client: client, url: "http://worker.test", token: "test-token", logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
			router := gin.New()
			router.POST("/previews", handler.Preview)
			body := `{"target":"` + time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano) + `"}`
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/previews", strings.NewReader(body)))
			if response.Code != test.wantCode || !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("got %d %s; want %d containing %q", response.Code, response.Body.String(), test.wantCode, test.wantBody)
			}
		})
	}
}

func TestApplyPreviewRejectsChangedOrUnavailablePreview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name       string
		workerBody string
		workerErr  error
		request    string
		wantCode   int
	}{
		{"not ready", `{"id":"123","state":"running"}`, nil, `{"id":"123","confirmation":"APPLY WAL PREVIEW 123"}`, http.StatusConflict},
		{"stale ID", `{"id":"124","state":"ready"}`, nil, `{"id":"123","confirmation":"APPLY WAL PREVIEW 123"}`, http.StatusConflict},
		{"selection mismatch", `{"id":"123","state":"ready","event_id":7,"selection_matched":false}`, nil, `{"id":"123","confirmation":"APPLY WAL PREVIEW 123"}`, http.StatusConflict},
		{"worker unavailable", "", errors.New("connection refused"), `{"id":"123","confirmation":"APPLY WAL PREVIEW 123"}`, http.StatusServiceUnavailable},
		{"malformed worker response", `not JSON`, nil, `{"id":"123","confirmation":"APPLY WAL PREVIEW 123"}`, http.StatusServiceUnavailable},
		{"wrong confirmation", `{"id":"123","state":"ready"}`, nil, `{"id":"123","confirmation":"wrong"}`, http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := backup.New(t.TempDir(), time.Hour, 7*24*time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
			handler := &RecoveryTimelineHandler{
				backup: service, url: "http://worker.test", token: "test-token",
				client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					require.Equal(t, "/previews", r.URL.Path)
					require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
					if test.workerErr != nil {
						return nil, test.workerErr
					}
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(test.workerBody)), Header: make(http.Header)}, nil
				})},
			}
			router := gin.New()
			router.POST("/apply", handler.ApplyPreview)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/apply", strings.NewReader(test.request)))
			require.Equal(t, test.wantCode, response.Code, response.Body.String())
			status, err := service.Status()
			require.NoError(t, err)
			require.False(t, status.Restoring)
		})
	}
}

func TestRecoveryPreviewRejectsInvalidTargetBeforeWorkerCall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, body := range []string{
		`{"target":"2099-01-01T00:00:00Z"}`,
		`{"event_id":3,"position":"before","target":"2026-10-18T12:00:00Z"}`,
		`{"position":"after","target":"2026-10-18T12:00:00Z"}`,
		`{"target":"not-a-time"}`,
	} {
		handler := &RecoveryTimelineHandler{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("invalid target reached worker")
			return nil, nil
		})}, url: "http://worker.test", token: "test-token"}
		router := gin.New()
		router.POST("/previews", handler.Preview)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/previews", strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, response.Code, body)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRecoveryTimelineRejectsInvalidCursor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct{ query, message string }{
		{"after=not-a-time", "after must be an ISO timestamp"},
		{"before_id=abc", "before_id must be positive"},
		{"before_id=0", "before_id must be positive"},
		{"before_id=-1", "before_id must be positive"},
		{"before_id=9223372036854775808", "before_id must be positive"},
		{"before_id=2", "before_at must be an ISO timestamp"},
		{"before_id=2&before_at=not-a-time", "before_at must be an ISO timestamp"},
	} {
		t.Run(test.query, func(t *testing.T) {
			handler := &RecoveryTimelineHandler{}
			router := gin.New()
			router.GET("/timeline", handler.Timeline)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/timeline?"+test.query, nil))
			require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
			require.JSONEq(t, `{"error":"`+test.message+`"}`, response.Body.String())
		})
	}
}
