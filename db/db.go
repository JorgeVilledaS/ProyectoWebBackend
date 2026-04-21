package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {
	// 1. Si existe DATABASE_URL (Railway), úsala directamente
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			return nil, fmt.Errorf("error opening database (DATABASE_URL): %w", err)
		}

		if err := db.Ping(); err != nil {
			return nil, fmt.Errorf("error connecting to database (DATABASE_URL): %w", err)
		}

		return db, nil
	}

	// 2. Si no existe, usa variables locales (.env)
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "series_tracker")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return db, nil
}

// RunMigrations crea las tablas necesarias si no existen.
func RunMigrations(db *sql.DB) error {
	queries := []string{
		// Tabla principal de series
		`CREATE TABLE IF NOT EXISTS series (
			id               SERIAL PRIMARY KEY,
			name             TEXT NOT NULL,
			current_episode  INT NOT NULL DEFAULT 1,
			total_episodes   INT NOT NULL,
			image_url        TEXT NOT NULL DEFAULT '',
			created_at       TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		// Tabla de ratings (relación 1:N — múltiples ratings por serie)
		`CREATE TABLE IF NOT EXISTS ratings (
			id         SERIAL PRIMARY KEY,
			series_id  INT NOT NULL REFERENCES series(id) ON DELETE CASCADE,
			score      INT NOT NULL CHECK (score >= 1 AND score <= 10),
			comment    TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migration error: %w", err)
		}
	}

	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}