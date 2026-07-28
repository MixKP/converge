package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/RinZ5/converge/backend/internal/branch"
	"github.com/RinZ5/converge/backend/internal/shared"
)

type BranchRepo struct {
	DB *sql.DB
}

func NewBranchRepository(database *sql.DB) *BranchRepo {
	return &BranchRepo{DB: database}
}

func (r *BranchRepo) GetBranches(ctx context.Context) ([]branch.Branch, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, name, capacity FROM branches ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var branches []branch.Branch
	for rows.Next() {
		var b branch.Branch
		if err := rows.Scan(&b.ID, &b.Name, &b.Capacity); err != nil {
			return nil, err
		}
		branches = append(branches, b)
	}
	return branches, rows.Err()
}

func (r *BranchRepo) GetBranchByID(ctx context.Context, branchID int) (*branch.Branch, error) {
	row := r.DB.QueryRowContext(ctx, `SELECT id, name, capacity FROM branches WHERE id = $1`, branchID)

	var b branch.Branch
	if err := row.Scan(&b.ID, &b.Name, &b.Capacity); err != nil {
		if err == sql.ErrNoRows {
			return nil, &shared.NotFoundError{Msg: fmt.Sprintf("branch %d not found", branchID)}
		}
		return nil, err
	}
	return &b, nil
}
