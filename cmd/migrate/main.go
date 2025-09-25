package main

import (
	"flag"
	"log"
	"os"

	"github.com/Capstone-E1/aquasmart_backend/config"
	"github.com/Capstone-E1/aquasmart_backend/internal/database"
)

func main() {
	var (
		drop   = flag.Bool("drop", false, "Drop all tables before creating")
		create = flag.Bool("create", true, "Create tables")
		check  = flag.Bool("check", false, "Check if tables exist")
	)
	flag.Parse()

	log.Println("🏗️  AquaSmart Database Migration Tool")
	log.Println("=====================================")

	// Load configuration
	cfg := config.Load()

	// Check if database credentials are provided
	if cfg.Database.Host == "localhost" || cfg.Database.Password == "" {
		log.Println("⚠️  Database credentials not configured. Please set environment variables:")
		log.Println("   DB_HOST=your-aiven-host.aivencloud.com")
		log.Println("   DB_PORT=your-port")
		log.Println("   DB_USER=your-username")
		log.Println("   DB_PASSWORD=your-password")
		log.Println("   DB_NAME=your-database-name")
		log.Println("   DB_SSLMODE=require")
		os.Exit(1)
	}

	// Connect to database
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Printf("✅ Connected to database: %s@%s:%s/%s",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// Drop tables if requested
	if *drop {
		log.Println("🗑️  Dropping existing tables...")
		if err := database.DropTables(db.DB); err != nil {
			log.Fatalf("❌ Failed to drop tables: %v", err)
		}
	}

	// Create tables
	if *create {
		log.Println("🏗️  Creating database tables...")
		if err := database.CreateTables(db.DB); err != nil {
			log.Fatalf("❌ Failed to create tables: %v", err)
		}
	}

	// Check tables
	if *check {
		log.Println("🔍 Checking if tables exist...")
		if err := database.CheckTablesExist(db.DB); err != nil {
			log.Fatalf("❌ Table check failed: %v", err)
		}
	}

	log.Println("🎉 Database migration completed successfully!")
}