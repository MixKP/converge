package db

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/RinZ5/converge/backend/internal/core/models"
)

type PostgresRepo struct {
	DB *sql.DB
}

func NewPostgresRepo(database *sql.DB) *PostgresRepo {
	return &PostgresRepo{DB: database}
}

func (p *PostgresRepo) GetActiveTeachers(ctx context.Context) ([]models.Teacher, error) {
	rows, err := p.DB.QueryContext(ctx, `SELECT id, name, email FROM teachers WHERE status = 'active' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teachers []models.Teacher
	for rows.Next() {
		var t models.Teacher
		if err := rows.Scan(&t.ID, &t.Name, &t.Email); err != nil {
			return nil, err
		}
		teachers = append(teachers, t)
	}
	return teachers, rows.Err()
}

func (p *PostgresRepo) GetTeachersBySubject(ctx context.Context, subjectID int) ([]models.Teacher, error) {
	rows, err := p.DB.QueryContext(ctx, `
		SELECT t.id, t.name, t.email
		FROM teachers t
		JOIN teacher_subjects ts ON t.id = ts.teacher_id
		WHERE ts.subject_id = $1 AND t.status = 'active'
		ORDER BY t.name`, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teachers []models.Teacher
	for rows.Next() {
		var t models.Teacher
		if err := rows.Scan(&t.ID, &t.Name, &t.Email); err != nil {
			return nil, err
		}
		teachers = append(teachers, t)
	}
	return teachers, rows.Err()
}

func (p *PostgresRepo) GetBranches(ctx context.Context) ([]models.Branch, error) {
	rows, err := p.DB.QueryContext(ctx, `SELECT id, name FROM branches ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var branches []models.Branch
	for rows.Next() {
		var b models.Branch
		if err := rows.Scan(&b.ID, &b.Name); err != nil {
			return nil, err
		}
		branches = append(branches, b)
	}
	return branches, rows.Err()
}

func (p *PostgresRepo) GetSubjects(ctx context.Context) ([]models.Subject, error) {
	rows, err := p.DB.QueryContext(ctx, `SELECT id, name FROM subjects ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []models.Subject
	for rows.Next() {
		var s models.Subject
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		subjects = append(subjects, s)
	}
	return subjects, rows.Err()
}

func (p *PostgresRepo) ReplaceWeeklyAvailability(ctx context.Context, teacherID int, slots []models.WeeklySlot) error {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM teacher_availability WHERE teacher_id = $1`, teacherID); err != nil {
		return err
	}

	for _, s := range slots {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO teacher_availability (teacher_id, day_of_week, start_time, end_time)
			VALUES ($1, $2, $3, $4)`,
			teacherID, s.DayOfWeek, s.Start, s.End); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *PostgresRepo) GetAllAvailability(ctx context.Context) ([]models.TeacherAvailability, error) {
	rows, err := p.DB.QueryContext(ctx, `
		SELECT t.id, t.name, t.email, ta.day_of_week, ta.start_time, ta.end_time
		FROM teachers t
		JOIN teacher_availability ta ON t.id = ta.teacher_id
		WHERE t.status = 'active'
		ORDER BY t.name, ta.day_of_week, ta.start_time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.TeacherAvailability, 0)
	var current *models.TeacherAvailability

	for rows.Next() {
		var teacherID int
		var name, email string
		var slot models.WeeklySlot
		if err := rows.Scan(&teacherID, &name, &email, &slot.DayOfWeek, &slot.Start, &slot.End); err != nil {
			return nil, err
		}
		if current == nil || current.Teacher.ID != teacherID {
			result = append(result, models.TeacherAvailability{
				Teacher: models.Teacher{ID: teacherID, Name: name, Email: email},
				Weekly:  []models.WeeklySlot{slot},
			})
			current = &result[len(result)-1]
		} else {
			current.Weekly = append(current.Weekly, slot)
		}
	}
	return result, rows.Err()
}

func (p *PostgresRepo) SaveRawSubmission(ctx context.Context, teacherID int, rawPayload []byte) error {
	_, err := p.DB.ExecContext(ctx, `
		INSERT INTO form_submission (teacher_id, raw_payload)
		VALUES ($1, $2)`, teacherID, json.RawMessage(rawPayload))
	return err
}
