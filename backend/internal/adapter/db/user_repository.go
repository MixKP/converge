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

// CreateParent inserts a parent user and links it to the given students in one
// transaction. Every student_id must reference an existing user with role
// 'student', otherwise the whole operation is rolled back.
func (r *UserRepo) CreateParent(ctx context.Context, name, passwordHash string, studentIDs []int) (*user.User, error) {
	if len(studentIDs) == 0 {
		return nil, &shared.ValidationError{Msg: "a parent must be linked to at least one student"}
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var u user.User
	err = tx.QueryRowContext(ctx, `
		INSERT INTO users (name, password_hash, role) VALUES ($1, $2, 'parent')
		RETURNING id, name, role`, name, passwordHash).Scan(&u.ID, &u.Name, &u.Role)
	if err != nil {
		if confErr := uniqueViolationError(err, fmt.Sprintf("username %q already taken", name)); confErr != nil {
			return nil, confErr
		}
		return nil, fmt.Errorf("create parent: %w", err)
	}

	seen := make(map[int]struct{}, len(studentIDs))
	uniqueIDs := make([]int, 0, len(studentIDs))
	for _, id := range studentIDs {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO parent_students (parent_id, student_id)
		SELECT $1, id FROM users WHERE id = ANY($2) AND role = 'student'`,
		u.ID, uniqueIDs)
	if err != nil {
		return nil, fmt.Errorf("link parent students: %w", err)
	}
	if n, _ := res.RowsAffected(); int(n) != len(uniqueIDs) {
		return nil, &shared.ValidationError{Msg: "one or more student_ids are not valid students"}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &u, nil
}

func scanUsers(rows *sql.Rows) ([]user.User, error) {
	var users []user.User
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepo) ListStudents(ctx context.Context) ([]user.User, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, name, role FROM users WHERE role = 'student' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (r *UserRepo) ListParentsWithStudents(ctx context.Context) ([]user.ParentWithStudents, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT p.id, p.name, p.role, s.id, s.name, s.role
		FROM users p
		LEFT JOIN parent_students ps ON ps.parent_id = p.id
		LEFT JOIN users s ON s.id = ps.student_id
		WHERE p.role = 'parent'
		ORDER BY p.id, s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []user.ParentWithStudents
	indexByID := map[int]int{}
	for rows.Next() {
		var pID int
		var pName, pRole string
		var sID sql.NullInt64
		var sName, sRole sql.NullString
		if err := rows.Scan(&pID, &pName, &pRole, &sID, &sName, &sRole); err != nil {
			return nil, err
		}
		idx, ok := indexByID[pID]
		if !ok {
			result = append(result, user.ParentWithStudents{
				ID: pID, Name: pName, Role: pRole,
				Students: []user.User{},
			})
			idx = len(result) - 1
			indexByID[pID] = idx
		}
		if sID.Valid {
			result[idx].Students = append(result[idx].Students, user.User{
				ID: int(sID.Int64), Name: sName.String, Role: sRole.String,
			})
		}
	}
	return result, rows.Err()
}

func (r *UserRepo) StudentsForParent(ctx context.Context, parentID int) ([]user.User, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT u.id, u.name, u.role
		FROM parent_students ps JOIN users u ON u.id = ps.student_id
		WHERE ps.parent_id = $1 ORDER BY u.id`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

// assertRole verifies a user exists and has the expected role.
func (r *UserRepo) assertRole(ctx context.Context, id int, want string) error {
	var role string
	err := r.DB.QueryRowContext(ctx, `SELECT role FROM users WHERE id = $1`, id).Scan(&role)
	if err == sql.ErrNoRows {
		return &shared.NotFoundError{Msg: fmt.Sprintf("user %d not found", id)}
	}
	if err != nil {
		return err
	}
	if role != want {
		return &shared.ValidationError{Msg: fmt.Sprintf("user %d is not a %s", id, want)}
	}
	return nil
}

func (r *UserRepo) LinkParentStudent(ctx context.Context, parentID, studentID int) error {
	if err := r.assertRole(ctx, parentID, "parent"); err != nil {
		return err
	}
	if err := r.assertRole(ctx, studentID, "student"); err != nil {
		return err
	}
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO parent_students (parent_id, student_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		parentID, studentID)
	return err
}

func (r *UserRepo) UnlinkParentStudent(ctx context.Context, parentID, studentID int) error {
	res, err := r.DB.ExecContext(ctx,
		`DELETE FROM parent_students WHERE parent_id = $1 AND student_id = $2`, parentID, studentID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return &shared.NotFoundError{Msg: fmt.Sprintf("parent %d is not linked to student %d", parentID, studentID)}
	}
	return nil
}

func (r *UserRepo) StudentIDsForParent(ctx context.Context, parentID int) ([]int, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT student_id FROM parent_students WHERE parent_id = $1 ORDER BY student_id`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
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
