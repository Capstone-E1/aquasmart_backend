package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️  Warning: .env file not found")
	}

	// Build connection string
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	sslMode := os.Getenv("DB_SSLMODE")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, sslMode)

	log.Println("🔄 Connecting to database...")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}

	log.Println("✅ Connected to database")
	
	// First, drop all views (views must be dropped before tables)
	log.Println("🗑️  Dropping all views...")
	views := []string{
		"current_water_quality",
		"latest_sensor_readings",
		"latest_readings_by_mode",
	}
	
	for _, view := range views {
		query := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE", view)
		_, err := db.Exec(query)
		if err != nil {
			log.Printf("⚠️  Warning dropping view %s: %v", view, err)
		} else {
			log.Printf("✅ Dropped view: %s", view)
		}
	}
	
	log.Println("🗑️  Dropping all tables...")

	// Drop all tables in reverse dependency order
	tables := []string{
		"schedule_executions",
		"filter_schedules",
		"water_quality_assessments",
		"filtration_process",
		"sensor_readings",
		"device_status",
		"devices",
		"schema_migrations",
	}

	for _, table := range tables {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)
		_, err := db.Exec(query)
		if err != nil {
			log.Printf("⚠️  Warning dropping %s: %v", table, err)
		} else {
			log.Printf("✅ Dropped table: %s", table)
		}
	}

	log.Println("")
	log.Println("✅ Database reset complete!")
	log.Println("🚀 Now run: go build -o server ./cmd/server && ./server")
	log.Println("   All migrations will be applied automatically on startup")
}
