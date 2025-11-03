package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations runs all SQL migration files from the migrations directory
func RunMigrations(db *sql.DB) error {
	log.Println("🔄 Running database migrations...")

	// Create schema_migrations table to track executed migrations
	createMigrationTable := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		id SERIAL PRIMARY KEY,
		filename VARCHAR(255) UNIQUE NOT NULL,
		executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	if _, err := db.Exec(createMigrationTable); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Mark old migrations as executed if tables already exist (for existing databases)
	var sensorTableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'sensor_readings'
		)
	`).Scan(&sensorTableExists)
	
	if err != nil {
		return fmt.Errorf("failed to check existing tables: %w", err)
	}

	if sensorTableExists {
		// Database already has tables, mark old migrations as executed
		log.Println("📋 Existing database detected, marking old migrations as executed...")
		oldMigrations := []string{
			"001_initial_schema.sql",
			"002_update_schema_filter_mode.sql", 
			"003_add_device_status.sql",
			"004_add_device_id.sql",
		}
		
		for _, filename := range oldMigrations {
			var count int
			db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE filename = $1", filename).Scan(&count)
			
			if count == 0 {
				_, err := db.Exec("INSERT INTO schema_migrations (filename) VALUES ($1)", filename)
				if err != nil {
					log.Printf("⚠️  Warning: Could not mark %s as executed: %v", filename, err)
				} else {
					log.Printf("✅ Marked %s as already executed", filename)
				}
			}
		}
	}

	// Get list of migration files
	migrationsDir := "migrations"
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort SQL files
	var sqlFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, file.Name())
		}
	}
	sort.Strings(sqlFiles)

	// Execute each migration if not already executed
	for _, filename := range sqlFiles {
		// Check if migration already executed
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE filename = $1", filename).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", filename, err)
		}

		if count > 0 {
			log.Printf("⏭️  Skipping already executed migration: %s", filename)
			continue
		}

		// Read migration file
		filePath := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Execute migration
		log.Printf("▶️  Running migration: %s", filename)
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		// Record migration as executed
		_, err = db.Exec("INSERT INTO schema_migrations (filename) VALUES ($1)", filename)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		log.Printf("✅ Successfully executed migration: %s", filename)
	}

	log.Println("✅ All migrations completed successfully")
	return nil
}

// CreateTables creates all necessary tables for the AquaSmart system
func CreateTables(db *sql.DB) error {
	log.Println("Creating database tables...")

	// Create sensor_readings table - stores all sensor data from IoT devices
	sensorReadingsTable := `
	CREATE TABLE IF NOT EXISTS sensor_readings (
		id SERIAL PRIMARY KEY,
		device_id VARCHAR(100) NOT NULL DEFAULT 'stm32_main',
		timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		filter_mode VARCHAR(50) NOT NULL CHECK (filter_mode IN ('drinking_water', 'household_water')),
		flow DECIMAL(10,2) NOT NULL CHECK (flow >= 0),
		ph DECIMAL(4,2) NOT NULL CHECK (ph >= 0 AND ph <= 14),
		turbidity DECIMAL(10,2) NOT NULL CHECK (turbidity >= 0),
		tds DECIMAL(10,2) NOT NULL CHECK (tds >= 0),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		CONSTRAINT unique_device_timestamp UNIQUE(device_id, timestamp)
	);`

	if _, err := db.Exec(sensorReadingsTable); err != nil {
		return fmt.Errorf("failed to create sensor_readings table: %w", err)
	}

	// Create device_status table - stores current operational state of devices
	deviceStatusTable := `
	CREATE TABLE IF NOT EXISTS device_status (
		id SERIAL PRIMARY KEY,
		device_id VARCHAR(100) UNIQUE NOT NULL,
		current_filter_mode VARCHAR(50) NOT NULL DEFAULT 'drinking_water' 
			CHECK (current_filter_mode IN ('drinking_water', 'household_water')),
		last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		is_active BOOLEAN DEFAULT true,
		total_readings INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	if _, err := db.Exec(deviceStatusTable); err != nil {
		return fmt.Errorf("failed to create device_status table: %w", err)
	}

	// Insert default device status for STM32
	insertDefaultDevice := `
	INSERT INTO device_status (device_id, current_filter_mode, last_seen, is_active, total_readings)
	VALUES ('stm32_main', 'drinking_water', NOW(), true, 0)
	ON CONFLICT (device_id) DO NOTHING;`

	if _, err := db.Exec(insertDefaultDevice); err != nil {
		log.Printf("Warning: Failed to insert default device status: %v", err)
	}

	// Create indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_sensor_readings_timestamp ON sensor_readings(timestamp DESC);",
		"CREATE INDEX IF NOT EXISTS idx_sensor_readings_device_id ON sensor_readings(device_id);",
		"CREATE INDEX IF NOT EXISTS idx_sensor_readings_filter_mode ON sensor_readings(filter_mode);",
		"CREATE INDEX IF NOT EXISTS idx_device_status_device_id ON device_status(device_id);",
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}

	log.Println("✅ Database tables created successfully")
	return nil
}

// DropTables drops all tables (useful for testing)
func DropTables(db *sql.DB) error {
	log.Println("Dropping database tables...")

	tables := []string{
		"sensor_readings",
		"device_status",
	}

	for _, table := range tables {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", table)
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	log.Println("✅ Database tables dropped successfully")
	return nil
}

// CheckTablesExist checks if all required tables exist
func CheckTablesExist(db *sql.DB) error {
	requiredTables := []string{
		"sensor_readings",
		"device_status",
	}

	for _, table := range requiredTables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = $1
		);`

		err := db.QueryRow(query, table).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}

		if !exists {
			return fmt.Errorf("table %s does not exist", table)
		}
	}

	log.Println("✅ All required tables exist")
	return nil
}