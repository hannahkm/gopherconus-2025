package dbhandling

import (
	"database/sql"

	"github.com/XSAM/otelsql"
	_ "github.com/lib/pq"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

// Create a new database to hold instrumentation logs
func Manual_InitDB() (*sql.DB, error) {
	db, err := otelsql.Open("postgres",
		"postgres://postgres:password@localhost:5432/instrumentation_db?sslmode=disable",
		otelsql.WithAttributes(
			semconv.DBSystemPostgreSQL,
		),
	)
	if err != nil {
		return nil, err
	}

	err = otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	))
	if err != nil {
		return nil, err
	}

	cmd := `
    CREATE TABLE IF NOT EXISTS instrumentation_logs (
        id SERIAL PRIMARY KEY,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        instrumentation VARCHAR(20) NOT NULL,
        error_status BOOLEAN DEFAULT FALSE
    )`

	_, err = db.Exec(cmd)
	if err != nil {
		return nil, err
	}

	return db, nil
}
