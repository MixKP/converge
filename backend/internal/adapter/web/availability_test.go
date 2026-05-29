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
	teachers         []models.Teacher
	getErr           error
	teachersBySub    []models.Teacher
	teachersBySubErr error
	branches         []models.Branch
	branchesErr      error
	subjects         []models.Subject
	subjectsErr      error
	availability     []models.TeacherAvailability
	availErr         error
	submitErr        error
	submitCalled     bool
}

func (m *mockService) GetActiveTeachers(ctx context.Context) ([]models.Teacher, error) {
	return m.teachers, m.getErr
}

func (m *mockService) GetTeachersBySubject(ctx context.Context, subjectID int) ([]models.Teacher, error) {
	return m.teachersBySub, m.teachersBySubErr
}

func (m *mockService) GetBranches(ctx context.Context) ([]models.Branch, error) {
	return m.branches, m.branchesErr
}

func (m *mockService) GetSubjects(ctx context.Context) ([]models.Subject, error) {
	return m.subjects, m.subjectsErr
}

func (m *mockService) GetAllAvailability(ctx context.Context) ([]models.TeacherAvailability, error) {
	return m.availability, m.availErr
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

func TestGetAllAvailabilitySuccess(t *testing.T) {
	mock := &mockService{
		availability: []models.TeacherAvailability{
			{
				Teacher: models.Teacher{ID: 1, Name: "Alice", Email: "alice@test.com"},
				Weekly: []models.WeeklySlot{
					{DayOfWeek: 0, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("12:00")},
					{DayOfWeek: 2, Start: models.TimeHHMM("09:00"), End: models.TimeHHMM("12:00")},
				},
			},
			{
				Teacher: models.Teacher{ID: 2, Name: "Bob", Email: "bob@test.com"},
				Weekly: []models.WeeklySlot{
					{DayOfWeek: 1, Start: models.TimeHHMM("10:00"), End: models.TimeHHMM("15:00")},
				},
			},
		},
	}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/availability", handler.GetAllAvailability)

	req := httptest.NewRequest(http.MethodGet, "/api/availability", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var availability []models.TeacherAvailability
	err := json.Unmarshal(w.Body.Bytes(), &availability)
	require.NoError(t, err)
	assert.Len(t, availability, 2)
	assert.Equal(t, "Alice", availability[0].Teacher.Name)
	assert.Len(t, availability[0].Weekly, 2)
	assert.Equal(t, "Bob", availability[1].Teacher.Name)
	assert.Len(t, availability[1].Weekly, 1)
}

func TestGetAllAvailabilityError(t *testing.T) {
	mock := &mockService{availErr: assert.AnError}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/availability", handler.GetAllAvailability)

	req := httptest.NewRequest(http.MethodGet, "/api/availability", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to retrieve availability")
}

func TestGetAllAvailabilityEmpty(t *testing.T) {
	mock := &mockService{
		availability: []models.TeacherAvailability{},
	}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/availability", handler.GetAllAvailability)

	req := httptest.NewRequest(http.MethodGet, "/api/availability", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var availability []models.TeacherAvailability
	err := json.Unmarshal(w.Body.Bytes(), &availability)
	require.NoError(t, err)
	assert.Len(t, availability, 0)
}

func TestGetTeachersBySubjectSuccess(t *testing.T) {
	mock := &mockService{
		teachersBySub: []models.Teacher{
			{ID: 1, Name: "Alice", Email: "alice@test.com"},
			{ID: 3, Name: "Carol", Email: "carol@test.com"},
		},
	}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers?subject_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var teachers []models.Teacher
	err := json.Unmarshal(w.Body.Bytes(), &teachers)
	require.NoError(t, err)
	assert.Len(t, teachers, 2)
	assert.Equal(t, "Alice", teachers[0].Name)
}

func TestGetTeachersBySubjectInvalidID(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers?subject_id=invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid subject_id")
}

func TestGetTeachersBySubjectNegativeID(t *testing.T) {
	mock := &mockService{}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers?subject_id=-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid subject_id")
}

func TestGetTeachersBySubjectError(t *testing.T) {
	mock := &mockService{teachersBySubErr: assert.AnError}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/teachers", handler.GetTeachers)

	req := httptest.NewRequest(http.MethodGet, "/api/teachers?subject_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to retrieve teachers")
}

func TestGetBranchesSuccess(t *testing.T) {
	mock := &mockService{
		branches: []models.Branch{
			{ID: 1, Name: "Main Campus"},
			{ID: 2, Name: "Downtown"},
		},
	}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/branches", handler.GetBranches)

	req := httptest.NewRequest(http.MethodGet, "/api/branches", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var branches []models.Branch
	err := json.Unmarshal(w.Body.Bytes(), &branches)
	require.NoError(t, err)
	assert.Len(t, branches, 2)
	assert.Equal(t, "Main Campus", branches[0].Name)
}

func TestGetBranchesError(t *testing.T) {
	mock := &mockService{branchesErr: assert.AnError}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/branches", handler.GetBranches)

	req := httptest.NewRequest(http.MethodGet, "/api/branches", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to retrieve branches")
}

func TestGetSubjectsSuccess(t *testing.T) {
	mock := &mockService{
		subjects: []models.Subject{
			{ID: 1, Name: "Mathematics"},
			{ID: 2, Name: "Physics"},
		},
	}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/subjects", handler.GetSubjects)

	req := httptest.NewRequest(http.MethodGet, "/api/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var subjects []models.Subject
	err := json.Unmarshal(w.Body.Bytes(), &subjects)
	require.NoError(t, err)
	assert.Len(t, subjects, 2)
	assert.Equal(t, "Mathematics", subjects[0].Name)
}

func TestGetSubjectsError(t *testing.T) {
	mock := &mockService{subjectsErr: assert.AnError}
	handler := NewAvailabilityHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/subjects", handler.GetSubjects)

	req := httptest.NewRequest(http.MethodGet, "/api/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to retrieve subjects")
}
