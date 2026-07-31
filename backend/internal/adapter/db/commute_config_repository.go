package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type CommuteConfigRepo struct {
	DB *sql.DB
}

func NewCommuteConfigRepository(database *sql.DB) *CommuteConfigRepo {
	return &CommuteConfigRepo{DB: database}
}

func (r *CommuteConfigRepo) GetCommuteMinutes(ctx context.Context) (int, error) {
	var minutes int
	err := r.DB.QueryRowContext(ctx, `SELECT commute_minutes FROM commute_config WHERE id = 1`).Scan(&minutes)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, &shared.NotFoundError{Msg: "commute config not found"}
		}
		return 0, fmt.Errorf("get commute minutes: %w", err)
	}
	return minutes, nil
}

func (r *CommuteConfigRepo) SetCommuteMinutes(ctx context.Context, minutes int) error {
	return execUpdateOne(ctx, r.DB,
		`UPDATE commute_config SET commute_minutes = $1 WHERE id = 1`, []any{minutes},
		"commute_time must not be negative", "commute config not found", "set commute minutes")
}
