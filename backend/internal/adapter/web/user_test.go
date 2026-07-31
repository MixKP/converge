package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/RinZ5/converge/backend/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserService struct {
	registerUser *user.User
	registerErr  error
	loginUser    *user.User
	loginErr     error

	registerCalled bool
	loginCalled    bool
	gotName        string
	gotPassword    string
}

func (m *mockUserService) Register(ctx context.Context, name, password string) (*user.User, error) {
	m.registerCalled = true
	m.gotName, m.gotPassword = name, password
	return m.registerUser, m.registerErr
}

func (m *mockUserService) Login(ctx context.Context, name, password string) (*user.User, error) {
	m.loginCalled = true
	m.gotName, m.gotPassword = name, password
	return m.loginUser, m.loginErr
}

func TestUserHandler_Register(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mock       *mockUserService
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "success",
			body:       `{"name":"alice","password":"s3cret"}`,
			mock:       &mockUserService{registerUser: &user.User{ID: 1, Name: "alice"}},
			wantStatus: http.StatusCreated,
			wantCalled: true,
		},
		{
			name:       "duplicate name conflict",
			body:       `{"name":"alice","password":"s3cret"}`,
			mock:       &mockUserService{registerErr: &shared.ConflictError{Msg: `username "alice" already taken`}},
			wantStatus: http.StatusConflict,
			wantCalled: true,
		},
		{
			name:       "service validation error",
			body:       `{"name":"alice","password":"s3cret"}`,
			mock:       &mockUserService{registerErr: &shared.ValidationError{Msg: "name must not be empty"}},
			wantStatus: http.StatusBadRequest,
			wantCalled: true,
		},
		{
			name:       "missing password fails binding",
			body:       `{"name":"alice"}`,
			mock:       &mockUserService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:       "malformed json",
			body:       `{"name":`,
			mock:       &mockUserService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:       "service error",
			body:       `{"name":"alice","password":"s3cret"}`,
			mock:       &mockUserService{registerErr: assert.AnError},
			wantStatus: http.StatusInternalServerError,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewUserHandler(tt.mock, slog.Default())

			gin.SetMode(gin.TestMode)
			r := gin.Default()
			r.POST("/api/register", handler.Register)

			req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantCalled, tt.mock.registerCalled)

			if tt.wantStatus == http.StatusCreated {
				var got user.User
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
				assert.Equal(t, user.User{ID: 1, Name: "alice"}, got)
				assert.NotContains(t, w.Body.String(), "password")
			}
		})
	}
}

func TestUserHandler_Login(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mock       *mockUserService
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "success",
			body:       `{"name":"alice","password":"s3cret"}`,
			mock:       &mockUserService{loginUser: &user.User{ID: 1, Name: "alice"}},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "invalid credentials",
			body:       `{"name":"alice","password":"wrong"}`,
			mock:       &mockUserService{loginErr: user.ErrInvalidCredentials},
			wantStatus: http.StatusUnauthorized,
			wantCalled: true,
		},
		{
			name:       "missing password fails binding",
			body:       `{"name":"alice"}`,
			mock:       &mockUserService{},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:       "service error",
			body:       `{"name":"alice","password":"s3cret"}`,
			mock:       &mockUserService{loginErr: assert.AnError},
			wantStatus: http.StatusInternalServerError,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewUserHandler(tt.mock, slog.Default())

			gin.SetMode(gin.TestMode)
			r := gin.Default()
			r.POST("/api/login", handler.Login)

			req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantCalled, tt.mock.loginCalled)

			if tt.wantStatus == http.StatusOK {
				var got user.User
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
				assert.Equal(t, user.User{ID: 1, Name: "alice"}, got)
				assert.NotContains(t, w.Body.String(), "password")
			}
		})
	}
}
