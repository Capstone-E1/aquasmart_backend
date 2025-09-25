package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Capstone-E1/aquasmart_backend/config"
	"github.com/Capstone-E1/aquasmart_backend/internal/database"
)

func main() {
	var (
		table = flag.String("table", "sensor_readings", "Table to view (sensor_readings, water_quality_assessments, device_status, filter_commands)")
		limit = flag.Int("limit", 10, "Number of records to show")
	)
	flag.Parse()

	log.Println("🔍 AquaSmart Database Viewer")
	log.Println("============================")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Printf("✅ Connected to database: %s@%s:%s/%s",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	switch *table {
	case "sensor_readings":
		viewSensorReadings(db, *limit)
	case "water_quality_assessments":
		viewWaterQuality(db, *limit)
	case "device_status":
		viewDeviceStatus(db, *limit)
	case "filter_commands":
		viewFilterCommands(db, *limit)
	default:
		log.Printf("Unknown table: %s", *table)
		log.Println("Available tables: sensor_readings, water_quality_assessments, device_status, filter_commands")
	}
}

func viewSensorReadings(db *database.DB, limit int) {
	query := `
		SELECT id, device_id, timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		ORDER BY timestamp DESC
		LIMIT $1`

	rows, err := db.Query(query, limit)
	if err != nil {
		log.Fatalf("❌ Query failed: %v", err)
	}
	defer rows.Close()

	fmt.Printf("\n📊 Latest %d Sensor Readings:\n", limit)
	fmt.Println("=====================================")
	fmt.Printf("%-4s %-12s %-20s %-15s %-6s %-5s %-9s %-6s\n",
		"ID", "Device", "Timestamp", "Filter Mode", "Flow", "pH", "Turbidity", "TDS")
	fmt.Println("-----------------------------------------------------------------------------------------------------")

	count := 0
	for rows.Next() {
		var id int
		var deviceID, filterMode string
		var timestamp string
		var flow, ph, turbidity, tds float64

		err := rows.Scan(&id, &deviceID, &timestamp, &filterMode, &flow, &ph, &turbidity, &tds)
		if err != nil {
			log.Printf("❌ Error scanning row: %v", err)
			continue
		}

		fmt.Printf("%-4d %-12s %-20s %-15s %-6.1f %-5.1f %-9.1f %-6.0f\n",
			id, deviceID, timestamp[:19], filterMode, flow, ph, turbidity, tds)
		count++
	}

	if count == 0 {
		fmt.Println("No sensor readings found.")
	} else {
		fmt.Printf("\nTotal: %d readings\n", count)
	}
}

func viewWaterQuality(db *database.DB, limit int) {
	query := `
		SELECT id, device_id, timestamp, ph_status, turbidity_status, tds_status, overall_quality
		FROM water_quality_assessments
		ORDER BY timestamp DESC
		LIMIT $1`

	rows, err := db.Query(query, limit)
	if err != nil {
		log.Fatalf("❌ Query failed: %v", err)
	}
	defer rows.Close()

	fmt.Printf("\n💧 Latest %d Water Quality Assessments:\n", limit)
	fmt.Println("================================================")
	fmt.Printf("%-4s %-12s %-20s %-12s %-15s %-12s %-15s\n",
		"ID", "Device", "Timestamp", "pH Status", "Turbidity", "TDS Status", "Overall")
	fmt.Println("--------------------------------------------------------------------------------------------------")

	count := 0
	for rows.Next() {
		var id int
		var deviceID, timestamp, phStatus, turbidityStatus, tdsStatus, overall string

		err := rows.Scan(&id, &deviceID, &timestamp, &phStatus, &turbidityStatus, &tdsStatus, &overall)
		if err != nil {
			log.Printf("❌ Error scanning row: %v", err)
			continue
		}

		fmt.Printf("%-4d %-12s %-20s %-12s %-15s %-12s %-15s\n",
			id, deviceID, timestamp[:19], phStatus, turbidityStatus, tdsStatus, overall)
		count++
	}

	if count == 0 {
		fmt.Println("No water quality assessments found.")
	} else {
		fmt.Printf("\nTotal: %d assessments\n", count)
	}
}

func viewDeviceStatus(db *database.DB, limit int) {
	query := `
		SELECT device_id, last_seen, is_active, current_filter_mode, total_readings
		FROM device_status
		ORDER BY last_seen DESC
		LIMIT $1`

	rows, err := db.Query(query, limit)
	if err != nil {
		log.Fatalf("❌ Query failed: %v", err)
	}
	defer rows.Close()

	fmt.Printf("\n🔌 Device Status:\n")
	fmt.Println("==================")
	fmt.Printf("%-12s %-20s %-8s %-15s %-8s\n",
		"Device ID", "Last Seen", "Active", "Filter Mode", "Readings")
	fmt.Println("---------------------------------------------------------------")

	count := 0
	for rows.Next() {
		var deviceID, lastSeen, filterMode string
		var isActive bool
		var totalReadings int

		err := rows.Scan(&deviceID, &lastSeen, &isActive, &filterMode, &totalReadings)
		if err != nil {
			log.Printf("❌ Error scanning row: %v", err)
			continue
		}

		activeStr := "Yes"
		if !isActive {
			activeStr = "No"
		}

		fmt.Printf("%-12s %-20s %-8s %-15s %-8d\n",
			deviceID, lastSeen[:19], activeStr, filterMode, totalReadings)
		count++
	}

	if count == 0 {
		fmt.Println("No devices found.")
	} else {
		fmt.Printf("\nTotal: %d devices\n", count)
	}
}

func viewFilterCommands(db *database.DB, limit int) {
	query := `
		SELECT id, command, mode, timestamp, status
		FROM filter_commands
		ORDER BY timestamp DESC
		LIMIT $1`

	rows, err := db.Query(query, limit)
	if err != nil {
		log.Fatalf("❌ Query failed: %v", err)
	}
	defer rows.Close()

	fmt.Printf("\n🔧 Latest %d Filter Commands:\n", limit)
	fmt.Println("===============================")
	fmt.Printf("%-4s %-20s %-18s %-20s %-10s\n",
		"ID", "Command", "Mode", "Timestamp", "Status")
	fmt.Println("------------------------------------------------------------------------")

	count := 0
	for rows.Next() {
		var id int
		var command, mode, timestamp, status string

		err := rows.Scan(&id, &command, &mode, &timestamp, &status)
		if err != nil {
			log.Printf("❌ Error scanning row: %v", err)
			continue
		}

		fmt.Printf("%-4d %-20s %-18s %-20s %-10s\n",
			id, command, mode, timestamp[:19], status)
		count++
	}

	if count == 0 {
		fmt.Println("No filter commands found.")
	} else {
		fmt.Printf("\nTotal: %d commands\n", count)
	}
}