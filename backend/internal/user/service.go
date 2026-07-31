package user

import (
	"context"
	"errors"
	"log/slog"

	"github.com/RinZ5/converge/backend/internal/shared"
	"golang.org/x/crypto/bcrypt"
)

type ValidationError = shared.ValidationError
type ConflictError = shared.ConflictError

// ErrInvalidCredentials is returned for any failed login. The message is
// intentionally generic so it never reveals whether the username exists or the
// password was wrong.
var ErrInvalidCredentials = &shared.ValidationError{Msg: "invalid username or password"}

type Service struct {
	store  UserStore
	logger *slog.Logger
}

func NewService(store UserStore, logger *slog.Logger) *Service {
	return &Service{store: store, logger: logger}
}

func (s *Service) Register(ctx context.Context, name, password string) (*User, error) {
	if err := shared.ValidateAll(RegisterRequest{Name: name, Password: password},
		shared.NonEmpty("name", func(r RegisterRequest) string { return r.Name }),
		shared.NonEmpty("password", func(r RegisterRequest) string { return r.Password }),
	); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.store.CreateUser(ctx, name, string(hash))
}

func (s *Service) Login(ctx context.Context, name, password string) (*User, error) {
	if err := shared.ValidateAll(LoginRequest{Name: name, Password: password},
		shared.NonEmpty("name", func(r LoginRequest) string { return r.Name }),
		shared.NonEmpty("password", func(r LoginRequest) string { return r.Password }),
	); err != nil {
		return nil, err
	}

	id, hash, err := s.store.GetCredentialByName(ctx, name)
	if err != nil {
		var notFound *shared.NotFoundError
		if errors.As(err, &notFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &User{ID: id, Name: name}, nil
}
