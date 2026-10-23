//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/adapter/db"
	"github.com/RinZ5/converge/backend/internal/adapter/token"
	"github.com/RinZ5/converge/backend/internal/backup"
	"github.com/RinZ5/converge/backend/internal/scheduling"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/RinZ5/converge/backend/internal/teacher"
	"github.com/RinZ5/converge/backend/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"golang.org/x/crypto/bcrypt"
)

func TestApplicationHTTPWithPostgres(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine", postgres.WithDatabase("converge_http_test"), postgres.WithUsername("test"), postgres.WithPassword("test"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	connection, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	database, err := db.InitDBWithConnString(connection)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	require.NoError(t, db.AutoMigrate(database))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	jwt := token.NewJWT([]byte("http-integration-secret"), time.Hour)
	users := user.NewService(db.NewUserRepository(database), jwt, logger)
	_, err = users.Register(ctx, "operator", "secret123", "admin", nil)
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	server := httptest.NewServer(newRouter(database, jwt, logger, backup.New("", time.Hour, 24*time.Hour, logger)))
	t.Cleanup(server.Close)
	client := server.Client()
	client.Timeout = 10 * time.Second
	request := func(method, path string, body any, bearer string, want int) []byte {
		t.Helper()
		var reader io.Reader
		if body != nil {
			data, e := json.Marshal(body)
			require.NoError(t, e)
			reader = bytes.NewReader(data)
		}
		req, e := http.NewRequest(method, server.URL+path, reader)
		require.NoError(t, e)
		req.Header.Set("Content-Type", "application/json")
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		response, e := client.Do(req)
		require.NoError(t, e)
		defer response.Body.Close()
		data, e := io.ReadAll(response.Body)
		require.NoError(t, e)
		require.Equal(t, want, response.StatusCode, "%s %s: %s", method, path, data)
		return data
	}
	login := func(name, password string) user.AuthResponse {
		t.Helper()
		var result user.AuthResponse
		require.NoError(t, json.Unmarshal(request("POST", "/api/login", map[string]any{"name": name, "password": password}, "", 200), &result))
		require.NotEmpty(t, result.Token)
		return result
	}
	admin := login("operator", "secret123")
	register := func(name, role string, ids []int) user.User {
		t.Helper()
		var result user.User
		require.NoError(t, json.Unmarshal(request("POST", "/api/register", map[string]any{"name": name, "password": "secret123", "role": role, "student_ids": ids}, admin.Token, 201), &result))
		require.NotZero(t, result.ID)
		require.Equal(t, name, result.Name)
		require.Equal(t, role, result.Role)
		return result
	}
	alice := register("alice", "student", nil)
	bob := register("bob", "student", nil)
	parent := register("guardian", "parent", []int{alice.ID})
	aliceAuth := login("alice", "secret123")
	bobAuth := login("bob", "secret123")
	parentAuth := login("guardian", "secret123")
	var hash string
	require.NoError(t, database.QueryRow("SELECT password_hash FROM users WHERE id=$1", alice.ID).Scan(&hash))
	require.NotEqual(t, "secret123", hash)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("secret123")))
	request("POST", "/api/register", map[string]any{"name": "alice", "password": "secret123", "role": "student"}, admin.Token, 409)
	request("POST", "/api/login", map[string]any{"name": "alice", "password": "wrong"}, "", 401)
	var students []user.User
	require.NoError(t, json.Unmarshal(request("GET", "/api/students", nil, admin.Token, 200), &students))
	require.Len(t, students, 2)
	require.ElementsMatch(t, []int{alice.ID, bob.ID}, []int{students[0].ID, students[1].ID})
	var parents []user.ParentWithStudents
	require.NoError(t, json.Unmarshal(request("GET", "/api/parents", nil, admin.Token, 200), &parents))
	require.Len(t, parents, 1)
	require.Len(t, parents[0].Students, 1)
	require.Equal(t, alice.ID, parents[0].Students[0].ID)
	parentPath := fmt.Sprintf("/api/parents/%d/students", parent.ID)
	request("POST", parentPath, map[string]any{"student_id": bob.ID}, admin.Token, 200)
	var linked []user.User
	require.NoError(t, json.Unmarshal(request("GET", parentPath, nil, admin.Token, 200), &linked))
	require.Len(t, linked, 2)
	require.ElementsMatch(t, []int{alice.ID, bob.ID}, []int{linked[0].ID, linked[1].ID})
	request("DELETE", fmt.Sprintf("%s/%d", parentPath, bob.ID), nil, admin.Token, 200)
	require.NoError(t, json.Unmarshal(request("GET", parentPath, nil, admin.Token, 200), &linked))
	require.Len(t, linked, 1)
	require.Equal(t, alice.ID, linked[0].ID)
	request("POST", parentPath, map[string]any{"student_id": 999999}, admin.Token, 404)

	var branch struct {
		ID       int `json:"id"`
		Capacity int `json:"capacity"`
	}
	require.NoError(t, json.Unmarshal(request("POST", "/api/branches", map[string]any{"name": "Campus", "capacity": 30}, admin.Token, 201), &branch))
	require.NotZero(t, branch.ID)
	require.Equal(t, 30, branch.Capacity)
	request("POST", "/api/branches", map[string]any{"name": "Campus", "capacity": 30}, admin.Token, 409)
	branchPath := fmt.Sprintf("/api/branches/%d", branch.ID)
	request("PATCH", branchPath+"/capacity", map[string]any{"capacity": 20}, admin.Token, 200)
	request("PATCH", branchPath+"/capacity", map[string]any{"capacity": 0}, admin.Token, 400)
	var capacity int
	require.NoError(t, database.QueryRow("SELECT capacity FROM branches WHERE id=$1", branch.ID).Scan(&capacity))
	require.Equal(t, 20, capacity)
	request("PATCH", branchPath+"/status", map[string]any{"status": "deactivated"}, admin.Token, 200)
	var branchStatus string
	require.NoError(t, database.QueryRow("SELECT status FROM branches WHERE id=$1", branch.ID).Scan(&branchStatus))
	require.Equal(t, "deactivated", branchStatus)
	request("PATCH", branchPath+"/status", map[string]any{"status": "active"}, admin.Token, 200)
	require.NoError(t, database.QueryRow("SELECT status FROM branches WHERE id=$1", branch.ID).Scan(&branchStatus))
	require.Equal(t, "active", branchStatus)
	var branches []struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Capacity int    `json:"capacity"`
	}
	require.NoError(t, json.Unmarshal(request("GET", "/api/branches", nil, admin.Token, 200), &branches))
	require.Len(t, branches, 1)
	require.Equal(t, "Campus", branches[0].Name)
	require.Equal(t, 20, branches[0].Capacity)
	request("PATCH", "/api/commute", map[string]any{"commute_time": 25}, admin.Token, 200)
	require.JSONEq(t, `{"commute_time":25}`, string(request("GET", "/api/commute", nil, admin.Token, 200)))
	require.Contains(t, string(request("GET", "/api/commute?source_branch=1&destination_branch=1", nil, admin.Token, 200)), `"commute_time":0`)
	require.Contains(t, string(request("GET", "/api/commute?source_branch=1&destination_branch=2", nil, admin.Token, 200)), `"commute_time":25`)
	request("PATCH", "/api/commute", map[string]any{"commute_time": -1}, admin.Token, 400)
	require.JSONEq(t, `{"commute_time":25}`, string(request("GET", "/api/commute", nil, admin.Token, 200)))

	var tutor teacher.Teacher
	require.NoError(t, json.Unmarshal(request("POST", "/api/teachers", map[string]any{"name": "Tutor", "email": "tutor@test.com", "gender": "female"}, admin.Token, 201), &tutor))
	require.NotZero(t, tutor.ID)
	teacherPath := fmt.Sprintf("/api/teachers/%d", tutor.ID)
	request("PATCH", teacherPath+"/gender", map[string]any{"gender": "male"}, admin.Token, 200)
	var gender string
	require.NoError(t, database.QueryRow("SELECT gender FROM teachers WHERE id=$1", tutor.ID).Scan(&gender))
	require.Equal(t, "male", gender)
	request("PATCH", teacherPath+"/gender", map[string]any{"gender": "female"}, admin.Token, 200)
	request("PATCH", teacherPath+"/status", map[string]any{"status": "deactivated"}, admin.Token, 200)
	require.Contains(t, string(request("GET", "/api/teachers?status=deactivated", nil, admin.Token, 200)), `"name":"Tutor"`)
	require.JSONEq(t, `[]`, string(request("GET", "/api/teachers?status=active", nil, "", 200)))
	request("PATCH", teacherPath+"/status", map[string]any{"status": "active"}, admin.Token, 200)
	var subjectID int
	require.NoError(t, database.QueryRow("INSERT INTO subjects (name) VALUES ('Piano') RETURNING id").Scan(&subjectID))
	request("PUT", teacherPath+"/subjects", map[string]any{"subject_ids": []int{subjectID}}, admin.Token, 200)
	var subjects []shared.Subject
	require.NoError(t, json.Unmarshal(request("GET", teacherPath+"/subjects", nil, admin.Token, 200), &subjects))
	require.Len(t, subjects, 1)
	require.Equal(t, subjectID, subjects[0].ID)
	require.Equal(t, "Piano", subjects[0].Name)
	require.Contains(t, string(request("GET", fmt.Sprintf("/api/teachers?subject_id=%d", subjectID), nil, "", 200)), `"name":"Tutor"`)
	require.Contains(t, string(request("GET", "/api/subjects", nil, "", 200)), `"name":"Piano"`)
	weekly := []shared.WeeklySlot{}
	for day := 0; day < 7; day++ {
		weekly = append(weekly, shared.WeeklySlot{DayOfWeek: day, Start: "09:00", End: "17:00"})
	}
	request("PUT", teacherPath+"/availability", map[string]any{"weekly": weekly}, admin.Token, 200)
	var availability []teacher.TeacherAvailability
	require.NoError(t, json.Unmarshal(request("GET", "/api/availability", nil, "", 200), &availability))
	require.Len(t, availability, 1)
	require.Len(t, availability[0].Weekly, 7)
	require.Equal(t, tutor.ID, availability[0].Teacher.ID)
	require.ElementsMatch(t, weekly, availability[0].Weekly)
	require.Contains(t, string(request("GET", "/api/teachers", nil, "", 200)), `"name":"Tutor"`)
	var evaluation scheduling.BookingResponse
	require.NoError(t, json.Unmarshal(request("POST", "/api/bookings", map[string]any{"subject_id": subjectID, "branch_id": branch.ID, "required_gender": "female", "preferred_teacher_id": tutor.ID, "duration_minutes": 60, "preferred_slots": []shared.WeeklySlot{{DayOfWeek: 1, Start: "10:00", End: "11:00"}}}, admin.Token, 200), &evaluation))
	require.Len(t, evaluation.Results, 1)
	require.NotNil(t, evaluation.Results[0].ExactMatch)
	require.Equal(t, tutor.ID, evaluation.Results[0].ExactMatch.TeacherID)
	now := time.Now().In(shared.LoadLocation()).AddDate(0, 0, 7)
	start := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, shared.LoadLocation())
	confirm := scheduling.ConfirmBookingRequest{TeacherID: tutor.ID, BranchID: branch.ID, SubjectID: subjectID, StudentID: alice.ID, StartTime: start, EndTime: start.Add(time.Hour)}
	var booked scheduling.Booking
	require.NoError(t, json.Unmarshal(request("POST", "/api/bookings/confirm", confirm, admin.Token, 201), &booked))
	require.NotZero(t, booked.ID)
	require.Equal(t, "alice", booked.StudentName)
	require.Equal(t, confirm.TeacherID, booked.TeacherID)
	require.Equal(t, confirm.BranchID, booked.BranchID)
	require.Equal(t, confirm.SubjectID, booked.SubjectID)
	second := confirm
	second.StudentID = bob.ID
	second.StartTime = start.Add(4 * time.Hour)
	second.EndTime = start.Add(5 * time.Hour)
	var bobBooking scheduling.Booking
	require.NoError(t, json.Unmarshal(request("POST", "/api/bookings/confirm", second, admin.Token, 201), &bobBooking))
	require.NotZero(t, bobBooking.ID)
	request("POST", "/api/bookings/confirm", confirm, admin.Token, 409)
	list := func(bearer string) []scheduling.Booking {
		t.Helper()
		var result []scheduling.Booking
		require.NoError(t, json.Unmarshal(request("GET", "/api/bookings", nil, bearer, 200), &result))
		return result
	}
	require.Len(t, list(admin.Token), 2)
	request("POST", parentPath, map[string]any{"student_id": bob.ID}, admin.Token, 200)
	require.Len(t, list(parentAuth.Token), 2)
	request("DELETE", fmt.Sprintf("%s/%d", parentPath, bob.ID), nil, admin.Token, 200)
	require.Len(t, list(aliceAuth.Token), 1)
	require.Len(t, list(parentAuth.Token), 1)
	require.Len(t, list(bobAuth.Token), 1)
	require.Equal(t, bob.ID, list(bobAuth.Token)[0].StudentID)
	require.Equal(t, alice.ID, list(parentAuth.Token)[0].StudentID)
	require.Equal(t, alice.ID, list(aliceAuth.Token)[0].StudentID)
	bookingPath := fmt.Sprintf("/api/bookings/%d", booked.ID)
	moved := scheduling.RescheduleBookingRequest{StartTime: start.Add(time.Hour), EndTime: start.Add(2 * time.Hour)}
	var rescheduled scheduling.Booking
	require.NoError(t, json.Unmarshal(request("PUT", bookingPath, moved, admin.Token, 200), &rescheduled))
	require.True(t, rescheduled.StartTime.Equal(moved.StartTime))
	require.Equal(t, booked.ID, rescheduled.ID)
	require.True(t, rescheduled.EndTime.Equal(moved.EndTime))
	var stored time.Time
	require.NoError(t, database.QueryRow("SELECT start_time FROM bookings WHERE id=$1", booked.ID).Scan(&stored))
	require.True(t, stored.Equal(moved.StartTime))
	require.JSONEq(t, `{"message":"Booking cancelled successfully"}`, string(request("DELETE", bookingPath, nil, admin.Token, 200)))
	require.Len(t, list(admin.Token), 1)
	request("DELETE", fmt.Sprintf("/api/bookings/%d", bobBooking.ID), nil, admin.Token, 200)
	require.JSONEq(t, `[]`, string(request("GET", "/api/bookings", nil, admin.Token, 200)))
	request("DELETE", bookingPath, nil, admin.Token, 404)
	request("POST", "/api/bookings/confirm", confirm, admin.Token, 201)
	request("POST", "/api/bookings/confirm", second, admin.Token, 201)
	require.JSONEq(t, `{"message":"Availability saved successfully","deleted_bookings":2}`, string(request("PUT", teacherPath+"/availability", map[string]any{"weekly": weekly}, admin.Token, 200)))
	require.Empty(t, list(admin.Token))
	for _, route := range []struct{ method, path string }{
		{"POST", "/api/register"}, {"GET", "/api/students"}, {"GET", "/api/parents"}, {"GET", parentPath}, {"POST", parentPath}, {"DELETE", fmt.Sprintf("%s/%d", parentPath, alice.ID)},
		{"POST", "/api/teachers"}, {"PATCH", teacherPath + "/status"}, {"PATCH", teacherPath + "/gender"}, {"GET", teacherPath + "/subjects"}, {"PUT", teacherPath + "/subjects"}, {"PUT", teacherPath + "/availability"},
		{"GET", "/api/branches"}, {"POST", "/api/branches"}, {"PATCH", branchPath + "/capacity"}, {"PATCH", branchPath + "/status"},
		{"GET", "/api/commute"}, {"PATCH", "/api/commute"}, {"POST", "/api/bookings"}, {"POST", "/api/bookings/confirm"}, {"PUT", bookingPath}, {"DELETE", bookingPath},
	} {
		request(route.method, route.path, nil, "", 401)
		request(route.method, route.path, nil, aliceAuth.Token, 403)
	}
	request("GET", "/api/bookings", nil, "", 401)
	require.JSONEq(t, `{"status":"healthy"}`, string(request("GET", "/health", nil, "", 200)))
}
