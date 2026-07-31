package user

import "context"

type UserStore interface {
	CreateUser(ctx context.Context, name, passwordHash string) (*User, error)
	GetCredentialByName(ctx context.Context, name string) (id int, passwordHash string, err error)
}
