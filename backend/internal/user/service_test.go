package user

import (
	"context"
	"log/slog"
	"testing"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

type mockStore struct {
	mock.Mock
}

func (m *mockStore) CreateUser(ctx context.Context, name, passwordHash string) (*User, error) {
	args := m.Called(ctx, name, passwordHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *mockStore) GetCredentialByName(ctx context.Context, name string) (id int, passwordHash string, err error) {
	args := m.Called(ctx, name)
	return args.Int(0), args.String(1), args.Error(2)
}

func TestUserService_Register_Success_HashesPassword(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	store.On("CreateUser", mock.Anything, "alice", mock.MatchedBy(func(hash string) bool {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte("s3cret")) == nil
	})).Return(&User{ID: 1, Name: "alice"}, nil)

	u, err := svc.Register(context.Background(), "alice", "s3cret")
	assert.NoError(t, err)
	assert.Equal(t, &User{ID: 1, Name: "alice"}, u)
	store.AssertExpectations(t)
}

func TestUserService_Register_DuplicateName_Conflict(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	store.On("CreateUser", mock.Anything, "alice", mock.Anything).
		Return(nil, &shared.ConflictError{Msg: `username "alice" already taken`})

	_, err := svc.Register(context.Background(), "alice", "s3cret")

	var confErr *shared.ConflictError
	assert.ErrorAs(t, err, &confErr)
}

func TestUserService_Register_EmptyPassword_Rejected(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	_, err := svc.Register(context.Background(), "alice", "")

	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	store.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Login_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	hash, err := bcrypt.GenerateFromPassword([]byte("s3cret"), bcrypt.DefaultCost)
	assert.NoError(t, err)
	store.On("GetCredentialByName", mock.Anything, "alice").Return(1, string(hash), nil)

	u, err := svc.Login(context.Background(), "alice", "s3cret")
	assert.NoError(t, err)
	assert.Equal(t, &User{ID: 1, Name: "alice"}, u)
}

func TestUserService_Login_WrongPassword_InvalidCredentials(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	hash, err := bcrypt.GenerateFromPassword([]byte("s3cret"), bcrypt.DefaultCost)
	assert.NoError(t, err)
	store.On("GetCredentialByName", mock.Anything, "alice").Return(1, string(hash), nil)

	_, err = svc.Login(context.Background(), "alice", "wrong")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestUserService_Login_UnknownUser_InvalidCredentials(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	store.On("GetCredentialByName", mock.Anything, "ghost").
		Return(0, "", &shared.NotFoundError{Msg: `user "ghost" not found`})

	_, err := svc.Login(context.Background(), "ghost", "s3cret")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestUserService_Login_EmptyName_NotQueried(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	_, err := svc.Login(context.Background(), "", "s3cret")

	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	store.AssertNotCalled(t, "GetCredentialByName", mock.Anything, mock.Anything)
}
