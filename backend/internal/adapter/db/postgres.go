package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func InitDB() (*sql.DB, error) {
	_ = godotenv.Load()
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "")
	dbname := getEnv("POSTGRES_DB", "postgres")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", url.QueryEscape(user), url.QueryEscape(password), host, port, dbname)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
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
			name VARCHAR(100) NOT NULL UNIQUE
		)`,

		`CREATE EXTENSION IF NOT EXISTS btree_gist;`,
		`CREATE TABLE IF NOT EXISTS bookings (
			id SERIAL PRIMARY KEY,
			teacher_id INTEGER NOT NULL REFERENCES teachers(id),
			branch_id INTEGER NOT NULL REFERENCES branches(id),
			subject_id INTEGER NOT NULL REFERENCES subjects(id),
			start_time TIMESTAMPTZ NOT NULL,
			end_time TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CHECK (start_time < end_time),
			EXCLUDE USING gist (teacher_id WITH =, tstzrange(start_time, end_time) WITH &&)
		)`,
	}
	for _, q := range queries {
		if _, err := database.Exec(q); err != nil {
			return fmt.Errorf("migration failed: %w\n%s", err, q)
		}
	}
	log.Println("Database schema ready")
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
