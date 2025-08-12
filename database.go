package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// DBManager handles the normal PostgreSQL connection for non-event data
type DBManager struct {
	db *sql.DB
}

// NewDBManager creates a new database manager with the given database URL.
// It establishes a connection, verifies connectivity, and initializes required tables.
func NewDBManager(databaseURL string) (*DBManager, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	manager := &DBManager{db: db}
	if err := manager.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database tables: %w", err)
	}

	return manager, nil
}

// initTables creates the necessary tables for the application.
// This method is called automatically during DBManager initialization.
func (dbm *DBManager) initTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS allowed_pubkeys (
		pubkey VARCHAR(64) PRIMARY KEY,
		reason TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := dbm.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create allowed_pubkeys table: %w", err)
	}

	return nil
}

// AddAllowedPubkey adds a pubkey to the allowed list with an optional reason.
// If the pubkey already exists, the operation is ignored (no error returned).
func (dbm *DBManager) AddAllowedPubkey(pubkey, reason string) error {
	if pubkey == "" {
		return fmt.Errorf("pubkey cannot be empty")
	}

	query := `INSERT INTO allowed_pubkeys (pubkey, reason) VALUES ($1, $2) ON CONFLICT (pubkey) DO NOTHING`
	if _, err := dbm.db.Exec(query, pubkey, reason); err != nil {
		return fmt.Errorf("failed to add allowed pubkey %s: %w", pubkey, err)
	}

	return nil
}

// RemoveAllowedPubkey removes a pubkey from the allowed list.
// Returns an error if the pubkey is not found in the allowed list.
func (dbm *DBManager) RemoveAllowedPubkey(pubkey string) error {
	if pubkey == "" {
		return fmt.Errorf("pubkey cannot be empty")
	}

	query := `DELETE FROM allowed_pubkeys WHERE pubkey = $1`
	result, err := dbm.db.Exec(query, pubkey)
	if err != nil {
		return fmt.Errorf("failed to remove allowed pubkey %s: %w", pubkey, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for pubkey %s: %w", pubkey, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pubkey %s not found in allowed list", pubkey)
	}

	return nil
}

// IsAllowedPubkey checks if a pubkey is in the allowed list.
// Returns true if the pubkey is allowed, false otherwise.
func (dbm *DBManager) IsAllowedPubkey(pubkey string) (bool, error) {
	if pubkey == "" {
		return false, nil
	}

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM allowed_pubkeys WHERE pubkey = $1)`
	if err := dbm.db.QueryRow(query, pubkey).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check if pubkey %s is allowed: %w", pubkey, err)
	}

	return exists, nil
}

// GetAllowedPubkeys returns all allowed pubkeys ordered by creation time.
// Returns an empty slice if no pubkeys are found.
func (dbm *DBManager) GetAllowedPubkeys() ([]string, error) {
	query := `SELECT pubkey FROM allowed_pubkeys ORDER BY created_at`
	rows, err := dbm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query allowed pubkeys: %w", err)
	}
	defer rows.Close()

	var pubkeys []string
	for rows.Next() {
		var pubkey string
		if err := rows.Scan(&pubkey); err != nil {
			return nil, fmt.Errorf("failed to scan pubkey row: %w", err)
		}
		pubkeys = append(pubkeys, pubkey)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred while iterating over pubkey rows: %w", err)
	}

	return pubkeys, nil
}

// Close closes the database connection.
// This should be called when the DBManager is no longer needed.
func (dbm *DBManager) Close() error {
	if dbm.db != nil {
		if err := dbm.db.Close(); err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
	}
	return nil
}

// Health checks the database connection health.
// Returns nil if the connection is healthy, an error otherwise.
func (dbm *DBManager) Health() error {
	if dbm.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	if err := dbm.db.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}
