package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/Capstone-E1/aquasmart_backend/config"
)

// DB holds the database connection
type DB struct {
	*sql.DB
}

// Connect establishes connection to PostgreSQL database
func Connect(cfg config.DatabaseConfig) (*DB, error) {
	// Build connection string for Aiven PostgreSQL with additional SSL parameters
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s sslrootcert=disable sslcert=disable sslkey=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	log.Printf("Connecting to Aiven database at %s:%s/%s", cfg.Host, cfg.Port, cfg.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	log.Println("Successfully connected to PostgreSQL database")

	return &DB{db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}

// BuildConnectionString builds a PostgreSQL connection string
func BuildConnectionString(cfg config.DatabaseConfig) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
}