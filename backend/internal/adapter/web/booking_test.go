package web

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/scheduling"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBookingSvc struct {
	result       *scheduling.BookingResponse
	evalErr      error
	confirm      *scheduling.Booking
	confErr      error
	cancelErr    error
	listAll      []scheduling.Booking
	listErr      error
	gotStudentID []int
}

func (m *mockBookingSvc) Evaluate(ctx context.Context, req scheduling.BookingRequest) (*scheduling.BookingResponse, error) {
	return m.result, m.evalErr
}

func (m *mockBookingSvc) Confirm(ctx context.Context, req scheduling.ConfirmBookingRequest) (*scheduling.Booking, error) {
	return m.confirm, m.confErr
}

func (m *mockBookingSvc) Cancel(ctx context.Context, bookingID int) error {
	return m.cancelErr
}

func (m *mockBookingSvc) ListAll(ctx context.Context) ([]scheduling.Booking, error) {
	return m.listAll, m.listErr
}

func (m *mockBookingSvc) ListForStudentIDs(ctx context.Context, studentIDs []int) ([]scheduling.Booking, error) {
	m.gotStudentID = studentIDs
	return m.listAll, m.listErr
}

type mockGuardian struct {
	ids []int
	err error
}

func (m *mockGuardian) StudentIDsForParent(ctx context.Context, parentID int) ([]int, error) {
	return m.ids, m.err
}

// listRouter builds a ListBookings route that injects the given principal.
func listRouter(h *BookingHandler, p shared.Principal) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/bookings", func(c *gin.Context) { c.Set(principalContextKey, p) }, h.ListBookings)
	return r
}

func slot(day int, start, end string) scheduling.WeeklySlot {
	return scheduling.WeeklySlot{DayOfWeek: day, Start: scheduling.TimeHHMM(start), End: scheduling.TimeHHMM(end)}
}

