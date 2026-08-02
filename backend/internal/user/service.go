package user

import (
	"context"
	"errors"
	"github.com/RinZ5/converge/backend/internal/shared"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
)

type ValidationError = shared.ValidationError
type ConflictError = shared.ConflictError

// ErrInvalidCredentials is returned for any failed login. The message is
// intentionally generic so it never reveals whether the username exists or the
// password was wrong.
var ErrInvalidCredentials = &shared.ValidationError{Msg: "invalid username or password"}

type Service struct {
	store  UserStore
	issuer TokenIssuer
	logger *slog.Logger
}

func NewService(store UserStore, issuer TokenIssuer, logger *slog.Logger) *Service {
	return &Service{store: store, issuer: issuer, logger: logger}
}

func (s *Service) Register(ctx context.Context, name, password, role string) (*User, error) {
	if err := shared.ValidateAll(RegisterRequest{Name: name, Password: password},
		shared.NonEmpty("name", func(r RegisterRequest) string { return r.Name }),
		shared.NonEmpty("password", func(r RegisterRequest) string { return r.Password }),
	); err != nil {
		return nil, err
	}

	validRole, err := shared.ParseRole(role)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.store.CreateUser(ctx, name, string(hash), string(validRole))
}

func (s *Service) Login(ctx context.Context, name, password string) (*User, string, error) {
	if err := shared.ValidateAll(LoginRequest{Name: name, Password: password},
		shared.NonEmpty("name", func(r LoginRequest) string { return r.Name }),
		shared.NonEmpty("password", func(r LoginRequest) string { return r.Password }),
	); err != nil {
		return nil, "", err
	}

	id, hash, role, err := s.store.GetCredentialByName(ctx, name)
	if err != nil {
		var notFound *shared.NotFoundError
		if errors.As(err, &notFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	validRole, err := shared.ParseRole(role)
	if err != nil {
		return nil, "", err
	}

	u := &User{ID: id, Name: name, Role: role}
	token, err := s.issuer.Issue(shared.Principal{UserID: u.ID, Name: u.Name, Role: validRole})
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}
