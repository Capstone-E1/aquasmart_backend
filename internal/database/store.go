package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

// DatabaseStore implements persistent storage using PostgreSQL
type DatabaseStore struct {
	db *sql.DB
}

// NewDatabaseStore creates a new database store
func NewDatabaseStore(db *sql.DB) *DatabaseStore {
	return &DatabaseStore{db: db}
}

// AddSensorReading stores a sensor reading in the database
func (s *DatabaseStore) AddSensorReading(reading models.SensorReading) {
	query := `
		INSERT INTO sensor_readings (device_id, timestamp, filter_mode, flow, ph, turbidity, tds)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (device_id, timestamp) DO UPDATE SET
			filter_mode = EXCLUDED.filter_mode,
			flow = EXCLUDED.flow,
			ph = EXCLUDED.ph,
			turbidity = EXCLUDED.turbidity,
			tds = EXCLUDED.tds`

	_, err := s.db.Exec(query, reading.DeviceID, reading.Timestamp, reading.FilterMode,
		reading.Flow, reading.Ph, reading.Turbidity, reading.TDS)
	if err != nil {
		log.Printf("❌ Error storing sensor reading: %v", err)
		return
	}

	// Update device status (last_seen, total_readings)
	s.updateDeviceStatus(reading.DeviceID)
}

// updateDeviceStatus updates the device status when new data arrives
func (s *DatabaseStore) updateDeviceStatus(deviceID string) {
	query := `
		INSERT INTO device_status (device_id, last_seen, total_readings, updated_at)
		VALUES ($1, NOW(), 1, NOW())
		ON CONFLICT (device_id) DO UPDATE SET
			last_seen = NOW(),
			total_readings = device_status.total_readings + 1,
			updated_at = NOW()`

	_, err := s.db.Exec(query, deviceID)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to update device status: %v", err)
	}
}

// GetLatestReading returns the most recent sensor reading
func (s *DatabaseStore) GetLatestReading() (*models.SensorReading, bool) {
	query := `
		SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		ORDER BY timestamp DESC
		LIMIT 1`

	var reading models.SensorReading
	err := s.db.QueryRow(query).Scan(
		&reading.DeviceID, &reading.Timestamp, &reading.FilterMode, &reading.Flow,
		&reading.Ph, &reading.Turbidity, &reading.TDS)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("❌ Error getting latest reading: %v", err)
		return nil, false
	}

	return &reading, true
}

// GetLatestReadingByMode returns the most recent reading for a specific filter mode
func (s *DatabaseStore) GetLatestReadingByMode(mode models.FilterMode) (*models.SensorReading, bool) {
	query := `
		SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		WHERE filter_mode = $1
		ORDER BY timestamp DESC
		LIMIT 1`

	var reading models.SensorReading
	err := s.db.QueryRow(query, string(mode)).Scan(
		&reading.DeviceID, &reading.Timestamp, &reading.FilterMode, &reading.Flow,
		&reading.Ph, &reading.Turbidity, &reading.TDS)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("❌ Error getting latest reading by mode: %v", err)
		return nil, false
	}

	return &reading, true
}

// GetLatestReadingByDevice returns the most recent reading for a specific device
func (s *DatabaseStore) GetLatestReadingByDevice(deviceID string) (*models.SensorReading, bool) {
	query := `
		SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		WHERE device_id = $1
		ORDER BY timestamp DESC
		LIMIT 1`

	var reading models.SensorReading
	err := s.db.QueryRow(query, deviceID).Scan(
		&reading.DeviceID, &reading.Timestamp, &reading.FilterMode, &reading.Flow,
		&reading.Ph, &reading.Turbidity, &reading.TDS)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("❌ Error getting latest reading by device: %v", err)
		return nil, false
	}

	return &reading, true
}