func TestBookingHandler_CreateBooking_ExactMatch(t *testing.T) {
	mock := &mockBookingSvc{
		result: &scheduling.BookingResponse{
			Results: []scheduling.SlotResult{{
				Slot: slot(0, "09:00", "10:00"),
				ExactMatch: &scheduling.BookingAlternative{
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
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	payload := scheduling.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredSlots:  []scheduling.WeeklySlot{slot(0, "09:00", "10:00")},
		DurationMinutes: 60,
		RequiredGender:  "female",
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

	var response scheduling.BookingResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response.Results, 1)
	assert.NotNil(t, response.Results[0].ExactMatch)
	assert.Equal(t, "Alice", response.Results[0].ExactMatch.TeacherName)
	assert.Equal(t, 100, response.Results[0].ExactMatch.Score)
	assert.Contains(t, response.Results[0].Message, "Exact match found")
}

func TestBookingHandler_CreateBooking_Alternatives(t *testing.T) {
	mock := &mockBookingSvc{
		result: &scheduling.BookingResponse{
			Results: []scheduling.SlotResult{{
				Slot: slot(0, "09:00", "10:00"),
				Alternatives: []scheduling.BookingAlternative{{
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
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	payload := scheduling.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredSlots:  []scheduling.WeeklySlot{slot(0, "09:00", "10:00")},
		DurationMinutes: 60,
		RequiredGender:  "female",
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

	var response scheduling.BookingResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response.Results, 1)
	assert.Nil(t, response.Results[0].ExactMatch)
	assert.Len(t, response.Results[0].Alternatives, 1)
}

func TestBookingHandler_CreateBooking_EmptyAlternatives(t *testing.T) {
	mock := &mockBookingSvc{
		result: &scheduling.BookingResponse{
			Results: []scheduling.SlotResult{{
				Slot:    slot(0, "09:00", "10:00"),
				Message: "No exact match found. No alternatives available.",
			}},
		},
	}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	payload := scheduling.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredSlots:  []scheduling.WeeklySlot{slot(0, "09:00", "10:00")},
		DurationMinutes: 60,
		RequiredGender:  "female",
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

func TestBookingHandler_CreateBooking_ValidationError(t *testing.T) {
	mock := &mockBookingSvc{
		evalErr: &scheduling.ValidationError{Msg: "preferred_slots must not be empty"},
	}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	payload := scheduling.BookingRequest{
		SubjectID:      0,
		BranchID:       2,
		RequiredGender: "female",
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
}

func TestBookingHandler_CreateBooking_InvalidJSON(t *testing.T) {
	mock := &mockBookingSvc{}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings", handler.CreateBooking)

	req := httptest.NewRequest(http.MethodPost, "/api/bookings", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBookingHandler_CreateBooking_InvalidRequiredGender(t *testing.T) {
	mock := &mockBookingSvc{}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	payload := scheduling.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredSlots:  []scheduling.WeeklySlot{slot(0, "09:00", "10:00")},
		DurationMinutes: 60,
		RequiredGender:  "Female",
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
}

func TestBookingHandler_CreateBooking_ServiceError(t *testing.T) {
	mock := &mockBookingSvc{evalErr: assert.AnError}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	payload := scheduling.BookingRequest{
		SubjectID:       3,
		BranchID:        2,
		PreferredSlots:  []scheduling.WeeklySlot{slot(0, "09:00", "10:00")},
		DurationMinutes: 60,
		RequiredGender:  "female",
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

func TestBookingHandler_CreateBooking_OptionalPreferredTeacher(t *testing.T) {
	mock := &mockBookingSvc{
		result: &scheduling.BookingResponse{
			Results: []scheduling.SlotResult{{
				Slot: slot(0, "09:00", "10:00"),
				Alternatives: []scheduling.BookingAlternative{{
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
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	preferredTeacher := 5
	payload := scheduling.BookingRequest{
		SubjectID:          3,
		BranchID:           2,
		PreferredSlots:     []scheduling.WeeklySlot{slot(0, "09:00", "10:00")},
		DurationMinutes:    60,
		PreferredTeacherID: shared.Some(preferredTeacher),
		RequiredGender:     "female",
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
	var response scheduling.BookingResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response.Results, 1)
}

func TestBookingHandler_ConfirmBooking_Success(t *testing.T) {
	mock := &mockBookingSvc{
		confirm: &scheduling.Booking{
			ID:          10,
			TeacherID:   1,
			BranchID:    2,
			SubjectID:   3,
			StartTime:   time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
			EndTime:     time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
			StudentID:   3,
			StudentName: "John Doe",
		},
	}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	payload := scheduling.ConfirmBookingRequest{
		TeacherID: 1,
		BranchID:  2,
		SubjectID: 3,
		StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StudentID: 3,
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

	var response scheduling.Booking
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 10, response.ID)
	assert.Equal(t, "John Doe", response.StudentName)
}

func TestBookingHandler_ConfirmBooking_ValidationError(t *testing.T) {
	mock := &mockBookingSvc{
		confErr: &scheduling.ValidationError{Msg: "student_id must be positive"},
	}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	payload := scheduling.ConfirmBookingRequest{
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
}

func TestBookingHandler_ConfirmBooking_ConflictError(t *testing.T) {
	mock := &mockBookingSvc{
		confErr: &scheduling.ConflictError{Msg: "overlapping booking with teacher requested time slot"},
	}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	payload := scheduling.ConfirmBookingRequest{
		TeacherID: 1,
		BranchID:  2,
		SubjectID: 3,
		StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StudentID: 3,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/bookings/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/bookings/confirm", handler.ConfirmBooking)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "overlapping booking")
}

func TestBookingHandler_ConfirmBooking_ServiceError(t *testing.T) {
	mock := &mockBookingSvc{confErr: assert.AnError}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	payload := scheduling.ConfirmBookingRequest{
		TeacherID: 1,
		BranchID:  2,
		SubjectID: 3,
		StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StudentID: 3,
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

func TestBookingHandler_CancelBooking_Success(t *testing.T) {
	mock := &mockBookingSvc{}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.DELETE("/api/bookings/:id", handler.CancelBooking)

	req := httptest.NewRequest(http.MethodDelete, "/api/bookings/10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Booking cancelled successfully")
}

func TestBookingHandler_CancelBooking_NotFound(t *testing.T) {
	mock := &mockBookingSvc{
		cancelErr: &scheduling.NotFoundError{Msg: "booking 999 not found"},
	}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.DELETE("/api/bookings/:id", handler.CancelBooking)

	req := httptest.NewRequest(http.MethodDelete, "/api/bookings/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}

func TestBookingHandler_CancelBooking_InvalidID(t *testing.T) {
	mock := &mockBookingSvc{}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.DELETE("/api/bookings/:id", handler.CancelBooking)

	req := httptest.NewRequest(http.MethodDelete, "/api/bookings/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid booking id")
}

func TestBookingHandler_CancelBooking_ServiceError(t *testing.T) {
	mock := &mockBookingSvc{cancelErr: assert.AnError}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.DELETE("/api/bookings/:id", handler.CancelBooking)

	req := httptest.NewRequest(http.MethodDelete, "/api/bookings/10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to cancel booking")
}

func TestBookingHandler_ListBookings_AdminSeesAll(t *testing.T) {
	mock := &mockBookingSvc{
		listAll: []scheduling.Booking{
			{ID: 1, TeacherID: 1, StudentID: 3, StudentName: "John Doe", StartTime: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)},
			{ID: 2, TeacherID: 2, StudentID: 4, StudentName: "Jane Doe", StartTime: time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), EndTime: time.Date(2026, 6, 2, 11, 0, 0, 0, time.UTC)},
		},
	}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())
	r := listRouter(handler, shared.Principal{UserID: 1, Role: shared.RoleAdmin})

	req := httptest.NewRequest(http.MethodGet, "/api/bookings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var bookings []scheduling.Booking
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &bookings))
	assert.Len(t, bookings, 2)
	assert.Equal(t, "John Doe", bookings[0].StudentName)
	assert.Nil(t, mock.gotStudentID) // admin path does not scope
}

func TestBookingHandler_ListBookings_StudentSeesOwn(t *testing.T) {
	mock := &mockBookingSvc{listAll: []scheduling.Booking{{ID: 1, StudentID: 7}}}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())
	r := listRouter(handler, shared.Principal{UserID: 7, Role: shared.RoleStudent})

	req := httptest.NewRequest(http.MethodGet, "/api/bookings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []int{7}, mock.gotStudentID) // scoped to self
}

func TestBookingHandler_ListBookings_ParentSeesChildren(t *testing.T) {
	mock := &mockBookingSvc{listAll: []scheduling.Booking{}}
	guardian := &mockGuardian{ids: []int{3, 4}}
	handler := NewBookingHandler(mock, guardian, slog.Default())
	r := listRouter(handler, shared.Principal{UserID: 9, Role: shared.RoleParent})

	req := httptest.NewRequest(http.MethodGet, "/api/bookings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []int{3, 4}, mock.gotStudentID) // scoped to the parent's students
}

func TestBookingHandler_ListBookings_ServiceError(t *testing.T) {
	mock := &mockBookingSvc{listErr: assert.AnError}
	handler := NewBookingHandler(mock, &mockGuardian{}, slog.Default())
	r := listRouter(handler, shared.Principal{UserID: 1, Role: shared.RoleAdmin})

	req := httptest.NewRequest(http.MethodGet, "/api/bookings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to list bookings")
}
