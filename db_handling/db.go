package dbhandling

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

// Create a new database to hold instrumentation logs
func InitDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", "postgres://postgres:password@localhost:5432/instrumentation_db?sslmode=disable")
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

type Row struct {
	ID              int       `db:"id"`
	Timestamp       time.Time `db:"timestamp"`
	Instrumentation string    `db:"instrumentation"`
	ErrorStatus     bool      `db:"error_status"`
}

// POST one row to the database, including the instrumentation type and if there was an error
func POST(db *sql.DB, instrumentationType string, hasError bool) error {
	query := `INSERT INTO instrumentation_logs (instrumentation, error_status) VALUES ($1, $2)`
	_, err := db.Exec(query, instrumentationType, hasError)
	return err
}

// GET the last n rows from the database
func GET(db *sql.DB, n int) ([]Row, error) {
	query := `SELECT id, instrumentation, timestamp, error_status 
              FROM instrumentation_logs 
              ORDER BY timestamp DESC 
              LIMIT $1`

	rows, err := db.Query(query, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []Row
	for rows.Next() {
		var log Row
		err := rows.Scan(&log.ID, &log.Instrumentation, &log.Timestamp, &log.ErrorStatus)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}
