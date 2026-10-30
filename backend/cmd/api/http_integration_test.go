//go:build integration

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	backupDir := t.TempDir()
	backupService := backup.New(backupDir, 24*time.Hour, 24*time.Hour, logger)
	backupService.SetAfterRestore(func() error { return db.RecoverAfterLiveRestore(database) })
	server := httptest.NewServer(newRouter(database, jwt, logger, backupService))
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
	request("DELETE", fmt.Sprintf("%s/%d", parentPath, alice.ID), nil, admin.Token, 400)
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
	businessSnapshot := func() string {
		t.Helper()
		var value string
		require.NoError(t, database.QueryRow(`SELECT jsonb_build_object(
   'teacher', (SELECT to_jsonb(x)-'status' FROM teachers x WHERE id=$1),
   'availability', (SELECT jsonb_agg(to_jsonb(x) ORDER BY id) FROM teacher_availability x WHERE teacher_id=$1),
   'subjects', (SELECT jsonb_agg(to_jsonb(x) ORDER BY subject_id) FROM teacher_subjects x WHERE teacher_id=$1),
   'branch', (SELECT to_jsonb(x)-'status' FROM branches x WHERE id=$2),
   'bookings', (SELECT jsonb_agg(to_jsonb(x) ORDER BY id) FROM bookings x))::text`, tutor.ID, branch.ID).Scan(&value))
		return value
	}
	beforeStatusChange := businessSnapshot()
	for _, state := range []string{"deactivated", "active"} {
		request("PATCH", teacherPath+"/status", map[string]any{"status": state}, admin.Token, 200)
		var storedStatus string
		require.NoError(t, database.QueryRow("SELECT status FROM teachers WHERE id=$1", tutor.ID).Scan(&storedStatus))
		require.Equal(t, state, storedStatus)
		require.JSONEq(t, beforeStatusChange, businessSnapshot())
	}
	for _, state := range []string{"deactivated", "active"} {
		request("PATCH", branchPath+"/status", map[string]any{"status": state}, admin.Token, 200)
		var storedStatus string
		require.NoError(t, database.QueryRow("SELECT status FROM branches WHERE id=$1", branch.ID).Scan(&storedStatus))
		require.Equal(t, state, storedStatus)
		require.JSONEq(t, beforeStatusChange, businessSnapshot())
	}

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

	request("POST", parentPath, map[string]any{"student_id": bob.ID}, admin.Token, 200)
	type unlinkResult struct {
		code int
		err  error
	}
	results := make(chan unlinkResult, 2)
	startUnlinks := make(chan struct{})
	for _, id := range []int{alice.ID, bob.ID} {
		go func(id int) {
			<-startUnlinks
			req, err := http.NewRequest(http.MethodDelete, server.URL+fmt.Sprintf("%s/%d", parentPath, id), nil)
			if err != nil {
				results <- unlinkResult{err: err}
				return
			}
			req.Header.Set("Authorization", "Bearer "+admin.Token)
			response, err := client.Do(req)
			if err != nil {
				results <- unlinkResult{err: err}
				return
			}
			_, readErr := io.Copy(io.Discard, response.Body)
			closeErr := response.Body.Close()
			if readErr != nil {
				err = readErr
			} else {
				err = closeErr
			}
			results <- unlinkResult{code: response.StatusCode, err: err}
		}(id)
	}
	close(startUnlinks)
	var codes []int
	for range 2 {
		result := <-results
		require.NoError(t, result.err)
		codes = append(codes, result.code)
	}
	require.ElementsMatch(t, []int{200, 400}, codes)
	require.NoError(t, json.Unmarshal(request("GET", parentPath, nil, admin.Token, 200), &linked))
	require.Len(t, linked, 1)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	for key, value := range map[string]string{
		"DB_HOST": host, "DB_PORT": port.Port(), "POSTGRES_USER": "test", "POSTGRES_PASSWORD": "test", "POSTGRES_DB": "converge_http_test",
		"PGHOST": host, "PGPORT": port.Port(), "PGUSER": "test", "PGPASSWORD": "test", "PGDATABASE": "converge_http_test",
	} {
		t.Setenv(key, value)
	}
	_, err = database.Exec("UPDATE branches SET capacity=10 WHERE id=$1", branch.ID)
	require.NoError(t, err)
	backupContext, stopBackups := context.WithCancel(ctx)
	defer stopBackups()
	backupService.Start(backupContext)
	backupStatus := func() backup.Status {
		t.Helper()
		var status backup.Status
		require.NoError(t, json.Unmarshal(request("GET", "/api/recovery/backups", nil, admin.Token, 200), &status))
		return status
	}
	require.Eventually(t, func() bool { status := backupStatus(); return status.LastSuccess != nil && !status.Running }, time.Minute, 50*time.Millisecond)
	automatic := backupStatus()
	require.NotEmpty(t, automatic.Latest)
	require.Eventually(t, func() bool { return time.Now().After(automatic.LastSuccess.Add(time.Second)) }, 3*time.Second, 10*time.Millisecond)
	_, err = database.Exec("UPDATE branches SET capacity=20 WHERE id=$1", branch.ID)
	require.NoError(t, err)
	request("POST", "/api/recovery/backups", map[string]any{}, admin.Token, 202)
	require.Eventually(t, func() bool {
		status := backupStatus()
		return !status.Running && !status.Queued && status.Latest != automatic.Latest
	}, time.Minute, 50*time.Millisecond)
	manual := backupStatus()
	require.True(t, manual.LastSuccess.After(*automatic.LastSuccess))
	_, err = database.Exec("UPDATE branches SET capacity=30 WHERE id=$1", branch.ID)
	require.NoError(t, err)
	_, err = database.Exec("INSERT INTO branches (name, capacity) VALUES ('after-logical-backup', 30)")
	require.NoError(t, err)
	const auditSnapshot = `SELECT COALESCE(json_agg(row_to_json(events) ORDER BY id), '[]'::json)::text FROM admin_audit_event events`
	const auditSnapshotUpTo = `SELECT COALESCE(json_agg(row_to_json(events) ORDER BY id), '[]'::json)::text FROM (SELECT * FROM admin_audit_event WHERE id <= $1) events`
	var historyBefore string
	var lastEventID int64
	require.NoError(t, database.QueryRow(auditSnapshot).Scan(&historyBefore))
	require.NoError(t, database.QueryRow("SELECT max(id) FROM admin_audit_event").Scan(&lastEventID))
	restoreBody := map[string]string{"backup_name": manual.Latest, "confirmation": "wrong"}
	request("POST", "/api/recovery/restores", restoreBody, admin.Token, 400)
	require.NoError(t, database.QueryRow("SELECT capacity FROM branches WHERE id=$1", branch.ID).Scan(&capacity))
	require.Equal(t, 30, capacity)
	restoreBody["confirmation"] = "RESTORE LIVE DATA from " + manual.Latest
	func() {
		require.True(t, backupService.EnterRequest())
		defer backupService.LeaveRequest()
		request("POST", "/api/recovery/restores", restoreBody, admin.Token, 202)
		request("POST", "/api/recovery/restores", restoreBody, admin.Token, 409)
		request("GET", "/api/branches", nil, admin.Token, 503)
		request("POST", "/api/recovery/backups", map[string]any{}, admin.Token, 503)
		status := backupStatus()
		require.True(t, status.Restoring)
		require.True(t, status.LastSuccess.Equal(*manual.LastSuccess))
	}()
	require.Eventually(t, func() bool { return !backupStatus().Restoring }, time.Minute, 50*time.Millisecond)
	restored := backupStatus()
	require.Empty(t, restored.RestoreError)
	require.NotNil(t, restored.RestoredAt)
	require.NotEqual(t, manual.Latest, restored.Latest)
	safetyPath := filepath.Join(backupDir, restored.Latest)
	safetyData, err := os.ReadFile(safetyPath)
	require.NoError(t, err)
	checksum, err := os.ReadFile(safetyPath + ".sha256")
	require.NoError(t, err)
	fields := strings.Fields(string(checksum))
	require.Len(t, fields, 2)
	digest := sha256.Sum256(safetyData)
	require.Equal(t, hex.EncodeToString(digest[:]), fields[0])
	listing, err := exec.CommandContext(ctx, "pg_restore", "--list", safetyPath).CombinedOutput()
	require.NoError(t, err, string(listing))
	require.NoError(t, database.QueryRow("SELECT capacity FROM branches WHERE id=$1", branch.ID).Scan(&capacity))
	require.Equal(t, 20, capacity)
	var laterRows int
	require.NoError(t, database.QueryRow("SELECT count(*) FROM branches WHERE name='after-logical-backup'").Scan(&laterRows))
	require.Zero(t, laterRows)
	var historyAfter string
	require.NoError(t, database.QueryRow(auditSnapshotUpTo, lastEventID).Scan(&historyAfter))
	require.JSONEq(t, historyBefore, historyAfter, "restore must preserve every recorded change, including changes after the selected backup")
	var timeline struct {
		Events []struct {
			ID        int64  `json:"id"`
			Resource  string `json:"resource"`
			Operation string `json:"operation"`
		} `json:"events"`
	}
	require.NoError(t, json.Unmarshal(request("GET", "/api/recovery/timeline", nil, admin.Token, 200), &timeline))
	require.GreaterOrEqual(t, len(timeline.Events), 2)
	require.Equal(t, "database", timeline.Events[0].Resource, "restoring live data must be recorded as a change")
	require.Equal(t, "restore", timeline.Events[0].Operation)
	require.Equal(t, lastEventID, timeline.Events[1].ID)
	_, err = database.Exec("UPDATE branches SET capacity=25 WHERE id=$1", branch.ID)
	require.NoError(t, err)
	var nextEventID int64
	require.NoError(t, database.QueryRow("SELECT max(id) FROM admin_audit_event").Scan(&nextEventID))
	require.Greater(t, nextEventID, lastEventID)

	options := backup.RestoreOptions{
		BackupDir: backupDir, BackupName: manual.Latest,
		SourceHost: host, SourcePort: port.Port(), TargetHost: host, TargetPort: port.Port(),
		TargetDB: "converge_http_test", TargetUser: "test", TargetPassword: "test",
		Confirmation: "RESTORE LIVE DATA from " + manual.Latest,
	}
	require.NoError(t, database.QueryRow(auditSnapshot).Scan(&historyBefore))
	_, err = database.Exec("CREATE VIEW restore_failure_guard AS SELECT id FROM branches")
	require.NoError(t, err)
	require.ErrorContains(t, backup.RestoreLive(ctx, options), "restore archive")
	require.NoError(t, database.QueryRow("SELECT capacity FROM branches WHERE id=$1", branch.ID).Scan(&capacity))
	require.Equal(t, 25, capacity)
	require.NoError(t, database.QueryRow(auditSnapshot).Scan(&historyAfter))
	require.JSONEq(t, historyBefore, historyAfter)
	_, err = database.Exec("DROP VIEW restore_failure_guard")
	require.NoError(t, err)

	_, err = database.Exec("DROP TRIGGER admin_data_change ON branches")
	require.NoError(t, err)
	legacyName := "legacy-without-audit.dump"
	legacyPath := filepath.Join(backupDir, legacyName)
	output, err := exec.CommandContext(ctx, "pg_dump", "--format=custom", "--exclude-table=public.admin_audit_event", "--file="+legacyPath).CombinedOutput()
	require.NoError(t, err, string(output))
	legacyData, err := os.ReadFile(legacyPath)
	require.NoError(t, err)
	legacyDigest := sha256.Sum256(legacyData)
	require.NoError(t, os.WriteFile(legacyPath+".sha256", []byte(hex.EncodeToString(legacyDigest[:])+"  "+legacyName+"\n"), 0600))
	require.NoError(t, db.RecoverAfterLiveRestore(database))
	_, err = database.Exec("UPDATE branches SET capacity=35 WHERE id=$1", branch.ID)
	require.NoError(t, err)
	require.NoError(t, database.QueryRow(auditSnapshot).Scan(&historyBefore))
	require.NoError(t, database.QueryRow("SELECT max(id) FROM admin_audit_event").Scan(&lastEventID))
	options.BackupName = legacyName
	options.Confirmation = "RESTORE LIVE DATA from " + legacyName
	require.NoError(t, backup.RestoreLive(ctx, options))
	require.NoError(t, db.RecoverAfterLiveRestore(database))
	require.NoError(t, database.QueryRow("SELECT capacity FROM branches WHERE id=$1", branch.ID).Scan(&capacity))
	require.Equal(t, 25, capacity)
	require.NoError(t, database.QueryRow(auditSnapshotUpTo, lastEventID).Scan(&historyAfter))
	require.JSONEq(t, historyBefore, historyAfter)
	require.NoError(t, database.QueryRow("SELECT max(id) FROM admin_audit_event").Scan(&lastEventID))
	_, err = database.Exec("UPDATE branches SET capacity=45 WHERE id=$1", branch.ID)
	require.NoError(t, err)
	require.NoError(t, database.QueryRow("SELECT max(id) FROM admin_audit_event").Scan(&nextEventID))
	require.Greater(t, nextEventID, lastEventID)
	selections, err := filepath.Glob(filepath.Join(backupDir, ".restore-history-*.list"))
	require.NoError(t, err)
	require.Empty(t, selections)

}
