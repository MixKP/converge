package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func Seed(database *sql.DB) error {
	log.Println("Seeding database...")

	if err := truncateSeedTables(database); err != nil {
		return err
	}

	if err := seedAdminUser(database); err != nil {
		return err
	}

	if err := seedDemoUsers(database); err != nil {
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

// seedAdminUser bootstraps the first admin. Idempotent: users is not truncated
// and ON CONFLICT keeps an existing admin unchanged across re-seeds.
// ADMIN_PASSWORD is required — there is no default, so a real deployment can
// never end up with a guessable admin credential.
func seedAdminUser(database *sql.DB) error {
	name := getEnv("ADMIN_USERNAME", "admin")
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		return fmt.Errorf("ADMIN_PASSWORD is required to seed the admin user")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	res, err := database.Exec(
		`INSERT INTO users (name, password_hash, role) VALUES ($1, $2, 'admin')
		 ON CONFLICT (name) DO NOTHING`,
		name, string(hash),
	)
	if err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("Seeded admin user %q", name)
	} else {
		log.Printf("Admin user %q already exists, left unchanged", name)
	}
	return nil
}

// seedDemoUsers creates login accounts for each non-admin role (student,
// parent) and links the parent to the students, so the RBAC flows can be
// exercised out of the box. Idempotent: the upsert returns the id whether the
// user was inserted or already existed, and the parent link uses ON CONFLICT.
// These are demo credentials (all password "password"); the admin account is
// the only one gated behind ADMIN_PASSWORD.
func seedDemoUsers(database *sql.DB) error {
	demo := []struct{ name, password, role string }{
		{"student1", "password", "student"},
		{"student2", "password", "student"},
		{"parent1", "password", "parent"},
	}

	ids := make(map[string]int, len(demo))
	for _, u := range demo {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash %s: %w", u.name, err)
		}
		var id int
		err = database.QueryRow(`
			INSERT INTO users (name, password_hash, role) VALUES ($1, $2, $3)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, u.name, string(hash), u.role).Scan(&id)
		if err != nil {
			return fmt.Errorf("seed user %s: %w", u.name, err)
		}
		ids[u.name] = id
	}

	for _, student := range []string{"student1", "student2"} {
		if _, err := database.Exec(`
			INSERT INTO parent_students (parent_id, student_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, ids["parent1"], ids[student]); err != nil {
			return fmt.Errorf("link parent1 -> %s: %w", student, err)
		}
	}

	log.Printf("Seeded demo users student1/student2/parent1 (password %q); parent1 guards student1,student2", "password")
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
