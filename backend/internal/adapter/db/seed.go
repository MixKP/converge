package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func Seed(database *sql.DB) error {
	log.Println("Seeding database...")

	if err := truncateSeedTables(database); err != nil {
		return err
	}

	branchIDs, err := seedBranches(database)
	if err != nil {
		return err
	}

	subjectIDs, err := seedSubjects(database)
	if err != nil {
		return err
	}

	teacherIDs, err := seedTeachers(database)
	if err != nil {
		return err
	}

	if err := seedTeacherSubjects(database, teacherIDs, subjectIDs); err != nil {
		return err
	}

	if err := seedTeacherAvailability(database, teacherIDs); err != nil {
		return err
	}

	if err := seedFormSubmissions(database, teacherIDs); err != nil {
		return err
	}

	log.Printf("Seed complete: %d branches, %d subjects, %d teachers",
		len(branchIDs), len(subjectIDs), len(teacherIDs))
	return nil
}

func truncateSeedTables(database *sql.DB) error {
	_, err := database.Exec(`TRUNCATE form_submission, teacher_availability, teacher_subjects, teachers, subjects, branches RESTART IDENTITY CASCADE`)
	if err != nil {
		return fmt.Errorf("truncate failed: %w", err)
	}
	return nil
}

func seedBranches(database *sql.DB) ([]int, error) {
	branches := []string{
		"Main Campus",
		"Downtown",
		"Westside",
		"Online",
	}

	ids := make([]int, 0, len(branches))
	for _, name := range branches {
		var id int
		err := database.QueryRow(
			`INSERT INTO branches (name) VALUES ($1) RETURNING id`, name,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("insert branch %q: %w", name, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func seedSubjects(database *sql.DB) ([]int, error) {
	subjects := []string{
		"Mathematics",
		"Physics",
		"English",
		"History",
		"Computer Science",
		"Art",
	}

	ids := make([]int, 0, len(subjects))
	for _, name := range subjects {
		var id int
		err := database.QueryRow(
			`INSERT INTO subjects (name) VALUES ($1) RETURNING id`, name,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("insert subject %q: %w", name, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

type teacherSeed struct {
	name   string
	email  string
	gender string
	status string
}

func seedTeachers(database *sql.DB) ([]int, error) {
	teachers := []teacherSeed{
		{name: "Alice Johnson", email: "alice@example.com", gender: "female", status: "active"},
		{name: "Bob Smith", email: "bob@example.com", gender: "male", status: "active"},
		{name: "Carol Williams", email: "carol@example.com", gender: "female", status: "active"},
		{name: "David Brown", email: "david@example.com", gender: "male", status: "deactivated"},
		{name: "Eva Martinez", email: "eva@example.com", gender: "female", status: "active"},
	}

	ids := make([]int, 0, len(teachers))
	for _, t := range teachers {
		var id int
		err := database.QueryRow(
			`INSERT INTO teachers (name, email, gender, status) VALUES ($1, $2, $3, $4) RETURNING id`,
			t.name, t.email, t.gender, t.status,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("insert teacher %q: %w", t.name, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func seedTeacherSubjects(database *sql.DB, teacherIDs, subjectIDs []int) error {
	// Alice: Mathematics, Physics
	// Bob: English, History
	// Carol: Computer Science, Art, Mathematics
	// David: Physics
	// Eva: English, Computer Science
	mappings := map[int][]int{
		teacherIDs[0]: {subjectIDs[0], subjectIDs[1]},
		teacherIDs[1]: {subjectIDs[2], subjectIDs[3]},
		teacherIDs[2]: {subjectIDs[4], subjectIDs[5], subjectIDs[0]},
		teacherIDs[3]: {subjectIDs[1]},
		teacherIDs[4]: {subjectIDs[2], subjectIDs[4]},
	}

	for teacherID, subjectIdxs := range mappings {
		for _, subjectID := range subjectIdxs {
			_, err := database.Exec(
				`INSERT INTO teacher_subjects (teacher_id, subject_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				teacherID, subjectID,
			)
			if err != nil {
				return fmt.Errorf("insert teacher_subject (teacher=%d, subject=%d): %w", teacherID, subjectID, err)
			}
		}
	}
	return nil
}

func seedTeacherAvailability(database *sql.DB, teacherIDs []int) error {
	type slot struct {
		dayOfWeek int
		start     string
		end       string
	}

	schedules := map[int][]slot{
		teacherIDs[0]: { // Alice: Mon/Wed/Fri 09:00-12:00
			{0, "09:00", "12:00"},
			{2, "09:00", "12:00"},
			{4, "09:00", "12:00"},
		},
		teacherIDs[1]: { // Bob: Tue/Thu 10:00-15:00
			{1, "10:00", "15:00"},
			{3, "10:00", "15:00"},
		},
		teacherIDs[2]: { // Carol: Mon-Thu 13:00-17:00
			{0, "13:00", "17:00"},
			{1, "13:00", "17:00"},
			{2, "13:00", "17:00"},
			{3, "13:00", "17:00"},
		},
		teacherIDs[3]: { // David: Mon 08:00-10:00 (deactivated, but has schedule)
			{0, "08:00", "10:00"},
		},
		teacherIDs[4]: { // Eva: Tue/Thu/Sat 09:00-14:00
			{1, "09:00", "14:00"},
			{3, "09:00", "14:00"},
			{5, "09:00", "14:00"},
		},
	}

	for teacherID, slots := range schedules {
		for _, s := range slots {
			_, err := database.Exec(
				`INSERT INTO teacher_availability (teacher_id, day_of_week, start_time, end_time)
				 VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
				teacherID, s.dayOfWeek, s.start, s.end,
			)
			if err != nil {
				return fmt.Errorf("insert availability (teacher=%d, day=%d): %w", teacherID, s.dayOfWeek, err)
			}
		}
	}
	return nil
}

func seedFormSubmissions(database *sql.DB, teacherIDs []int) error {
	type formSubmission struct {
		teacherID int
		payload   map[string]interface{}
	}

	submissions := []formSubmission{
		{
			teacherID: teacherIDs[0],
			payload: map[string]interface{}{
				"teacher_id": teacherIDs[0],
				"weekly": []map[string]interface{}{
					{"day_of_week": 0, "start": "09:00", "end": "12:00"},
					{"day_of_week": 2, "start": "09:00", "end": "12:00"},
					{"day_of_week": 4, "start": "09:00", "end": "12:00"},
				},
				"submitted_at": time.Now().Format(time.RFC3339),
			},
		},
		{
			teacherID: teacherIDs[1],
			payload: map[string]interface{}{
				"teacher_id": teacherIDs[1],
				"weekly": []map[string]interface{}{
					{"day_of_week": 1, "start": "10:00", "end": "15:00"},
					{"day_of_week": 3, "start": "10:00", "end": "15:00"},
				},
				"submitted_at": time.Now().Format(time.RFC3339),
			},
		},
	}

	for _, s := range submissions {
		payloadBytes, err := json.Marshal(s.payload)
		if err != nil {
			return fmt.Errorf("marshal form payload: %w", err)
		}
		_, err = database.Exec(
			`INSERT INTO form_submission (teacher_id, raw_payload) VALUES ($1, $2)`,
			s.teacherID, payloadBytes,
		)
		if err != nil {
			return fmt.Errorf("insert form_submission (teacher=%d): %w", s.teacherID, err)
		}
	}
	return nil
}
