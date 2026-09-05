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
	rows, err := r.DB.QueryContext(ctx, `SELECT id, name, capacity, status FROM branches ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var branches []branch.Branch
	for rows.Next() {
		var b branch.Branch
		if err := rows.Scan(&b.ID, &b.Name, &b.Capacity, &b.Status); err != nil {
			return nil, err
		}
		branches = append(branches, b)
	}
	return branches, rows.Err()
}

func (r *BranchRepo) GetBranchByID(ctx context.Context, branchID int) (*branch.Branch, error) {
	row := r.DB.QueryRowContext(ctx, `SELECT id, name, capacity, status FROM branches WHERE id = $1`, branchID)

	var b branch.Branch
	if err := row.Scan(&b.ID, &b.Name, &b.Capacity, &b.Status); err != nil {
		if err == sql.ErrNoRows {
			return nil, &shared.NotFoundError{Msg: fmt.Sprintf("branch %d not found", branchID)}
		}
		return nil, err
	}
	return &b, nil
}

func (r *BranchRepo) AddBranch(ctx context.Context, name string, capacity int) (*branch.Branch, error) {
	row := r.DB.QueryRowContext(ctx,
		`INSERT INTO branches (name, capacity, status) VALUES ($1, $2, 'active') RETURNING id, name, capacity, status`,
		name, capacity)

	var b branch.Branch
	if err := row.Scan(&b.ID, &b.Name, &b.Capacity, &b.Status); err != nil {
		if confErr := uniqueViolationError(err, fmt.Sprintf("branch %q already exists", name)); confErr != nil {
			return nil, confErr
		}
		if valErr := checkViolationError(err, "capacity must not be negative"); valErr != nil {
			return nil, valErr
		}
		return nil, fmt.Errorf("add branch: %w", err)
	}
	return &b, nil
}

func (r *BranchRepo) SetCapacity(ctx context.Context, branchID, capacity int) error {
	return execUpdateOne(ctx, r.DB,
		`UPDATE branches SET capacity = $1 WHERE id = $2`, []any{capacity, branchID},
		"capacity must not be negative", fmt.Sprintf("branch %d not found", branchID), "set branch capacity")
}

func (r *BranchRepo) SetStatus(ctx context.Context, branchID int, status string) error {
	return execUpdateOne(ctx, r.DB,
		`UPDATE branches SET status = $1 WHERE id = $2`, []any{status, branchID},
		"invalid status", fmt.Sprintf("branch %d not found", branchID), "set branch status")
}
