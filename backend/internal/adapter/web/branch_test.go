package web

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RinZ5/converge/backend/internal/branch"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBranchService struct {
	branches        []branch.Branch
	getErr          error
	setCapacityErr  error
	setCapacityArgs struct {
		branchID, capacity int
	}
	setCapacityCalled bool
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

func TestBranchHandler_UpdateBranchCapacity_Success(t *testing.T) {
	mock := &mockBranchService{}
	handler := NewBranchHandler(mock, slog.Default())

	body, _ := json.Marshal(map[string]interface{}{"capacity": 30})
	req := httptest.NewRequest(http.MethodPatch, "/api/branches/1/capacity", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/branches/:id/capacity", handler.UpdateBranchCapacity)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mock.setCapacityCalled)
	assert.Equal(t, 1, mock.setCapacityArgs.branchID)
	assert.Equal(t, 30, mock.setCapacityArgs.capacity)
}

func TestBranchHandler_UpdateBranchCapacity_ZeroCapacityAllowed(t *testing.T) {
	mock := &mockBranchService{}
	handler := NewBranchHandler(mock, slog.Default())

	body, _ := json.Marshal(map[string]interface{}{"capacity": 0})
	req := httptest.NewRequest(http.MethodPatch, "/api/branches/1/capacity", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/branches/:id/capacity", handler.UpdateBranchCapacity)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mock.setCapacityCalled)
	assert.Equal(t, 0, mock.setCapacityArgs.capacity)
}

func TestBranchHandler_UpdateBranchCapacity_InvalidID(t *testing.T) {
	mock := &mockBranchService{}
	handler := NewBranchHandler(mock, slog.Default())

	body, _ := json.Marshal(map[string]interface{}{"capacity": 30})
	req := httptest.NewRequest(http.MethodPatch, "/api/branches/abc/capacity", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/branches/:id/capacity", handler.UpdateBranchCapacity)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.setCapacityCalled)
}

func TestBranchHandler_UpdateBranchCapacity_NotFound(t *testing.T) {
	mock := &mockBranchService{setCapacityErr: &shared.NotFoundError{Msg: "branch 99 not found"}}
	handler := NewBranchHandler(mock, slog.Default())

	body, _ := json.Marshal(map[string]interface{}{"capacity": 30})
	req := httptest.NewRequest(http.MethodPatch, "/api/branches/99/capacity", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/branches/:id/capacity", handler.UpdateBranchCapacity)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestBranchHandler_UpdateBranchCapacity_ValidationError(t *testing.T) {
	mock := &mockBranchService{setCapacityErr: &shared.ValidationError{Msg: "capacity must not be negative"}}
	handler := NewBranchHandler(mock, slog.Default())

	body, _ := json.Marshal(map[string]interface{}{"capacity": -1})
	req := httptest.NewRequest(http.MethodPatch, "/api/branches/1/capacity", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/branches/:id/capacity", handler.UpdateBranchCapacity)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBranchHandler_UpdateBranchCapacity_ServiceError(t *testing.T) {
	mock := &mockBranchService{setCapacityErr: assert.AnError}
	handler := NewBranchHandler(mock, slog.Default())

	body, _ := json.Marshal(map[string]interface{}{"capacity": 30})
	req := httptest.NewRequest(http.MethodPatch, "/api/branches/1/capacity", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/branches/:id/capacity", handler.UpdateBranchCapacity)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
