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
	activeTeachers   []teacher.Teacher
	activeErr        error
	inactiveTeachers []teacher.Teacher
	inactiveErr      error
	teachersBySub    []teacher.Teacher
	teachersBySubErr error
	subjects         []shared.Subject
	subjectsErr      error
	availability     []teacher.TeacherAvailability
	availErr         error
	submitErr        error
	submitDeleted    int
	submitTeacherID  int
	submitCalled     bool
	addedTeacher     *teacher.Teacher
	addTeacherErr    error
	addTeacherCalled bool
	setStatusErr     error
	setStatusCalled  bool
	setGenderErr     error
	setGenderCalled  bool

	setSubjectsErr    error
	setSubjectsCalled bool
	setSubjectsIDs    []int
}

func (m *mockService) GetAllTeachers(ctx context.Context) ([]teacher.Teacher, error) {
	return m.teachers, m.getErr
}

func (m *mockService) GetActiveTeachers(ctx context.Context) ([]teacher.Teacher, error) {
	return m.activeTeachers, m.activeErr
}

func (m *mockService) GetInactiveTeachers(ctx context.Context) ([]teacher.Teacher, error) {
	return m.inactiveTeachers, m.inactiveErr
}

func (m *mockService) GetTeachersBySubject(ctx context.Context, subjectID int) ([]teacher.Teacher, error) {
	return m.teachersBySub, m.teachersBySubErr
}

func (m *mockService) GetSubjects(ctx context.Context) ([]shared.Subject, error) {
	return m.subjects, m.subjectsErr
}

func (m *mockService) GetAllAvailability(ctx context.Context) ([]teacher.TeacherAvailability, error) {
	return m.availability, m.availErr
}

func (m *mockService) SubmitWeeklyAvailability(ctx context.Context, teacherID int, slots []shared.WeeklySlot) (int, error) {
	m.submitCalled = true
	m.submitTeacherID = teacherID
	return m.submitDeleted, m.submitErr
}

func (m *mockService) AddTeacher(ctx context.Context, name, email, gender string) (*teacher.Teacher, error) {
	m.addTeacherCalled = true
	return m.addedTeacher, m.addTeacherErr
}

func (m *mockService) SetStatus(ctx context.Context, teacherID int, status string) error {
	m.setStatusCalled = true
	return m.setStatusErr
}

func (m *mockService) SetGender(ctx context.Context, teacherID int, gender string) error {
	m.setGenderCalled = true
	return m.setGenderErr
}

func (m *mockService) SubjectsForTeacher(ctx context.Context, teacherID int) ([]shared.Subject, error) {
	return m.subjects, m.subjectsErr
}

