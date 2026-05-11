package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	teachers      []models.Teacher
	getErr        error
	replaceCalled bool
	replaceErr    error
	saveCalled    bool
	saveErr       error
}

func (m *mockRepo) GetActiveTeachers(ctx context.Context) ([]models.Teacher, error) {
	return m.teachers, m.getErr
}
func (m *mockRepo) ReplaceWeeklyAvailability(ctx context.Context, teacherID int, slots []models.WeeklySlot) error {
	m.replaceCalled = true
	return m.replaceErr
}
func (m *mockRepo) SaveRawSubmission(ctx context.Context, teacherID int, rawPayload []byte) error {
	m.saveCalled = true
	return m.saveErr
}

func TestGetTeachersSuccess(t *testing.T) {
	mock := &mockRepo{
		teachers: []models.Teacher{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}},
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
}

func TestGetTeachersError(t *testing.T) {
	mock := &mockRepo{getErr: assert.AnError}
	handler := NewAvailabilityHandler(mock)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubmitWeeklyAvailabilitySuccess(t *testing.T) {
	mock := &mockRepo{}
	handler := NewAvailabilityHandler(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 42,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 1, Start: "09:00", End: "10:00"},
			{DayOfWeek: 1, Start: "11:00", End: "12:00"},
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
	assert.True(t, mock.replaceCalled)
	assert.True(t, mock.saveCalled)
}

func TestSubmitWeeklyAvailabilityOverlap(t *testing.T) {
	mock := &mockRepo{}
	handler := NewAvailabilityHandler(mock)

	payload := models.AvailabilityPayload{
		TeacherID: 1,
		Weekly: []models.WeeklySlot{
			{DayOfWeek: 1, Start: "09:00", End: "11:00"},
			{DayOfWeek: 1, Start: "10:30", End: "12:00"},
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
	assert.False(t, mock.replaceCalled)
}

func TestSubmitWeeklyAvailabilityInvalidJSON(t *testing.T) {
	mock := &mockRepo{}
	handler := NewAvailabilityHandler(mock)
	r := gin.Default()
	r.POST("/api/availability", handler.SubmitWeeklyAvailability)

	req := httptest.NewRequest(http.MethodPost, "/api/availability", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.replaceCalled)
}
