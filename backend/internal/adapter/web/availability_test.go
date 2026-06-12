package web

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/RinZ5/converge/backend/internal/teacher"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	teachers         []teacher.Teacher
	getErr           error
	teachersBySub    []teacher.Teacher
	teachersBySubErr error
	branches         []shared.Branch
	branchesErr      error
	subjects         []shared.Subject
	subjectsErr      error
	availability     []teacher.TeacherAvailability
	availErr         error
	submitErr        error
	submitCalled     bool
}

func (m *mockService) GetActiveTeachers(ctx context.Context) ([]teacher.Teacher, error) {
	return m.teachers, m.getErr
}

func (m *mockService) GetTeachersBySubject(ctx context.Context, subjectID int) ([]teacher.Teacher, error) {
	return m.teachersBySub, m.teachersBySubErr
}

func (m *mockService) GetBranches(ctx context.Context) ([]shared.Branch, error) {
	return m.branches, m.branchesErr
}

func (m *mockService) GetSubjects(ctx context.Context) ([]shared.Subject, error) {
	return m.subjects, m.subjectsErr
}

func (m *mockService) GetAllAvailability(ctx context.Context) ([]teacher.TeacherAvailability, error) {
	return m.availability, m.availErr
}

func (m *mockService) SubmitWeeklyAvailability(ctx context.Context, teacherID int, slots []shared.WeeklySlot) error {
	m.submitCalled = true
	return m.submitErr
}

func TestAvailabilityHandler_GetTeachers_Success(t *testing.T) {
	mock := &mockService{
		teachers: []teacher.Teacher{
			{ID: 1, Name: "Alice", Email: "alice@test.com"},
			{ID: 2, Name: "Bob", Email: "bob@test.com"},
		},
	}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var teachers []teacher.Teacher
	err := json.Unmarshal(w.Body.Bytes(), &teachers)
	require.NoError(t, err)
	assert.Len(t, teachers, 2)
	assert.Equal(t, "alice@test.com", teachers[0].Email)
	assert.Equal(t, "bob@test.com", teachers[1].Email)
}

