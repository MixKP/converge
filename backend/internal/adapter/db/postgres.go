package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

const (
	dbPingTimeout = 10 * time.Second
	dbRetryDelay  = 2 * time.Second
	dbMaxRetries  = 5
)

func InitDB() (*sql.DB, error) {
	_ = godotenv.Load()
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "")
	dbname := getEnv("POSTGRES_DB", "postgres")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", url.QueryEscape(user), url.QueryEscape(password), host, port, dbname)
	return InitDBWithConnString(dsn)
}

func InitDBWithConnString(connStr string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	var pingErr error
	for range dbMaxRetries {
		ctx, cancel := context.WithTimeout(context.Background(), dbPingTimeout)
		pingErr = db.PingContext(ctx)
		cancel()
		if pingErr == nil {
			return db, nil
		}
		time.Sleep(dbRetryDelay)
	}

	db.Close()
	return nil, fmt.Errorf("ping database after %d retries: %w", dbMaxRetries, pingErr)
}

func AutoMigrate(database *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS teachers (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(255),
			status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active','deactivated')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS teacher_availability (
			id SERIAL PRIMARY KEY,
			teacher_id INTEGER NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
			day_of_week INTEGER CHECK (day_of_week BETWEEN 0 AND 6),
			start_time TIME NOT NULL,
			end_time TIME NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(teacher_id, day_of_week, start_time, end_time),
			CHECK (start_time < end_time)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_teacher_availability_teacher ON teacher_availability(teacher_id)`,

		`CREATE TABLE IF NOT EXISTS form_submission (
			id SERIAL PRIMARY KEY,
			teacher_id INTEGER REFERENCES teachers(id) ON DELETE SET NULL,
			raw_payload JSONB NOT NULL,
			submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS subjects (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL UNIQUE
		)`,

		`CREATE TABLE IF NOT EXISTS teacher_subjects (
			teacher_id INTEGER NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
			subject_id INTEGER NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
			PRIMARY KEY (teacher_id, subject_id)
		)`,

		`CREATE TABLE IF NOT EXISTS branches (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL UNIQUE,
			capacity INTEGER NOT NULL DEFAULT 0 CHECK (capacity >= 0)
		)`,

		`CREATE EXTENSION IF NOT EXISTS btree_gist;`,
		`CREATE TABLE IF NOT EXISTS bookings (
			id SERIAL PRIMARY KEY,
			teacher_id INTEGER NOT NULL REFERENCES teachers(id),
			branch_id INTEGER NOT NULL REFERENCES branches(id),
			subject_id INTEGER NOT NULL REFERENCES subjects(id),
			start_time TIMESTAMPTZ NOT NULL,
			end_time TIMESTAMPTZ NOT NULL,
			student_id INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CHECK (start_time < end_time),
			EXCLUDE USING gist (teacher_id WITH =, tstzrange(start_time, end_time) WITH &&)
		)`,
		`ALTER TABLE branches ADD COLUMN IF NOT EXISTS capacity INTEGER NOT NULL DEFAULT 0`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conrelid = 'branches'::regclass
				  AND conname = 'branches_capacity_check'
			) THEN
				ALTER TABLE branches ADD CONSTRAINT branches_capacity_check CHECK (capacity >= 0);
			END IF;
		END $$;`,
		`ALTER TABLE teachers ADD COLUMN IF NOT EXISTS gender VARCHAR(20) CHECK (gender IN ('male','female','lgbtq+'))`,
		`UPDATE teachers SET gender = 'male' WHERE gender IS NULL`,
		`ALTER TABLE teachers ALTER COLUMN gender SET NOT NULL`,

		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(20) NOT NULL DEFAULT 'student' CHECK (role IN ('admin','student','parent')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'student' CHECK (role IN ('admin','student','parent'))`,
		`DELETE FROM users WHERE role = 'teacher'`,
		`ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check`,
		`ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin','student','parent'))`,

		// bookings.student_id and its FK are added here (after users exists).
		// client_name is superseded by student_id; the student's name is read via a join.
		`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS student_id INTEGER`,
		`ALTER TABLE bookings DROP COLUMN IF EXISTS client_name`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conrelid = 'bookings'::regclass
				  AND conname = 'bookings_student_fk'
			) THEN
				ALTER TABLE bookings ADD CONSTRAINT bookings_student_fk
					FOREIGN KEY (student_id) REFERENCES users(id);
			END IF;
		END $$;`,
		`DELETE FROM bookings WHERE student_id IS NULL`,
		`ALTER TABLE bookings ALTER COLUMN student_id SET NOT NULL`,
		`CREATE TABLE IF NOT EXISTS parent_students (
			parent_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			student_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			PRIMARY KEY (parent_id, student_id)
		)`,

		`CREATE TABLE IF NOT EXISTS commute_config (
			id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
			commute_minutes INTEGER NOT NULL DEFAULT 30 CHECK (commute_minutes >= 0)
		)`,
		`INSERT INTO commute_config (id, commute_minutes) VALUES (1, 30) ON CONFLICT (id) DO NOTHING`,
	}
	for _, q := range queries {
		if _, err := database.Exec(q); err != nil {
			return fmt.Errorf("migration failed: %w\n%s", err, q)
		}
	}
	slog.Info("Database schema ready")
	return nil
}

func CloseDB(database *sql.DB) error {
	if database != nil {
		return database.Close()
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
