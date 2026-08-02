package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/RinZ5/converge/backend/internal/user"
)

type UserRepo struct {
	DB *sql.DB
}

func NewUserRepository(database *sql.DB) *UserRepo {
	return &UserRepo{DB: database}
}

func (r *UserRepo) CreateUser(ctx context.Context, name, passwordHash, role string) (*user.User, error) {
	row := r.DB.QueryRowContext(ctx, `
		INSERT INTO users (name, password_hash, role) VALUES ($1, $2, $3)
		RETURNING id, name, role`, name, passwordHash, role)

	var u user.User
	if err := row.Scan(&u.ID, &u.Name, &u.Role); err != nil {
		if confErr := uniqueViolationError(err, fmt.Sprintf("username %q already taken", name)); confErr != nil {
			return nil, confErr
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) GetCredentialByName(ctx context.Context, name string) (id int, passwordHash, role string, err error) {
	row := r.DB.QueryRowContext(ctx, `SELECT id, password_hash, role FROM users WHERE name = $1`, name)

	if err := row.Scan(&id, &passwordHash, &role); err != nil {
		if err == sql.ErrNoRows {
			return 0, "", "", &shared.NotFoundError{Msg: fmt.Sprintf("user %q not found", name)}
		}
		return 0, "", "", err
	}
	return id, passwordHash, role, nil
}
