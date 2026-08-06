package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
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

	demoUserIDs, err := seedDemoUsers(database)
	if err != nil {
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

	bookingCount, err := seedBookings(database, demoUserIDs, teacherIDs, subjectIDs, branchIDs)
	if err != nil {
		return err
	}

	log.Printf("Seed complete: %d branches, %d subjects, %d teachers, %d bookings",
		len(branchIDs), len(subjectIDs), len(teacherIDs), bookingCount)
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
// exercised out of the box. These use a shared default password, so they are
// gated behind SEED_DEMO_USERS and skipped unless it is explicitly enabled;
// this keeps known-credential accounts out of shared or production databases.
// Idempotent: the upsert returns the id whether the user was inserted or
// already existed, and the parent link uses ON CONFLICT.
// Returns the seeded user ids by name, or nil when seeding is disabled, so
// seedBookings knows whether there are students to book classes for.
func seedDemoUsers(database *sql.DB) (map[string]int, error) {
	if enabled, _ := strconv.ParseBool(os.Getenv("SEED_DEMO_USERS")); !enabled {
		log.Println("SEED_DEMO_USERS not enabled; skipping demo user seeding")
		return nil, nil
	}

	demo := []struct{ name, password, role string }{
		{"student1", "password", "student"},
		{"student2", "password", "student"},
		{"parent1", "password", "parent"},
	}

	ids := make(map[string]int, len(demo))
	for _, u := range demo {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash %s: %w", u.name, err)
		}
		var id int
		err = database.QueryRow(`
			INSERT INTO users (name, password_hash, role) VALUES ($1, $2, $3)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, u.name, string(hash), u.role).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("seed user %s: %w", u.name, err)
		}
		ids[u.name] = id
	}

	for _, student := range []string{"student1", "student2"} {
		if _, err := database.Exec(`
			INSERT INTO parent_students (parent_id, student_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, ids["parent1"], ids[student]); err != nil {
			return nil, fmt.Errorf("link parent1 -> %s: %w", student, err)
		}
	}

	log.Println("Seeded demo users student1/student2/parent1; parent1 guards student1,student2")
	return ids, nil
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

// seedBookings creates confirmed classes for the demo students, split either
// side of today so both the upcoming and past views have something to show.
// It depends on the demo students, so it is skipped whenever seedDemoUsers was
// (there is nobody to book a class for). Bookings are not truncated directly:
// TRUNCATE ... CASCADE on teachers/branches/subjects already clears them.
//
// The times are deliberately laid out so no teacher is double-booked, since
// the bookings table carries a gist EXCLUDE constraint on
// (teacher_id, tstzrange(start_time, end_time)) that would reject an overlap.
func seedBookings(database *sql.DB, userIDs map[string]int, teacherIDs, subjectIDs, branchIDs []int) (int, error) {
	if len(userIDs) == 0 {
		log.Println("No demo students seeded; skipping booking seeding")
		return 0, nil
	}

	// Anchored to local midnight so a class seeded for "tomorrow at 09:00"
	// lands at that wall-clock time rather than drifting with the run time.
	loc := shared.LoadLocation()
	now := time.Now().In(loc)
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	type bookingSeed struct {
		student   string
		teacher   int // index into teacherIDs
		subject   int // index into subjectIDs
		branch    int // index into branchIDs
		dayOffset int // relative to today; negative is a past class
		hour      int
	}

	// Alice=0, Bob=1, Carol=2, David=3 (deactivated, unbooked), Eva=4.
	// Subjects: Mathematics=0, Physics=1, English=2, History=3, CS=4, Art=5.
	// Branches: Main Campus=0, Downtown=1, Westside=2, Online=3.
	seeds := []bookingSeed{
		{"student1", 0, 0, 0, -7, 9},
		{"student1", 1, 2, 1, -3, 10},
		{"student2", 2, 4, 2, -2, 13},
		// Two classes on the same day for student1, so the day grouping shows
		// more than one card under a heading.
		{"student1", 0, 0, 0, 1, 9},
		{"student1", 2, 5, 3, 1, 14},
		{"student2", 1, 3, 1, 2, 11},
		{"student1", 0, 1, 0, 5, 10},
		{"student2", 4, 2, 0, 5, 9},
	}

	for _, s := range seeds {
		studentID, ok := userIDs[s.student]
		if !ok {
			return 0, fmt.Errorf("seed booking: unknown demo student %q", s.student)
		}
		start := midnight.AddDate(0, 0, s.dayOffset).Add(time.Duration(s.hour) * time.Hour)
		_, err := database.Exec(`
			INSERT INTO bookings (teacher_id, branch_id, subject_id, start_time, end_time, student_id)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			teacherIDs[s.teacher], branchIDs[s.branch], subjectIDs[s.subject],
			start, start.Add(time.Hour), studentID,
		)
		if err != nil {
			return 0, fmt.Errorf("insert booking (student=%s, start=%s): %w", s.student, start, err)
		}
	}

	log.Printf("Seeded %d bookings for the demo students", len(seeds))
	return len(seeds), nil
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
