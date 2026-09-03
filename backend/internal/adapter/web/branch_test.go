package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RinZ5/converge/backend/internal/branch"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBranchService struct {
	branches      []branch.Branch
	getErr        error
	addBranchErr  error
	addBranchArgs struct {
		name     string
		capacity int
	}
	addBranchCalled bool
	setCapacityErr  error
	setCapacityArgs struct {
		branchID, capacity int
	}
	setCapacityCalled bool
}

func (m *mockBranchService) AddBranch(ctx context.Context, name string, capacity int) (*branch.Branch, error) {
	m.addBranchCalled = true
	m.addBranchArgs.name = name
	m.addBranchArgs.capacity = capacity
	if m.addBranchErr != nil {
		return nil, m.addBranchErr
	}
	return &branch.Branch{ID: 1, Name: name, Capacity: capacity}, nil
}

func (m *mockBranchService) GetBranches(ctx context.Context) ([]branch.Branch, error) {
	return m.branches, m.getErr
}

func (m *mockBranchService) SetCapacity(ctx context.Context, branchID, capacity int) error {
	m.setCapacityCalled = true
	m.setCapacityArgs.branchID = branchID
	m.setCapacityArgs.capacity = capacity
	return m.setCapacityErr
}

func TestBranchHandler_GetBranches_Success(t *testing.T) {
	mock := &mockBranchService{branches: []branch.Branch{{ID: 1, Name: "Siam", Capacity: 30}}}
	handler := NewBranchHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/branches", handler.GetBranches)

	req := httptest.NewRequest(http.MethodGet, "/api/branches", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var branches []branch.Branch
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &branches))
	assert.Len(t, branches, 1)
	assert.Equal(t, 30, branches[0].Capacity)
}

func TestBranchHandler_GetBranches_Error(t *testing.T) {
	mock := &mockBranchService{getErr: assert.AnError}
	handler := NewBranchHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/branches", handler.GetBranches)

	req := httptest.NewRequest(http.MethodGet, "/api/branches", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestBranchHandler_UpdateBranchCapacity(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		body         string
		mock         *mockBranchService
		wantStatus   int
		wantCalled   bool
		wantCapacity int
	}{
		{
			name:         "success",
			path:         "/api/branches/1/capacity",
			body:         `{"capacity": 30}`,
			mock:         &mockBranchService{},
			wantStatus:   http.StatusOK,
			wantCalled:   true,
			wantCapacity: 30,
		},
		{
			name:         "zero capacity allowed",
			path:         "/api/branches/1/capacity",
			body:         `{"capacity": 0}`,
			mock:         &mockBranchService{},
			wantStatus:   http.StatusOK,
			wantCalled:   true,
			wantCapacity: 0,
		},
		{
			name:       "invalid id",
			path:       "/api/branches/abc/capacity",
			body:       `{"capacity": 30}`,
			mock:       &mockBranchService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:       "capacity omitted",
			path:       "/api/branches/1/capacity",
			body:       `{}`,
			mock:       &mockBranchService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:       "capacity null",
			path:       "/api/branches/1/capacity",
			body:       `{"capacity": null}`,
			mock:       &mockBranchService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:         "not found",
			path:         "/api/branches/99/capacity",
			body:         `{"capacity": 30}`,
			mock:         &mockBranchService{setCapacityErr: &shared.NotFoundError{Msg: "branch 99 not found"}},
			wantStatus:   http.StatusNotFound,
			wantCalled:   true,
			wantCapacity: 30,
		},
		{
			name:         "validation error",
			path:         "/api/branches/1/capacity",
			body:         `{"capacity": -1}`,
			mock:         &mockBranchService{setCapacityErr: &shared.ValidationError{Msg: "capacity must not be negative"}},
			wantStatus:   http.StatusBadRequest,
			wantCalled:   true,
			wantCapacity: -1,
		},
		{
			name:         "service error",
			path:         "/api/branches/1/capacity",
			body:         `{"capacity": 30}`,
			mock:         &mockBranchService{setCapacityErr: assert.AnError},
			wantStatus:   http.StatusInternalServerError,
			wantCalled:   true,
			wantCapacity: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewBranchHandler(tt.mock, slog.Default())

			req := httptest.NewRequest(http.MethodPatch, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			gin.SetMode(gin.TestMode)
			r := gin.Default()
			r.PATCH("/api/branches/:id/capacity", handler.UpdateBranchCapacity)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantCalled, tt.mock.setCapacityCalled)
			if tt.wantCalled {
				assert.Equal(t, tt.wantCapacity, tt.mock.setCapacityArgs.capacity)
			}
		})
	}
}

func TestBranchHandler_CreateBranch(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		mock         *mockBranchService
		wantStatus   int
		wantCalled   bool
		wantName     string
		wantCapacity int
	}{
		{
			name:         "success",
			body:         `{"name": "Siam", "capacity": 30}`,
			mock:         &mockBranchService{},
			wantStatus:   http.StatusCreated,
			wantCalled:   true,
			wantName:     "Siam",
			wantCapacity: 30,
		},
		{
			name:         "capacity omitted defaults to zero",
			body:         `{"name": "Siam"}`,
			mock:         &mockBranchService{},
			wantStatus:   http.StatusCreated,
			wantCalled:   true,
			wantName:     "Siam",
			wantCapacity: 0,
		},
		{
			name:       "name missing",
			body:       `{"capacity": 30}`,
			mock:       &mockBranchService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:       "invalid json",
			body:       `{"name":`,
			mock:       &mockBranchService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:         "duplicate name",
			body:         `{"name": "Siam"}`,
			mock:         &mockBranchService{addBranchErr: &shared.ConflictError{Msg: `branch name "Siam" already exists`}},
			wantStatus:   http.StatusConflict,
			wantCalled:   true,
			wantName:     "Siam",
			wantCapacity: 0,
		},
		{
			name:         "validation error",
			body:         `{"name": "Siam", "capacity": -1}`,
			mock:         &mockBranchService{addBranchErr: &shared.ValidationError{Msg: "capacity must not be negative"}},
			wantStatus:   http.StatusBadRequest,
			wantCalled:   true,
			wantName:     "Siam",
			wantCapacity: -1,
		},
		{
			name:         "service error",
			body:         `{"name": "Siam"}`,
			mock:         &mockBranchService{addBranchErr: assert.AnError},
			wantStatus:   http.StatusInternalServerError,
			wantCalled:   true,
			wantName:     "Siam",
			wantCapacity: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewBranchHandler(tt.mock, slog.Default())

			req := httptest.NewRequest(http.MethodPost, "/api/branches", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			gin.SetMode(gin.TestMode)
			r := gin.Default()
			r.POST("/api/branches", handler.CreateBranch)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantCalled, tt.mock.addBranchCalled)
			if tt.wantCalled {
				assert.Equal(t, tt.wantName, tt.mock.addBranchArgs.name)
				assert.Equal(t, tt.wantCapacity, tt.mock.addBranchArgs.capacity)
			}
			if tt.wantStatus == http.StatusCreated {
				var created branch.Branch
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
				assert.Equal(t, tt.wantName, created.Name)
				assert.Equal(t, tt.wantCapacity, created.Capacity)
			}
		})
	}
}
