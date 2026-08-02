package user

import (
	"context"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"testing"
)

type mockStore struct {
	mock.Mock
}

// stubIssuer is a fixed token issuer for service tests.
type stubIssuer struct {
	token string
	err   error
}

func (s stubIssuer) Issue(shared.Principal) (string, error) { return s.token, s.err }

func (m *mockStore) CreateUser(ctx context.Context, name, passwordHash, role string) (*User, error) {
	args := m.Called(ctx, name, passwordHash, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *mockStore) GetCredentialByName(ctx context.Context, name string) (id int, passwordHash, role string, err error) {
	args := m.Called(ctx, name)
	return args.Int(0), args.String(1), args.String(2), args.Error(3)
}

func TestUserService_Register_Success_HashesPassword(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, stubIssuer{token: "tok-xyz"}, slog.Default())

	store.On("CreateUser", mock.Anything, "alice", mock.MatchedBy(func(hash string) bool {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte("s3cret")) == nil
	}), "teacher").Return(&User{ID: 1, Name: "alice", Role: "teacher"}, nil)

	u, err := svc.Register(context.Background(), "alice", "s3cret", "teacher")
	assert.NoError(t, err)
	assert.Equal(t, &User{ID: 1, Name: "alice", Role: "teacher"}, u)
	store.AssertExpectations(t)
}

func TestUserService_Register_InvalidRole_Rejected(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, stubIssuer{token: "tok-xyz"}, slog.Default())

	_, err := svc.Register(context.Background(), "alice", "s3cret", "superuser")

	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	store.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Register_DuplicateName_Conflict(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, stubIssuer{token: "tok-xyz"}, slog.Default())

	store.On("CreateUser", mock.Anything, "alice", mock.Anything, "student").
		Return(nil, &shared.ConflictError{Msg: `username "alice" already taken`})

	_, err := svc.Register(context.Background(), "alice", "s3cret", "student")

	var confErr *shared.ConflictError
	assert.ErrorAs(t, err, &confErr)
}

func TestUserService_Register_EmptyPassword_Rejected(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, stubIssuer{token: "tok-xyz"}, slog.Default())

	_, err := svc.Register(context.Background(), "alice", "", "student")

	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	store.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Login_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, stubIssuer{token: "tok-xyz"}, slog.Default())

	hash, err := bcrypt.GenerateFromPassword([]byte("s3cret"), bcrypt.DefaultCost)
	assert.NoError(t, err)
	store.On("GetCredentialByName", mock.Anything, "alice").Return(1, string(hash), "teacher", nil)

	u, tok, err := svc.Login(context.Background(), "alice", "s3cret")
	assert.NoError(t, err)
	assert.Equal(t, &User{ID: 1, Name: "alice", Role: "teacher"}, u)
	assert.Equal(t, "tok-xyz", tok)
}

func TestUserService_Login_WrongPassword_InvalidCredentials(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, stubIssuer{token: "tok-xyz"}, slog.Default())

	hash, err := bcrypt.GenerateFromPassword([]byte("s3cret"), bcrypt.DefaultCost)
	assert.NoError(t, err)
	store.On("GetCredentialByName", mock.Anything, "alice").Return(1, string(hash), "teacher", nil)

	_, _, err = svc.Login(context.Background(), "alice", "wrong")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestUserService_Login_UnknownUser_InvalidCredentials(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, stubIssuer{token: "tok-xyz"}, slog.Default())

	store.On("GetCredentialByName", mock.Anything, "ghost").
		Return(0, "", "", &shared.NotFoundError{Msg: `user "ghost" not found`})

	_, _, err := svc.Login(context.Background(), "ghost", "s3cret")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestUserService_Login_EmptyName_NotQueried(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, stubIssuer{token: "tok-xyz"}, slog.Default())

	_, _, err := svc.Login(context.Background(), "", "s3cret")

	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	store.AssertNotCalled(t, "GetCredentialByName", mock.Anything, mock.Anything)
}
