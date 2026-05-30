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

func slot(day int, start, end string) models.WeeklySlot {
	return models.WeeklySlot{DayOfWeek: day, Start: models.TimeHHMM(start), End: models.TimeHHMM(end)}
}

func TestCreateBookingExactMatch(t *testing.T) {
	mock := &mockBookingSvc{
		result: &models.BookingResponse{
			Results: []models.SlotResult{{
				Slot: slot(0, "09:00", "10:00"),
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
			}},
		},
	}
	handler := NewBookingHandler(mock)

	payload := models.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredSlots:  []models.WeeklySlot{slot(0, "09:00", "10:00")},
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
	assert.Len(t, response.Results, 1)
	assert.NotNil(t, response.Results[0].ExactMatch)
	assert.Equal(t, "Alice", response.Results[0].ExactMatch.TeacherName)
	assert.Equal(t, 100, response.Results[0].ExactMatch.Score)
	assert.Contains(t, response.Results[0].Message, "Exact match found")
}

func TestCreateBookingAlternatives(t *testing.T) {
	mock := &mockBookingSvc{
		result: &models.BookingResponse{
			Results: []models.SlotResult{{
				Slot: slot(0, "09:00", "10:00"),
				Alternatives: []models.BookingAlternative{{
					TeacherID:   2,
					TeacherName: "Bob",
					BranchID:    2,
					SubjectID:   3,
					StartTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
					EndTime:     time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC),
					Score:       85,
					Reasons:     []string{"Within 1hr of window"},
				}},
				Message: "No exact match found. 1 alternative(s) returned.",
			}},
		},
	}
	handler := NewBookingHandler(mock)

	payload := models.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredSlots:  []models.WeeklySlot{slot(0, "09:00", "10:00")},
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
	assert.Len(t, response.Results, 1)
	assert.Nil(t, response.Results[0].ExactMatch)
	assert.Len(t, response.Results[0].Alternatives, 1)
}

func TestCreateBookingEmptyAlternatives(t *testing.T) {
	mock := &mockBookingSvc{
		result: &models.BookingResponse{
			Results: []models.SlotResult{{
				Slot:    slot(0, "09:00", "10:00"),
				Message: "No exact match found. No alternatives available.",
			}},
		},
	}
	handler := NewBookingHandler(mock)

	payload := models.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredSlots:  []models.WeeklySlot{slot(0, "09:00", "10:00")},
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
		evalErr: &service.ValidationError{Msg: "preferred_slots must not be empty"},
	}
	handler := NewBookingHandler(mock)

	payload := models.BookingRequest{
		SubjectID: 0,
		BranchID:  2,
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
	assert.Contains(t, w.Body.String(), "preferred_slots must not be empty")
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
		PreferredSlots:  []models.WeeklySlot{slot(0, "09:00", "10:00")},
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
			Results: []models.SlotResult{{
				Slot: slot(0, "09:00", "10:00"),
				Alternatives: []models.BookingAlternative{{
					TeacherID:   1,
					TeacherName: "Alice",
					BranchID:    2,
					SubjectID:   3,
					StartTime:   time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
					EndTime:     time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
					Score:       80,
					Reasons:     []string{"ok"},
				}},
				Message: "No exact match found. 1 alternative(s) returned.",
			}},
		},
	}
	handler := NewBookingHandler(mock)

	preferredTeacher := 5
	payload := models.BookingRequest{
		SubjectID:          3,
		BranchID:           2,
		PreferredSlots:     []models.WeeklySlot{slot(0, "09:00", "10:00")},
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
	assert.Len(t, response.Results, 1)
}
