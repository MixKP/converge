package user

import (
	"context"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type UserStore interface {
	CreateUser(ctx context.Context, name, passwordHash, role string) (*User, error)
	GetCredentialByName(ctx context.Context, name string) (id int, passwordHash, role string, err error)
}

type TokenIssuer interface {
	Issue(p shared.Principal) (string, error)
}
