package database

import (
	"database/sql"
	"fmt"
	"log"
)

// CreateTables creates all necessary tables for the AquaSmart system
func CreateTables(db *sql.DB) error {
	log.Println("Creating database tables...")

	// Create sensor_readings table
	sensorReadingsTable := `
	CREATE TABLE IF NOT EXISTS sensor_readings (
		id SERIAL PRIMARY KEY,
		device_id VARCHAR(100) NOT NULL DEFAULT 'default',
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

	// Create water_quality_assessments table
	waterQualityTable := `
	CREATE TABLE IF NOT EXISTS water_quality_assessments (
		id SERIAL PRIMARY KEY,
		sensor_reading_id INTEGER REFERENCES sensor_readings(id) ON DELETE CASCADE,
		device_id VARCHAR(100) NOT NULL,
		timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
		filter_mode VARCHAR(50) NOT NULL,
		flow DECIMAL(10,2) NOT NULL,
		ph DECIMAL(4,2) NOT NULL,
		ph_status VARCHAR(50) NOT NULL,
		turbidity DECIMAL(10,2) NOT NULL,
		turbidity_status VARCHAR(50) NOT NULL,
		tds DECIMAL(10,2) NOT NULL,
		tds_status VARCHAR(50) NOT NULL,
		overall_quality VARCHAR(50) NOT NULL CHECK (overall_quality IN ('Excellent', 'Good', 'Poor', 'Danger')),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	if _, err := db.Exec(waterQualityTable); err != nil {
		return fmt.Errorf("failed to create water_quality_assessments table: %w", err)
	}

	// Create filter_commands table
	filterCommandsTable := `
	CREATE TABLE IF NOT EXISTS filter_commands (
		id SERIAL PRIMARY KEY,
		command VARCHAR(100) NOT NULL,
		mode VARCHAR(50) NOT NULL CHECK (mode IN ('drinking_water', 'household_water')),
		timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
		status VARCHAR(50) DEFAULT 'sent' CHECK (status IN ('sent', 'acknowledged', 'failed')),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	if _, err := db.Exec(filterCommandsTable); err != nil {
		return fmt.Errorf("failed to create filter_commands table: %w", err)
	}

	// Create command_responses table
	commandResponsesTable := `
	CREATE TABLE IF NOT EXISTS command_responses (
		id SERIAL PRIMARY KEY,
		filter_command_id INTEGER REFERENCES filter_commands(id) ON DELETE SET NULL,
		command VARCHAR(100) NOT NULL,
		status VARCHAR(50) NOT NULL CHECK (status IN ('success', 'error', 'processing')),
		message TEXT,
		timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	if _, err := db.Exec(commandResponsesTable); err != nil {
		return fmt.Errorf("failed to create command_responses table: %w", err)
	}

	// Create device_status table
	deviceStatusTable := `
	CREATE TABLE IF NOT EXISTS device_status (
		id SERIAL PRIMARY KEY,
		device_id VARCHAR(100) UNIQUE NOT NULL,
		last_seen TIMESTAMP WITH TIME ZONE NOT NULL,
		is_active BOOLEAN DEFAULT true,
		current_filter_mode VARCHAR(50) DEFAULT 'drinking_water',
		total_readings INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	if _, err := db.Exec(deviceStatusTable); err != nil {
		return fmt.Errorf("failed to create device_status table: %w", err)
	}

	// Create indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_sensor_readings_timestamp ON sensor_readings(timestamp DESC);",
		"CREATE INDEX IF NOT EXISTS idx_sensor_readings_device_id ON sensor_readings(device_id);",
		"CREATE INDEX IF NOT EXISTS idx_sensor_readings_filter_mode ON sensor_readings(filter_mode);",
		"CREATE INDEX IF NOT EXISTS idx_water_quality_timestamp ON water_quality_assessments(timestamp DESC);",
		"CREATE INDEX IF NOT EXISTS idx_water_quality_device_id ON water_quality_assessments(device_id);",
		"CREATE INDEX IF NOT EXISTS idx_filter_commands_timestamp ON filter_commands(timestamp DESC);",
		"CREATE INDEX IF NOT EXISTS idx_command_responses_timestamp ON command_responses(timestamp DESC);",
		"CREATE INDEX IF NOT EXISTS idx_device_status_last_seen ON device_status(last_seen DESC);",
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
		"command_responses",
		"filter_commands",
		"water_quality_assessments",
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
		"water_quality_assessments",
		"filter_commands",
		"command_responses",
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