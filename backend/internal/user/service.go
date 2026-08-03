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

const maxPasswordBytes = 72

// ErrInvalidCredentials is returned for any failed login. The message is
// intentionally generic so it never reveals whether the username exists or the
// password was wrong.
var ErrInvalidCredentials = &shared.ValidationError{Msg: "invalid username or password"}

var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

type Service struct {
	store  UserStore
	issuer TokenIssuer
	logger *slog.Logger
}

func NewService(store UserStore, issuer TokenIssuer, logger *slog.Logger) *Service {
	return &Service{store: store, issuer: issuer, logger: logger}
}

func (s *Service) Register(ctx context.Context, name, password, role string, studentIDs []int) (*User, error) {
	if err := shared.ValidateAll(RegisterRequest{Name: name, Password: password},
		shared.NonEmpty("name", func(r RegisterRequest) string { return r.Name }),
		shared.NonEmpty("password", func(r RegisterRequest) string { return r.Password }),
		shared.MaxBytes("password", maxPasswordBytes, func(r RegisterRequest) string { return r.Password }),
	); err != nil {
		return nil, err
	}

	validRole, err := shared.ParseRole(role)
	if err != nil {
		return nil, err
	}

	if validRole == shared.RoleParent && len(studentIDs) == 0 {
		return nil, &ValidationError{Msg: "a parent must be linked to at least one student"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	if validRole == shared.RoleParent {
		return s.store.CreateParent(ctx, name, string(hash), studentIDs)
	}
	return s.store.CreateUser(ctx, name, string(hash), string(validRole))
}

func (s *Service) StudentIDsForParent(ctx context.Context, parentID int) ([]int, error) {
	return s.store.StudentIDsForParent(ctx, parentID)
}

func (s *Service) ListStudents(ctx context.Context) ([]User, error) {
	return s.store.ListStudents(ctx)
}

func (s *Service) ListParents(ctx context.Context) ([]ParentWithStudents, error) {
	return s.store.ListParentsWithStudents(ctx)
}

func (s *Service) StudentsForParent(ctx context.Context, parentID int) ([]User, error) {
	return s.store.StudentsForParent(ctx, parentID)
}

func (s *Service) AddStudentToParent(ctx context.Context, parentID, studentID int) error {
	return s.store.LinkParentStudent(ctx, parentID, studentID)
}

func (s *Service) RemoveStudentFromParent(ctx context.Context, parentID, studentID int) error {
	return s.store.UnlinkParentStudent(ctx, parentID, studentID)
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
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
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
