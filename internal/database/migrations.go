package database

import (
	"database/sql"
	"fmt"
	"log"
)

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