package db

import (
	"context"
	"encoding/json"

	"github.com/RinZ5/converge/backend/internal/core/models"
)

type PostgresRepo struct{}

func (p *PostgresRepo) GetActiveTeachers(ctx context.Context) ([]models.Teacher, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, name FROM teachers WHERE status = 'active' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teachers []models.Teacher
	for rows.Next() {
		var t models.Teacher
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		teachers = append(teachers, t)
	}
	return teachers, rows.Err()
}

func (p *PostgresRepo) ReplaceWeeklyAvailability(ctx context.Context, teacherID int, slots []models.WeeklySlot) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM teacher_weekly_availability WHERE teacher_id = $1`, teacherID); err != nil {
		return err
	}

	for _, s := range slots {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO teacher_weekly_availability (teacher_id, day_of_week, start_time, end_time)
			VALUES ($1, $2, $3, $4)`,
			teacherID, s.DayOfWeek, s.Start, s.End); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *PostgresRepo) SaveRawSubmission(ctx context.Context, teacherID int, rawPayload []byte) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO availability_form_submissions (teacher_id, raw_payload)
		VALUES ($1, $2)`, teacherID, json.RawMessage(rawPayload))
	return err
}