func (m *mockService) SetTeacherSubjects(ctx context.Context, teacherID int, subjectIDs []int) error {
	m.setSubjectsCalled = true
	m.setSubjectsIDs = subjectIDs
	return m.setSubjectsErr
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

func TestAvailabilityHandler_SubmitWeeklyAvailability_InvalidTeacherID(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())
	r := gin.Default()
	r.PUT("/api/teachers/:id/availability", handler.SubmitWeeklyAvailability)

	req := httptest.NewRequest(http.MethodPut, "/api/teachers/abc/availability", bytes.NewReader([]byte(`{"weekly":[]}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.submitCalled)
}

func TestAvailabilityHandler_SubmitWeeklyAvailability_TeacherNotFound(t *testing.T) {
	mock := &mockService{submitErr: &shared.NotFoundError{Msg: "teacher 99 not found"}}
	handler := NewAvailabilityHandler(mock, slog.Default())
	r := gin.Default()
	r.PUT("/api/teachers/:id/availability", handler.SubmitWeeklyAvailability)

	body := `{"weekly":[{"day_of_week":0,"start":"09:00","end":"10:00"}]}`
	req := httptest.NewRequest(http.MethodPut, "/api/teachers/99/availability", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "teacher 99 not found")
}

func TestAvailabilityHandler_SubmitWeeklyAvailability_Overlap(t *testing.T) {
	mock := &mockService{
		submitErr: &teacher.ValidationError{
			Msg: "overlapping slots on day 0: 09:00-11:00 and 10:30-12:00",
		},
	}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{
		"weekly": []map[string]interface{}{
			{"day_of_week": 0, "start": "09:00", "end": "11:00"},
			{"day_of_week": 0, "start": "10:30", "end": "12:00"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/teachers/1/availability", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r := gin.Default()
	r.PUT("/api/teachers/:id/availability", handler.SubmitWeeklyAvailability)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "overlapping")
}

func TestAvailabilityHandler_SubmitWeeklyAvailability_InvalidJSON(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())
	r := gin.Default()
	r.PUT("/api/teachers/:id/availability", handler.SubmitWeeklyAvailability)

	req := httptest.NewRequest(http.MethodPut, "/api/teachers/1/availability", bytes.NewReader([]byte(`{`)))
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
		"weekly": []map[string]interface{}{
			{"day_of_week": 0, "start": "09:00", "end": "10:00"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/teachers/1/availability", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r := gin.Default()
	r.PUT("/api/teachers/:id/availability", handler.SubmitWeeklyAvailability)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to save availability")
	assert.True(t, mock.submitCalled)
}

func TestAvailabilityHandler_SubmitWeeklyAvailability_EmptyBody(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())

	req := httptest.NewRequest(http.MethodPut, "/api/teachers/1/availability", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.Default()
	r.PUT("/api/teachers/:id/availability", handler.SubmitWeeklyAvailability)

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

func TestAvailabilityHandler_GetTeachers_StatusAll(t *testing.T) {
	mock := &mockService{
		teachers: []teacher.Teacher{
			{ID: 1, Name: "Alice", Status: "active"},
			{ID: 2, Name: "Bob", Status: "deactivated"},
		},
	}
	handler := NewAvailabilityHandler(mock, slog.Default())
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers?status=all", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var teachers []teacher.Teacher
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &teachers))
	assert.Len(t, teachers, 2)
}

func TestAvailabilityHandler_GetTeachers_StatusInvalid(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers?status=bogus", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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

func TestAvailabilityHandler_CreateTeacher_MissingGender(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{
		"name":  "Alice",
		"email": "alice@test.com",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/teachers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/teachers", handler.CreateTeacher)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.addTeacherCalled)
}

func TestAvailabilityHandler_CreateTeacher_InvalidJSON(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/teachers", handler.CreateTeacher)

	req := httptest.NewRequest(http.MethodPost, "/api/teachers", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.addTeacherCalled)
}

func TestAvailabilityHandler_CreateTeacher_ServiceError(t *testing.T) {
	mock := &mockService{addTeacherErr: assert.AnError}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{
		"name":   "Alice",
		"email":  "alice@test.com",
		"gender": "female",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/teachers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/teachers", handler.CreateTeacher)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to create teacher")
	assert.True(t, mock.addTeacherCalled)
}

func TestAvailabilityHandler_CreateTeacher_ValidationError(t *testing.T) {
	mock := &mockService{addTeacherErr: &teacher.ValidationError{Msg: "invalid gender"}}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{
		"name":   "Alice",
		"email":  "alice@test.com",
		"gender": "robot",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/teachers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/teachers", handler.CreateTeacher)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid gender")
}

func TestAvailabilityHandler_UpdateTeacherStatus_InvalidID(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{"status": "active"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/teachers/abc/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/teachers/:id/status", handler.UpdateTeacherStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.setStatusCalled)
}

func TestAvailabilityHandler_UpdateTeacherStatus_MissingStatus(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())

	req := httptest.NewRequest(http.MethodPatch, "/api/teachers/1/status", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/teachers/:id/status", handler.UpdateTeacherStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.setStatusCalled)
}

func TestAvailabilityHandler_UpdateTeacherStatus_NotFound(t *testing.T) {
	mock := &mockService{setStatusErr: &shared.NotFoundError{Msg: "teacher 99 not found"}}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{"status": "active"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/teachers/99/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/teachers/:id/status", handler.UpdateTeacherStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, mock.setStatusCalled)
}

func TestAvailabilityHandler_UpdateTeacherStatus_ServiceError(t *testing.T) {
	mock := &mockService{setStatusErr: assert.AnError}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{"status": "active"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/teachers/1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/teachers/:id/status", handler.UpdateTeacherStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to update teacher status")
	assert.True(t, mock.setStatusCalled)
}

func TestAvailabilityHandler_UpdateTeacherStatus_ValidationError(t *testing.T) {
	mock := &mockService{setStatusErr: &teacher.ValidationError{Msg: "invalid status"}}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{"status": "robot"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/teachers/1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/teachers/:id/status", handler.UpdateTeacherStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid status")
}

func TestAvailabilityHandler_UpdateTeacherGender_InvalidID(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{"gender": "female"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/teachers/abc/gender", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/teachers/:id/gender", handler.UpdateTeacherGender)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.setGenderCalled)
}

func TestAvailabilityHandler_UpdateTeacherGender_MissingGender(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock, slog.Default())

	req := httptest.NewRequest(http.MethodPatch, "/api/teachers/1/gender", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/teachers/:id/gender", handler.UpdateTeacherGender)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.setGenderCalled)
}

func TestAvailabilityHandler_UpdateTeacherGender_NotFound(t *testing.T) {
	mock := &mockService{setGenderErr: &shared.NotFoundError{Msg: "teacher 99 not found"}}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{"gender": "female"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/teachers/99/gender", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/teachers/:id/gender", handler.UpdateTeacherGender)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, mock.setGenderCalled)
}

func TestAvailabilityHandler_UpdateTeacherGender_ValidationError(t *testing.T) {
	mock := &mockService{setGenderErr: &teacher.ValidationError{Msg: "invalid gender"}}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{"gender": "robot"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/teachers/1/gender", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/teachers/:id/gender", handler.UpdateTeacherGender)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid gender")
}

func TestAvailabilityHandler_UpdateTeacherGender_ServiceError(t *testing.T) {
	mock := &mockService{setGenderErr: assert.AnError}
	handler := NewAvailabilityHandler(mock, slog.Default())

	payload := map[string]interface{}{"gender": "female"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/teachers/1/gender", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PATCH("/api/teachers/:id/gender", handler.UpdateTeacherGender)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to update teacher gender")
	assert.True(t, mock.setGenderCalled)
}
