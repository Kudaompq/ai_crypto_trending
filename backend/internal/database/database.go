package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitDB initializes the SQLite database
func InitDB() error {
	// Create data directory if it doesn't exist
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	dbPath := filepath.Join(dataDir, "opportunities.db")

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Test connection
	if err = DB.Ping(); err != nil {
		return err
	}

	// Create tables
	if err = createTables(); err != nil {
		return err
	}

	log.Println("✅ Database initialized:", dbPath)
	return nil
}

// createTables creates the necessary database tables
func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS opportunities (
		id TEXT PRIMARY KEY,
		symbol TEXT NOT NULL,
		type TEXT NOT NULL,
		strategy TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		entry_price REAL NOT NULL,
		entry_reasons TEXT NOT NULL,
		stop_loss_price REAL NOT NULL,
		stop_loss_distance_pct REAL NOT NULL,
		stop_loss_method TEXT NOT NULL,
		take_profit_levels TEXT NOT NULL,
		risk_reward_ratio REAL NOT NULL,
		risk_amount REAL NOT NULL,
		reward_amount REAL NOT NULL,
		risk_pct REAL NOT NULL,
		reward_pct REAL NOT NULL,
		confidence_score INTEGER NOT NULL,
		confidence_level TEXT NOT NULL,
		confidence_factors TEXT NOT NULL,
		expires_at INTEGER NOT NULL,
		status TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_symbol ON opportunities(symbol);
	CREATE INDEX IF NOT EXISTS idx_status ON opportunities(status);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON opportunities(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_expires_at ON opportunities(expires_at);
	`

	_, err := DB.Exec(schema)
	return err
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