// GetAllLatestReadingsByDevice returns the latest reading for each device
func (s *DatabaseStore) GetAllLatestReadingsByDevice() map[string]models.SensorReading {
	query := `
		SELECT DISTINCT ON (device_id)
			device_id, timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		ORDER BY device_id, timestamp DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("❌ Error getting all latest readings by device: %v", err)
		return map[string]models.SensorReading{}
	}
	defer rows.Close()

	result := make(map[string]models.SensorReading)
	for rows.Next() {
		var reading models.SensorReading
		err := rows.Scan(
			&reading.DeviceID, &reading.Timestamp, &reading.FilterMode, &reading.Flow,
			&reading.Ph, &reading.Turbidity, &reading.TDS)
		if err != nil {
			log.Printf("⚠️  Warning: Error scanning reading: %v", err)
			continue
		}
		result[reading.DeviceID] = reading
	}

	return result
}

// GetAllLatestReadings returns the latest readings for each filter mode
func (s *DatabaseStore) GetAllLatestReadings() []models.SensorReading {
	readings := []models.SensorReading{}

	// Get latest for drinking_water
	if reading, exists := s.GetLatestReadingByMode(models.FilterModeDrinking); exists {
		readings = append(readings, *reading)
	}

	// Get latest for household_water
	if reading, exists := s.GetLatestReadingByMode(models.FilterModeHousehold); exists {
		readings = append(readings, *reading)
	}

	return readings
}

// GetRecentReadings returns the N most recent sensor readings
func (s *DatabaseStore) GetRecentReadings(limit int) []models.SensorReading {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		ORDER BY timestamp DESC
		LIMIT $1`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		log.Printf("❌ Error getting recent readings: %v", err)
		return []models.SensorReading{}
	}
	defer rows.Close()

	var readings []models.SensorReading
	for rows.Next() {
		var reading models.SensorReading
		err := rows.Scan(
			&reading.DeviceID, &reading.Timestamp, &reading.FilterMode, &reading.Flow,
			&reading.Ph, &reading.Turbidity, &reading.TDS)
		if err != nil {
			log.Printf("⚠️  Warning: Error scanning reading: %v", err)
			continue
		}
		readings = append(readings, reading)
	}

	return readings
}

// GetRecentReadingsByMode returns recent readings for a specific filter mode
func (s *DatabaseStore) GetRecentReadingsByMode(mode models.FilterMode, limit int) []models.SensorReading {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		WHERE filter_mode = $1
		ORDER BY timestamp DESC
		LIMIT $2`

	rows, err := s.db.Query(query, string(mode), limit)
	if err != nil {
		log.Printf("❌ Error getting recent readings by mode: %v", err)
		return []models.SensorReading{}
	}
	defer rows.Close()

	var readings []models.SensorReading
	for rows.Next() {
		var reading models.SensorReading
		err := rows.Scan(
			&reading.DeviceID, &reading.Timestamp, &reading.FilterMode, &reading.Flow,
			&reading.Ph, &reading.Turbidity, &reading.TDS)
		if err != nil {
			log.Printf("⚠️  Warning: Error scanning reading: %v", err)
			continue
		}
		readings = append(readings, reading)
	}

	return readings
}

// GetRecentReadingsByDevice returns recent readings for a specific device
func (s *DatabaseStore) GetRecentReadingsByDevice(deviceID string, limit int) []models.SensorReading {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		WHERE device_id = $1
		ORDER BY timestamp DESC
		LIMIT $2`

	rows, err := s.db.Query(query, deviceID, limit)
	if err != nil {
		log.Printf("❌ Error getting recent readings by device: %v", err)
		return []models.SensorReading{}
	}
	defer rows.Close()

	var readings []models.SensorReading
	for rows.Next() {
		var reading models.SensorReading
		err := rows.Scan(
			&reading.DeviceID, &reading.Timestamp, &reading.FilterMode, &reading.Flow,
			&reading.Ph, &reading.Turbidity, &reading.TDS)
		if err != nil {
			log.Printf("⚠️  Warning: Error scanning reading: %v", err)
			continue
		}
		readings = append(readings, reading)
	}

	return readings
}