func TestAvailabilityHandler_GetTeachers_Error(t *testing.T) {
	mock := &mockService{getErr: assert.AnError}
	handler := NewAvailabilityHandler(mock, slog.Default())
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAvailabilityHandler_SubmitWeeklyAvailability_Success(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{
		"teacher_id": 42,
		"weekly": []map[string]interface{}{
			{"day_of_week": 0, "start": "09:00", "end": "10:00"},
			{"day_of_week": 0, "start": "11:00", "end": "12:00"},
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

func TestAvailabilityHandler_SubmitWeeklyAvailability_Overlap(t *testing.T) {
	mock := &mockService{
		submitErr: &teacher.ValidationError{
			Msg: "overlapping slots on day 0: 09:00-11:00 and 10:30-12:00",
		},
	}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{
		"teacher_id": 1,
		"weekly": []map[string]interface{}{
			{"day_of_week": 0, "start": "09:00", "end": "11:00"},
			{"day_of_week": 0, "start": "10:30", "end": "12:00"},
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

func TestAvailabilityHandler_SubmitWeeklyAvailability_InvalidJSON(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())
	r := gin.Default()
	r.POST("/api/availability", handler.SubmitWeeklyAvailability)

	req := httptest.NewRequest(http.MethodPost, "/api/availability", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.submitCalled)
}

func TestAvailabilityHandler_SubmitWeeklyAvailability_ServiceError(t *testing.T) {
	mock := &mockService{submitErr: assert.AnError}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{
		"teacher_id": 1,
		"weekly": []map[string]interface{}{
			{"day_of_week": 0, "start": "09:00", "end": "10:00"},
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

func TestAvailabilityHandler_SubmitWeeklyAvailability_EmptyBody(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())

	req := httptest.NewRequest(http.MethodPost, "/api/availability", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.Default()
	r.POST("/api/availability", handler.SubmitWeeklyAvailability)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.submitCalled)
}

func TestAvailabilityHandler_GetTeachers_EmptyList(t *testing.T) {
	mock := &mockService{
		teachers: []teacher.Teacher{},
	}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var teachers []teacher.Teacher
	err := json.Unmarshal(w.Body.Bytes(), &teachers)
	require.NoError(t, err)
	assert.Len(t, teachers, 0)
}

func TestAvailabilityHandler_GetAllAvailability_Success(t *testing.T) {
	mock := &mockService{
		availability: []teacher.TeacherAvailability{
			{
				Teacher: teacher.Teacher{ID: 1, Name: "Alice"},
				Weekly: []shared.WeeklySlot{
					{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("12:00")},
					{DayOfWeek: 2, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("12:00")},
				},
			},
			{
				Teacher: teacher.Teacher{ID: 2, Name: "Bob"},
				Weekly: []shared.WeeklySlot{
					{DayOfWeek: 1, Start: shared.TimeHHMM("10:00"), End: shared.TimeHHMM("15:00")},
				},
			},
		},
	}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/availability", handler.GetAllAvailability)

	req := httptest.NewRequest(http.MethodGet, "/api/availability", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var availability []teacher.TeacherAvailability
	err := json.Unmarshal(w.Body.Bytes(), &availability)
	require.NoError(t, err)
	assert.Len(t, availability, 2)
	assert.Equal(t, "Alice", availability[0].Teacher.Name)
	assert.Len(t, availability[0].Weekly, 2)
	assert.Equal(t, "Bob", availability[1].Teacher.Name)
	assert.Len(t, availability[1].Weekly, 1)
}

func TestAvailabilityHandler_GetAllAvailability_Error(t *testing.T) {
	mock := &mockService{availErr: assert.AnError}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/availability", handler.GetAllAvailability)

	req := httptest.NewRequest(http.MethodGet, "/api/availability", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to retrieve availability")
}

func TestAvailabilityHandler_GetAllAvailability_Empty(t *testing.T) {
	mock := &mockService{
		availability: []teacher.TeacherAvailability{},
	}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/availability", handler.GetAllAvailability)

	req := httptest.NewRequest(http.MethodGet, "/api/availability", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var availability []teacher.TeacherAvailability
	err := json.Unmarshal(w.Body.Bytes(), &availability)
	require.NoError(t, err)
	assert.Len(t, availability, 0)
}

func TestAvailabilityHandler_GetTeachersBySubject_Success(t *testing.T) {
	mock := &mockService{
		teachersBySub: []teacher.Teacher{
			{ID: 1, Name: "Alice", Email: "alice@test.com"},
			{ID: 3, Name: "Carol", Email: "carol@test.com"},
		},
	}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers?subject_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var teachers []teacher.Teacher
	err := json.Unmarshal(w.Body.Bytes(), &teachers)
	require.NoError(t, err)
	assert.Len(t, teachers, 2)
	assert.Equal(t, "Alice", teachers[0].Name)
}

func TestAvailabilityHandler_GetTeachersBySubject_InvalidID(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers?subject_id=invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid subject_id")
}

func TestAvailabilityHandler_GetTeachersBySubject_NegativeID(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers?subject_id=-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid subject_id")
}

func TestAvailabilityHandler_GetTeachersBySubject_Error(t *testing.T) {
	mock := &mockService{teachersBySubErr: assert.AnError}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers?subject_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to retrieve teachers")
}

func TestAvailabilityHandler_GetBranches_Success(t *testing.T) {
	mock := &mockService{
		branches: []shared.Branch{
			{ID: 1, Name: "Main Campus"},
			{ID: 2, Name: "Downtown"},
		},
	}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/branches", handler.GetBranches)

	req := httptest.NewRequest(http.MethodGet, "/api/branches", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var branches []shared.Branch
	err := json.Unmarshal(w.Body.Bytes(), &branches)
	require.NoError(t, err)
	assert.Len(t, branches, 2)
	assert.Equal(t, "Main Campus", branches[0].Name)
}

func TestAvailabilityHandler_GetBranches_Error(t *testing.T) {
	mock := &mockService{branchesErr: assert.AnError}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/branches", handler.GetBranches)

	req := httptest.NewRequest(http.MethodGet, "/api/branches", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to retrieve branches")
}

func TestAvailabilityHandler_GetSubjects_Success(t *testing.T) {
	mock := &mockService{
		subjects: []shared.Subject{
			{ID: 1, Name: "Mathematics"},
			{ID: 2, Name: "Physics"},
		},
	}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/subjects", handler.GetSubjects)

	req := httptest.NewRequest(http.MethodGet, "/api/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var subjects []shared.Subject
	err := json.Unmarshal(w.Body.Bytes(), &subjects)
	require.NoError(t, err)
	assert.Len(t, subjects, 2)
	assert.Equal(t, "Mathematics", subjects[0].Name)
}

func TestAvailabilityHandler_GetSubjects_Error(t *testing.T) {
	mock := &mockService{subjectsErr: assert.AnError}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/subjects", handler.GetSubjects)

	req := httptest.NewRequest(http.MethodGet, "/api/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to retrieve subjects")
}
