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

var db *sql.DB

func InitDB() error {
	_ = godotenv.Load()
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "")
	dbname := getEnv("POSTGRES_DB", "postgres")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", url.QueryEscape(user), url.QueryEscape(password), host, port, dbname)
	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func AutoMigrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS teachers (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			contact_details VARCHAR(255),
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
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migration failed: %w\n%s", err, q)
		}
	}
	log.Println("Database schema ready")
	return nil
}

func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
