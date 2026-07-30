package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RinZ5/converge/backend/internal/commute"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCommuteService struct {
	minutes int
	err     error
	args    struct {
		source, dest int
	}
	called bool
}

func (m *mockCommuteService) CommuteTime(ctx context.Context, sourceBranchID, destBranchID int) (int, error) {
	m.called = true
	m.args.source = sourceBranchID
	m.args.dest = destBranchID
	return m.minutes, m.err
}

func TestCommuteHandler_GetCommute(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		mock       *mockCommuteService
		wantStatus int
		wantCalled bool
		wantTime   int
	}{
		{
			name:       "no params returns default",
			path:       "/api/commute",
			mock:       &mockCommuteService{},
			wantStatus: http.StatusOK,
			wantCalled: false,
			wantTime:   commute.DefaultCommuteMinutes,
		},
		{
			name:       "same branch returns zero",
			path:       "/api/commute?source_branch=1&destination_branch=1",
			mock:       &mockCommuteService{minutes: 0},
			wantStatus: http.StatusOK,
			wantCalled: true,
			wantTime:   0,
		},
		{
			name:       "different branches returns default",
			path:       "/api/commute?source_branch=1&destination_branch=2",
			mock:       &mockCommuteService{minutes: commute.DefaultCommuteMinutes},
			wantStatus: http.StatusOK,
			wantCalled: true,
			wantTime:   commute.DefaultCommuteMinutes,
		},
		{
			name:       "only source provided",
			path:       "/api/commute?source_branch=1",
			mock:       &mockCommuteService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:       "only destination provided",
			path:       "/api/commute?destination_branch=2",
			mock:       &mockCommuteService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:       "non-numeric source",
			path:       "/api/commute?source_branch=abc&destination_branch=2",
			mock:       &mockCommuteService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:       "non-numeric destination",
			path:       "/api/commute?source_branch=1&destination_branch=xyz",
			mock:       &mockCommuteService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:       "service validation error",
			path:       "/api/commute?source_branch=0&destination_branch=2",
			mock:       &mockCommuteService{err: &shared.ValidationError{Msg: "source_branch must be positive"}},
			wantStatus: http.StatusBadRequest,
			wantCalled: true,
		},
		{
			name:       "service error",
			path:       "/api/commute?source_branch=1&destination_branch=2",
			mock:       &mockCommuteService{err: assert.AnError},
			wantStatus: http.StatusInternalServerError,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewCommuteHandler(tt.mock, slog.Default())

			gin.SetMode(gin.TestMode)
			r := gin.Default()
			r.GET("/api/commute", handler.GetCommute)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantCalled, tt.mock.called)

			if tt.wantStatus == http.StatusOK {
				var resp commute.CommuteResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, tt.wantTime, resp.CommuteTime)
			}
		})
	}
}