// GetReadingsByDevice returns all readings for a specific device
func (s *DatabaseStore) GetReadingsByDevice(deviceID string) []models.SensorReading {
	query := `
		SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		WHERE device_id = $1
		ORDER BY timestamp DESC`

	rows, err := s.db.Query(query, deviceID)
	if err != nil {
		log.Printf("❌ Error getting readings by device: %v", err)
		return []models.SensorReading{}
	}
	defer rows.Close()

	var readings []models.SensorReading
	for rows.Next() {
		var reading models.SensorReading
		err := rows.Scan(
			&reading.DeviceID, &reading.Timestamp, &reading.FilterMode, &reading.Flow,
			&reading.Ph, &reading.Turbidity, &reading.TDS)
		if err != nil {
			log.Printf("⚠️  Warning: Error scanning reading: %v", err)
			continue
		}
		readings = append(readings, reading)
	}

	return readings
}

// GetReadingsInRange returns all readings within a time range
func (s *DatabaseStore) GetReadingsInRange(start, end time.Time) []models.SensorReading {
	query := `
		SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		WHERE timestamp BETWEEN $1 AND $2
		ORDER BY timestamp DESC`

	rows, err := s.db.Query(query, start, end)
	if err != nil {
		log.Printf("❌ Error getting readings in range: %v", err)
		return []models.SensorReading{}
	}
	defer rows.Close()

	var readings []models.SensorReading
	for rows.Next() {
		var reading models.SensorReading
		err := rows.Scan(
			&reading.DeviceID, &reading.Timestamp, &reading.FilterMode, &reading.Flow,
			&reading.Ph, &reading.Turbidity, &reading.TDS)
		if err != nil {
			log.Printf("⚠️  Warning: Error scanning reading: %v", err)
			continue
		}
		readings = append(readings, reading)
	}

	return readings
}

// GetHistoricalReadings returns readings in a time range with optional filter mode
func (s *DatabaseStore) GetHistoricalReadings(start, end time.Time, filterMode *models.FilterMode) ([]models.SensorReading, error) {
	var query string
	var args []interface{}

	if filterMode != nil {
		query = `
			SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
			FROM sensor_readings
			WHERE timestamp BETWEEN $1 AND $2 AND filter_mode = $3
			ORDER BY timestamp DESC`
		args = []interface{}{start, end, string(*filterMode)}
	} else {
		query = `
			SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
			FROM sensor_readings
			WHERE timestamp BETWEEN $1 AND $2
			ORDER BY timestamp DESC`
		args = []interface{}{start, end}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical readings: %w", err)
	}
	defer rows.Close()

	var readings []models.SensorReading
	for rows.Next() {
		var reading models.SensorReading
		err := rows.Scan(
			&reading.DeviceID, &reading.Timestamp, &reading.FilterMode, &reading.Flow,
			&reading.Ph, &reading.Turbidity, &reading.TDS)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reading: %w", err)
		}
		readings = append(readings, reading)
	}

	return readings, nil
}

// GetRecentReadingsWithFilter returns recent readings with optional filter mode
func (s *DatabaseStore) GetRecentReadingsWithFilter(limit int, filterMode *models.FilterMode) ([]models.SensorReading, error) {
	if limit <= 0 {
		limit = 50
	}

	var query string
	var args []interface{}

	if filterMode != nil {
		query = `
			SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
			FROM sensor_readings
			WHERE filter_mode = $1
			ORDER BY timestamp DESC
			LIMIT $2`
		args = []interface{}{string(*filterMode), limit}
	} else {
		query = `
			SELECT device_id, timestamp, filter_mode, flow, ph, turbidity, tds
			FROM sensor_readings
			ORDER BY timestamp DESC
			LIMIT $1`
		args = []interface{}{limit}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent readings: %w", err)
	}
	defer rows.Close()

	var readings []models.SensorReading
	for rows.Next() {
		var reading models.SensorReading
		err := rows.Scan(
			&reading.DeviceID, &reading.Timestamp, &reading.FilterMode, &reading.Flow,
			&reading.Ph, &reading.Turbidity, &reading.TDS)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reading: %w", err)
		}
		readings = append(readings, reading)
	}

	return readings, nil
}

// GetReadingCount returns the total number of readings stored
func (s *DatabaseStore) GetReadingCount() int {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sensor_readings").Scan(&count)
	if err != nil {
		log.Printf("❌ Error getting reading count: %v", err)
		return 0
	}
	return count
}

