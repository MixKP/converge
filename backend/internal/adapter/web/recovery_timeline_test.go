package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
