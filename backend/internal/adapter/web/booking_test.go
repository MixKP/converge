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
	result    *models.BookingResponse
	evalErr   error
	confirm   *models.Booking
	confErr   error
	cancelErr error
	listAll   []models.Booking
	listErr   error
}

func (m *mockBookingSvc) Evaluate(ctx context.Context, req models.BookingRequest) (*models.BookingResponse, error) {
	return m.result, m.evalErr
}

func (m *mockBookingSvc) Confirm(ctx context.Context, req models.ConfirmBookingRequest) (*models.Booking, error) {
	return m.confirm, m.confErr
}

func (m *mockBookingSvc) Cancel(ctx context.Context, bookingID int) error {
	return m.cancelErr
}

func (m *mockBookingSvc) ListAll(ctx context.Context) ([]models.Booking, error) {
	return m.listAll, m.listErr
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
	assert.Equal(t, http.StatusBadRequest, w.Code)
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

func TestConfirmBookingSuccess(t *testing.T) {
	mock := &mockBookingSvc{
		confirm: &models.Booking{
			ID:         10,
			TeacherID:  1,
			BranchID:   2,
			SubjectID:  3,
			StartTime:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
			EndTime:    time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
			ClientName: "John Doe",
		},
	}
	handler := NewBookingHandler(mock)

	payload := models.ConfirmBookingRequest{
		TeacherID:  1,
		BranchID:   2,
		SubjectID:  3,
		StartTime:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		ClientName: "John Doe",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/bookings/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings/confirm", handler.ConfirmBooking)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.Booking
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 10, response.ID)
	assert.Equal(t, "John Doe", response.ClientName)
}

func TestConfirmBookingValidationError(t *testing.T) {
	mock := &mockBookingSvc{
		confErr: &service.ValidationError{Msg: "client_name must not be empty"},
	}
	handler := NewBookingHandler(mock)

	payload := models.ConfirmBookingRequest{
		TeacherID: 1,
		BranchID:  2,
		SubjectID: 3,
		StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/bookings/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings/confirm", handler.ConfirmBooking)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConfirmBookingServiceError(t *testing.T) {
	mock := &mockBookingSvc{confErr: assert.AnError}
	handler := NewBookingHandler(mock)

	payload := models.ConfirmBookingRequest{
		TeacherID:  1,
		BranchID:   2,
		SubjectID:  3,
		StartTime:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		ClientName: "John Doe",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/bookings/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings/confirm", handler.ConfirmBooking)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to confirm booking")
}

func TestCancelBookingSuccess(t *testing.T) {
	mock := &mockBookingSvc{}
	handler := NewBookingHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.DELETE("/api/bookings/:id", handler.CancelBooking)

	req := httptest.NewRequest(http.MethodDelete, "/api/bookings/10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Booking cancelled successfully")
}

func TestCancelBookingNotFound(t *testing.T) {
	mock := &mockBookingSvc{
		cancelErr: &service.NotFoundError{Msg: "booking 999 not found"},
	}
	handler := NewBookingHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.DELETE("/api/bookings/:id", handler.CancelBooking)

	req := httptest.NewRequest(http.MethodDelete, "/api/bookings/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}

func TestCancelBookingInvalidID(t *testing.T) {
	mock := &mockBookingSvc{}
	handler := NewBookingHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.DELETE("/api/bookings/:id", handler.CancelBooking)

	req := httptest.NewRequest(http.MethodDelete, "/api/bookings/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid booking id")
}

func TestCancelBookingServiceError(t *testing.T) {
	mock := &mockBookingSvc{cancelErr: assert.AnError}
	handler := NewBookingHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.DELETE("/api/bookings/:id", handler.CancelBooking)

	req := httptest.NewRequest(http.MethodDelete, "/api/bookings/10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to cancel booking")
}

func TestListBookingsSuccess(t *testing.T) {
	mock := &mockBookingSvc{
		listAll: []models.Booking{
			{ID: 1, TeacherID: 1, ClientName: "John Doe", StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)},
			{ID: 2, TeacherID: 2, ClientName: "Jane Doe", StartTime: time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), EndTime: time.Date(2026, 6, 2, 11, 0, 0, 0, time.UTC)},
		},
	}
	handler := NewBookingHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/bookings", handler.ListBookings)

	req := httptest.NewRequest(http.MethodGet, "/api/bookings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var bookings []models.Booking
	err := json.Unmarshal(w.Body.Bytes(), &bookings)
	require.NoError(t, err)
	assert.Len(t, bookings, 2)
	assert.Equal(t, "John Doe", bookings[0].ClientName)
}

func TestListBookingsEmpty(t *testing.T) {
	mock := &mockBookingSvc{listAll: []models.Booking{}}
	handler := NewBookingHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/bookings", handler.ListBookings)

	req := httptest.NewRequest(http.MethodGet, "/api/bookings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]", w.Body.String())
}

func TestListBookingsServiceError(t *testing.T) {
	mock := &mockBookingSvc{listErr: assert.AnError}
	handler := NewBookingHandler(mock)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/bookings", handler.ListBookings)

	req := httptest.NewRequest(http.MethodGet, "/api/bookings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to list bookings")
}
