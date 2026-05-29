package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	teachers     []models.Teacher
	getErr       error
	submitErr    error
	submitCalled bool
}

func (m *mockService) GetActiveTeachers(ctx context.Context) ([]models.Teacher, error) {
	return m.teachers, m.getErr
}

func (m *mockService) SubmitWeeklyAvailability(ctx context.Context, payload models.AvailabilityPayload) error {
	m.submitCalled = true
	return m.submitErr
}

func TestGetTeachersSuccess(t *testing.T) {
	mock := &mockService{
		teachers: []models.Teacher{
			{ID: 1, Name: "Alice", Email: "alice@test.com"},
			{ID: 2, Name: "Bob", Email: "bob@test.com"},
		},
	}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var teachers []models.Teacher
	err := json.Unmarshal(w.Body.Bytes(), &teachers)
	require.NoError(t, err)
	assert.Len(t, teachers, 2)
	assert.Equal(t, "alice@test.com", teachers[0].Email)
	assert.Equal(t, "bob@test.com", teachers[1].Email)
}

func TestGetTeachersError(t *testing.T) {
	mock := &mockService{getErr: assert.AnError}
	handler := NewAvailabilityHandler(mock)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubmitWeeklyAvailabilitySuccess(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 42,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
			{DayOfWeek: 0, Start: models.TimeHHMM("11:00"), End: models.TimeHHMM("12:00")},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/availability", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r := gin.Default()
	r.POST("/api/availability", handler.SubmitWeeklyAvailability)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, mock.submitCalled)
}

func TestSubmitWeeklyAvailabilityOverlap(t *testing.T) {
	mock := &mockService{
		submitErr: &service.ValidationError{
			Msg: "overlapping slots on day 0: 09:00-11:00 and 10:30-12:00",
		},
	}
	handler := NewAvailabilityHandler(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("11:00")},
			{DayOfWeek: 0, Start: models.TimeHHMM("10:30"), End: models.TimeHHMM("12:00")},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/availability", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r := gin.Default()
	r.POST("/api/availability", handler.SubmitWeeklyAvailability)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "overlapping")
}

func TestSubmitWeeklyAvailabilityInvalidJSON(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock)
	r := gin.Default()
	r.POST("/api/availability", handler.SubmitWeeklyAvailability)

	req := httptest.NewRequest(http.MethodPost, "/api/availability", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.submitCalled)
}

func TestSubmitWeeklyAvailabilityServiceError(t *testing.T) {
	mock := &mockService{submitErr: assert.AnError}
	handler := NewAvailabilityHandler(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("10:00")},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/availability", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r := gin.Default()
	r.POST("/api/availability", handler.SubmitWeeklyAvailability)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to save availability")
	assert.True(t, mock.submitCalled)
}

func TestSubmitWeeklyAvailabilityEmptyBody(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/availability", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.Default()
	r.POST("/api/availability", handler.SubmitWeeklyAvailability)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.submitCalled)
}

func TestGetTeachersEmptyList(t *testing.T) {
	mock := &mockService{
		teachers: []models.Teacher{},
	}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var teachers []models.Teacher
	err := json.Unmarshal(w.Body.Bytes(), &teachers)
	require.NoError(t, err)
	assert.Len(t, teachers, 0)
}
