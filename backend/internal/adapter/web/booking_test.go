package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBookingSvc struct {
	result  *models.BookingResponse
	evalErr error
}

func (m *mockBookingSvc) Evaluate(ctx context.Context, req models.BookingRequest) (*models.BookingResponse, error) {
	return m.result, m.evalErr
}

func TestCreateBookingExactMatch(t *testing.T) {
	mock := &mockBookingSvc{
		result: &models.BookingResponse{
			ExactMatch: &models.BookingAlternative{
				TeacherID:   1,
				TeacherName: "Alice",
				BranchID:    2,
				SubjectID:   3,
				StartTime:   time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
				EndTime:     time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
				Score:       100,
				Reasons:     []string{"Exact match"},
			},
			Message: "Exact match found",
		},
	}
	handler := NewBookingHandler(mock)

	payload := models.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredStart:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		DurationMinutes: 60,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings", handler.CreateBooking)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.BookingResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response.ExactMatch)
	assert.Equal(t, "Alice", response.ExactMatch.TeacherName)
	assert.Equal(t, 100, response.ExactMatch.Score)
	assert.Contains(t, response.Message, "Exact match found")
}

func TestCreateBookingAlternatives(t *testing.T) {
	mock := &mockBookingSvc{
		result: &models.BookingResponse{
			Alternatives: []models.BookingAlternative{
				{
					TeacherID:   2,
					TeacherName: "Bob",
					BranchID:    2,
					SubjectID:   3,
					StartTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
					EndTime:     time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC),
					Score:       85,
					Reasons:     []string{"Within 1hr – off by 60m"},
				},
				{
					TeacherID:   3,
					TeacherName: "Charlie",
					BranchID:    2,
					SubjectID:   3,
					StartTime:   time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
					EndTime:     time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
					Score:       75,
					Reasons:     []string{"Different teacher"},
				},
			},
			Message: "No exact match found. 2 alternative(s) returned. Room availability not checked.",
		},
	}
	handler := NewBookingHandler(mock)

	payload := models.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredStart:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		DurationMinutes: 60,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings", handler.CreateBooking)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.BookingResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Nil(t, response.ExactMatch)
	assert.Len(t, response.Alternatives, 2)
	assert.Equal(t, "Bob", response.Alternatives[0].TeacherName)
	assert.Equal(t, 85, response.Alternatives[0].Score)
	assert.Contains(t, response.Message, "Room availability not checked")
}

func TestCreateBookingEmptyAlternatives(t *testing.T) {
	mock := &mockBookingSvc{
		result: &models.BookingResponse{
			Message: "No exact match found. No alternatives available.",
		},
	}
	handler := NewBookingHandler(mock)

	payload := models.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredStart:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		DurationMinutes: 60,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings", handler.CreateBooking)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "No alternatives available")
}

func TestCreateBookingValidationError(t *testing.T) {
	mock := &mockBookingSvc{
		evalErr: &service.ValidationError{Msg: "subject_id must be positive"},
	}
	handler := NewBookingHandler(mock)

	payload := models.BookingRequest{
		SubjectID:       0,
		BranchID:        2,
		PreferredStart:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		DurationMinutes: 60,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings", handler.CreateBooking)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "subject_id must be positive")
}

func TestCreateBookingInvalidJSON(t *testing.T) {
	mock := &mockBookingSvc{}
	handler := NewBookingHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings", handler.CreateBooking)

	req := httptest.NewRequest(http.MethodPost, "/api/bookings", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateBookingServiceError(t *testing.T) {
	mock := &mockBookingSvc{evalErr: assert.AnError}
	handler := NewBookingHandler(mock)

	payload := models.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredStart:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		DurationMinutes: 60,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings", handler.CreateBooking)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to evaluate booking")
}

func TestCreateBookingOptionalPreferredTeacher(t *testing.T) {
	mock := &mockBookingSvc{
		result: &models.BookingResponse{
			Alternatives: []models.BookingAlternative{
				{
					TeacherID:   1,
					TeacherName: "Alice",
					BranchID:    2,
					SubjectID:   3,
					StartTime:   time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
					EndTime:     time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
					Score:       80,
					Reasons:     []string{"ok"},
				},
			},
			Message: "No exact match found. 1 alternative(s) returned. Room availability not checked.",
		},
	}
	handler := NewBookingHandler(mock)

	preferredTeacher := 5
	payload := models.BookingRequest{
		SubjectID:          3,
		BranchID:           2,
		PreferredStart:     time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		DurationMinutes:    60,
		PreferredTeacherID: &preferredTeacher,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings", handler.CreateBooking)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response models.BookingResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response.Alternatives, 1)
}