// GetCurrentFilterMode returns the current filter mode setting
func (s *DatabaseStore) GetCurrentFilterMode() models.FilterMode {
	var filterMode string
	// Try to get from any device in the system (prioritize most recent update)
	query := `SELECT current_filter_mode FROM device_status ORDER BY updated_at DESC LIMIT 1`
	
	err := s.db.QueryRow(query).Scan(&filterMode)
	if err != nil {
		log.Printf("⚠️  Failed to get current filter mode from database: %v, using default", err)
		return models.FilterModeDrinking // Default fallback
	}
	
	return models.FilterMode(filterMode)
}

// SetCurrentFilterMode sets the current filter mode for ALL devices
func (s *DatabaseStore) SetCurrentFilterMode(mode models.FilterMode) {
	log.Printf("Setting filter mode to: %s for all devices", mode)
	
	// Update filter mode for ALL devices in the system
	query := `
		UPDATE device_status 
		SET current_filter_mode = $1, updated_at = NOW()
		WHERE device_id IN ('stm32_main', 'stm32_pre', 'stm32_post')
	`
	
	result, err := s.db.Exec(query, string(mode))
	if err != nil {
		log.Printf("❌ Failed to set filter mode in database: %v", err)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	log.Printf("✅ Filter mode changed to %s for %d devices", mode, rowsAffected)
}

// GetWaterQualityStatus returns water quality assessment for latest reading
func (s *DatabaseStore) GetWaterQualityStatus() (*models.WaterQualityStatus, bool) {
	reading, exists := s.GetLatestReading()
	if !exists {
		return nil, false
	}
	
	status := reading.ToWaterQualityStatus()
	return &status, true
}

// GetWaterQualityStatusByMode returns water quality assessment for a specific filter mode
func (s *DatabaseStore) GetWaterQualityStatusByMode(mode models.FilterMode) (*models.WaterQualityStatus, bool) {
	reading, exists := s.GetLatestReadingByMode(mode)
	if !exists {
		return nil, false
	}
	
	status := reading.ToWaterQualityStatus()
	return &status, true
}

// GetAllWaterQualityStatus returns water quality assessment for all filter modes
func (s *DatabaseStore) GetAllWaterQualityStatus() []models.WaterQualityStatus {
	readings := s.GetAllLatestReadings()
	statuses := make([]models.WaterQualityStatus, 0, len(readings))
	
	for _, reading := range readings {
		status := reading.ToWaterQualityStatus()
		statuses = append(statuses, status)
	}
	
	return statuses
}

// Placeholder methods for filtration process (not used with HTTP-only communication)
func (s *DatabaseStore) GetFiltrationProcess() (*models.FiltrationProcess, bool) {
	return nil, false
}

func (s *DatabaseStore) SetFiltrationProcess(process *models.FiltrationProcess) {}

func (s *DatabaseStore) UpdateFiltrationProgress(currentFlowRate float64) {}

func (s *DatabaseStore) StartFiltrationProcess(mode models.FilterMode, targetVolume float64) {}

func (s *DatabaseStore) CompleteFiltrationProcess() {}

func (s *DatabaseStore) ClearFiltrationProcess() {}

func (s *DatabaseStore) ClearCompletedProcess() {}

func (s *DatabaseStore) CanChangeFilterMode() (bool, string) {
	return true, ""
}

func (s *DatabaseStore) GetActiveDevices() []string {
	query := `SELECT device_id FROM device_status WHERE is_active = true`
	
	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("❌ Error getting active devices: %v", err)
		return []string{}
	}
	defer rows.Close()
	
	var devices []string
	for rows.Next() {
		var deviceID string
		if err := rows.Scan(&deviceID); err != nil {
			continue
		}
		devices = append(devices, deviceID)
	}
	
	return devices
}

// SetLEDCommand sets the LED command (stored in memory, not in database for simplicity)
// For production, you might want to store this in a commands table
var ledCommand string = "OFF"

func (s *DatabaseStore) SetLEDCommand(command string) {
	ledCommand = command
}

// GetLEDCommand retrieves the current LED command
func (s *DatabaseStore) GetLEDCommand() string {
	return ledCommand
}
