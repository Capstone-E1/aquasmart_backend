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

// AddSensorReading stores a sensor reading in the database (matches in-memory store interface)
func (s *DatabaseStore) AddSensorReading(reading models.SensorReading) {
	err := s.StoreSensorReading(reading)
	if err != nil {
		log.Printf("Error storing sensor reading: %v", err)
	}
}

// StoreSensorReading stores a sensor reading in the database
func (s *DatabaseStore) StoreSensorReading(reading models.SensorReading) error {
	query := `
		INSERT INTO sensor_readings (device_id, timestamp, filter_mode, flow, ph, turbidity, tds)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (device_id, timestamp) DO UPDATE SET
			filter_mode = EXCLUDED.filter_mode,
			flow = EXCLUDED.flow,
			ph = EXCLUDED.ph,
			turbidity = EXCLUDED.turbidity,
			tds = EXCLUDED.tds`

	_, err := s.db.Exec(query, "default", reading.Timestamp, reading.FilterMode,
		reading.Flow, reading.Ph, reading.Turbidity, reading.TDS)
	if err != nil {
		return fmt.Errorf("failed to store sensor reading: %w", err)
	}

	// Update device status
	s.updateDeviceStatus("default", reading.FilterMode)

	// Also store water quality assessment
	waterQuality := reading.ToWaterQualityStatus()
	if err := s.StoreWaterQualityStatus(waterQuality); err != nil {
		log.Printf("Warning: Failed to store water quality status: %v", err)
	}

	return nil
}

// StoreWaterQualityStatus stores water quality assessment in the database
func (s *DatabaseStore) StoreWaterQualityStatus(status models.WaterQualityStatus) error {
	query := `
		INSERT INTO water_quality_assessments
		(device_id, timestamp, filter_mode, flow, ph, ph_status, turbidity, turbidity_status,
		 tds, tds_status, overall_quality)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := s.db.Exec(query, "default", status.Timestamp, status.FilterMode,
		status.Flow, status.Ph, status.PhStatus, status.Turbidity, status.TurbStatus,
		status.TDS, status.TDSStatus, status.OverallQuality)
	if err != nil {
		return fmt.Errorf("failed to store water quality status: %w", err)
	}

	return nil
}

// GetLatestReading returns the most recent sensor reading (matches in-memory store interface)
func (s *DatabaseStore) GetLatestReading() (*models.SensorReading, bool) {
	reading, err := s.getLatestReading()
	if err != nil || reading == nil {
		return nil, false
	}
	return reading, true
}

// getLatestReading returns the most recent sensor reading
func (s *DatabaseStore) getLatestReading() (*models.SensorReading, error) {
	query := `
		SELECT timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		ORDER BY timestamp DESC
		LIMIT 1`

	var reading models.SensorReading
	err := s.db.QueryRow(query).Scan(
		&reading.Timestamp, &reading.FilterMode, &reading.Flow,
		&reading.Ph, &reading.Turbidity, &reading.TDS)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest reading: %w", err)
	}

	return &reading, nil
}

// GetLatestReadingByMode returns the most recent reading for a specific filter mode (matches in-memory store interface)
func (s *DatabaseStore) GetLatestReadingByMode(mode models.FilterMode) (*models.SensorReading, bool) {
	reading, err := s.getLatestReadingByMode(mode)
	if err != nil || reading == nil {
		return nil, false
	}
	return reading, true
}

// getLatestReadingByMode returns the most recent reading for a specific filter mode
func (s *DatabaseStore) getLatestReadingByMode(mode models.FilterMode) (*models.SensorReading, error) {
	query := `
		SELECT timestamp, filter_mode, flow, ph, turbidity, tds
		FROM sensor_readings
		WHERE filter_mode = $1
		ORDER BY timestamp DESC
		LIMIT 1`

	var reading models.SensorReading
	err := s.db.QueryRow(query, mode).Scan(
		&reading.Timestamp, &reading.FilterMode, &reading.Flow,
		&reading.Ph, &reading.Turbidity, &reading.TDS)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest reading by mode: %w", err)
	}

	return &reading, nil
}

// GetRecentReadingsWithFilter returns recent sensor readings with optional filtering
func (s *DatabaseStore) GetRecentReadingsWithFilter(limit int, filterMode *models.FilterMode) ([]models.SensorReading, error) {
	return s.getRecentReadings(limit, filterMode)
}

// GetHistoricalReadings returns readings within a time range
func (s *DatabaseStore) GetHistoricalReadings(start, end time.Time, filterMode *models.FilterMode) ([]models.SensorReading, error) {
	var query string
	var args []interface{}

	if filterMode != nil {
		query = `
			SELECT timestamp, filter_mode, flow, ph, turbidity, tds
			FROM sensor_readings
			WHERE timestamp BETWEEN $1 AND $2 AND filter_mode = $3
			ORDER BY timestamp DESC`
		args = []interface{}{start, end, *filterMode}
	} else {
		query = `
			SELECT timestamp, filter_mode, flow, ph, turbidity, tds
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
		err := rows.Scan(&reading.Timestamp, &reading.FilterMode, &reading.Flow,
			&reading.Ph, &reading.Turbidity, &reading.TDS)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reading: %w", err)
		}
		readings = append(readings, reading)
	}

	return readings, nil
}

// GetReadingCount returns the total number of stored readings (matches in-memory store interface)
func (s *DatabaseStore) GetReadingCount() int {
	count, err := s.GetTotalReadingCount()
	if err != nil {
		log.Printf("Error getting reading count: %v", err)
		return 0
	}
	return count
}

// GetTotalReadingCount returns the total number of stored readings
func (s *DatabaseStore) GetTotalReadingCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sensor_readings").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get reading count: %w", err)
	}
	return count, nil
}

// GetCurrentFilterMode returns the current filter mode (placeholder - implement based on your logic)
func (s *DatabaseStore) GetCurrentFilterMode() models.FilterMode {
	// This could be stored in device_status table or retrieved from latest reading
	return models.FilterModeDrinking // Default fallback
}

// SetCurrentFilterMode sets the current filter mode (placeholder - implement based on your logic)
func (s *DatabaseStore) SetCurrentFilterMode(mode models.FilterMode) {
	// Implement logic to store current filter mode
	log.Printf("Setting filter mode to: %s", mode)
}

// GetWaterQualityStatus returns the latest water quality status
func (s *DatabaseStore) GetWaterQualityStatus() (*models.WaterQualityStatus, bool) {
	reading, exists := s.GetLatestReading()
	if !exists {
		return nil, false
	}
	status := reading.ToWaterQualityStatus()
	return &status, true
}

// GetWaterQualityStatusByMode returns water quality status for a specific mode
func (s *DatabaseStore) GetWaterQualityStatusByMode(mode models.FilterMode) (*models.WaterQualityStatus, bool) {
	reading, exists := s.GetLatestReadingByMode(mode)
	if !exists {
		return nil, false
	}
	status := reading.ToWaterQualityStatus()
	return &status, true
}

// GetRecentReadings returns recent readings (matches in-memory store interface)
func (s *DatabaseStore) GetRecentReadings(limit int) []models.SensorReading {
	readings, err := s.getRecentReadings(limit, nil)
	if err != nil {
		log.Printf("Error getting recent readings: %v", err)
		return []models.SensorReading{}
	}
	return readings
}

// getRecentReadings is the internal method with error handling
func (s *DatabaseStore) getRecentReadings(limit int, filterMode *models.FilterMode) ([]models.SensorReading, error) {
	var query string
	var args []interface{}

	if filterMode != nil {
		query = `
			SELECT timestamp, filter_mode, flow, ph, turbidity, tds
			FROM sensor_readings
			WHERE filter_mode = $1
			ORDER BY timestamp DESC
			LIMIT $2`
		args = []interface{}{*filterMode, limit}
	} else {
		query = `
			SELECT timestamp, filter_mode, flow, ph, turbidity, tds
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
		err := rows.Scan(&reading.Timestamp, &reading.FilterMode, &reading.Flow,
			&reading.Ph, &reading.Turbidity, &reading.TDS)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reading: %w", err)
		}
		readings = append(readings, reading)
	}

	return readings, nil
}

// GetAllLatestReadings returns latest readings from all filter modes
func (s *DatabaseStore) GetAllLatestReadings() []models.SensorReading {
	var readings []models.SensorReading

	// Get latest for each filter mode
	for _, mode := range []models.FilterMode{models.FilterModeDrinking, models.FilterModeHousehold} {
		if reading, exists := s.GetLatestReadingByMode(mode); exists {
			readings = append(readings, *reading)
		}
	}

	return readings
}

// GetRecentReadingsByMode returns recent readings for a specific filter mode
func (s *DatabaseStore) GetRecentReadingsByMode(mode models.FilterMode, limit int) []models.SensorReading {
	readings, err := s.getRecentReadings(limit, &mode)
	if err != nil {
		log.Printf("Error getting recent readings by mode: %v", err)
		return []models.SensorReading{}
	}
	return readings
}

// GetReadingsInRange returns readings within a time range
func (s *DatabaseStore) GetReadingsInRange(start, end time.Time) []models.SensorReading {
	readings, err := s.GetHistoricalReadings(start, end, nil)
	if err != nil {
		log.Printf("Error getting readings in range: %v", err)
		return []models.SensorReading{}
	}
	return readings
}

// GetAllWaterQualityStatus returns water quality status for all modes
func (s *DatabaseStore) GetAllWaterQualityStatus() []models.WaterQualityStatus {
	var statuses []models.WaterQualityStatus

	// Get status for each filter mode
	for _, mode := range []models.FilterMode{models.FilterModeDrinking, models.FilterModeHousehold} {
		if status, exists := s.GetWaterQualityStatusByMode(mode); exists {
			statuses = append(statuses, *status)
		}
	}

	return statuses
}

// GetActiveDevices returns a list of active device IDs (matches in-memory store interface)
func (s *DatabaseStore) GetActiveDevices() []string {
	devices, err := s.getActiveDevices()
	if err != nil {
		log.Printf("Error getting active devices: %v", err)
		return []string{}
	}
	return devices
}

// getActiveDevices returns a list of active device IDs
func (s *DatabaseStore) getActiveDevices() ([]string, error) {
	query := `
		SELECT DISTINCT device_id
		FROM device_status
		WHERE is_active = true AND last_seen > NOW() - INTERVAL '1 hour'
		ORDER BY device_id`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active devices: %w", err)
	}
	defer rows.Close()

	var devices []string
	for rows.Next() {
		var deviceID string
		if err := rows.Scan(&deviceID); err != nil {
			return nil, fmt.Errorf("failed to scan device ID: %w", err)
		}
		devices = append(devices, deviceID)
	}

	return devices, nil
}

// StoreFilterCommand stores a filter command in the database
func (s *DatabaseStore) StoreFilterCommand(cmd models.FilterCommand) error {
	query := `
		INSERT INTO filter_commands (command, mode, timestamp)
		VALUES ($1, $2, $3)`

	_, err := s.db.Exec(query, cmd.Command, cmd.Mode, cmd.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to store filter command: %w", err)
	}

	return nil
}

// updateDeviceStatus updates or creates device status record
func (s *DatabaseStore) updateDeviceStatus(deviceID string, filterMode models.FilterMode) {
	query := `
		INSERT INTO device_status (device_id, last_seen, current_filter_mode, total_readings, updated_at)
		VALUES ($1, NOW(), $2, 1, NOW())
		ON CONFLICT (device_id) DO UPDATE SET
			last_seen = NOW(),
			current_filter_mode = EXCLUDED.current_filter_mode,
			total_readings = device_status.total_readings + 1,
			updated_at = NOW()`

	_, err := s.db.Exec(query, deviceID, filterMode)
	if err != nil {
		log.Printf("Warning: Failed to update device status: %v", err)
	}
}

// === Filtration Process Methods (Database Implementation) ===

// Note: For simplicity, these methods fall back to in-memory storage
// In production, you might want to store filtration processes in the database

var inMemoryProcess *models.FiltrationProcess

// GetFiltrationProcess returns the current filtration process state
func (s *DatabaseStore) GetFiltrationProcess() (*models.FiltrationProcess, bool) {
	if inMemoryProcess == nil {
		return nil, false
	}

	// Return a copy to avoid race conditions
	processCopy := *inMemoryProcess
	return &processCopy, true
}

// SetFiltrationProcess sets the current filtration process state
func (s *DatabaseStore) SetFiltrationProcess(process *models.FiltrationProcess) {
	if process == nil {
		inMemoryProcess = nil
		return
	}

	// Store a copy to avoid external modifications
	processCopy := *process
	inMemoryProcess = &processCopy
}

// UpdateFiltrationProgress updates the current filtration progress with flow rate
func (s *DatabaseStore) UpdateFiltrationProgress(currentFlowRate float64) {
	if inMemoryProcess == nil {
		return
	}

	inMemoryProcess.UpdateProgress(currentFlowRate)
}

// StartFiltrationProcess starts a new filtration process
func (s *DatabaseStore) StartFiltrationProcess(mode models.FilterMode, targetVolume float64) {
	inMemoryProcess = models.NewFiltrationProcess(mode, targetVolume)
	s.SetCurrentFilterMode(mode)
}

// CompleteFiltrationProcess marks the current filtration process as completed
func (s *DatabaseStore) CompleteFiltrationProcess() {
	if inMemoryProcess != nil {
		inMemoryProcess.State = models.FiltrationStateCompleted
		inMemoryProcess.Progress = 100.0
	}
}

// CanChangeFilterMode returns whether filter mode can be changed and the reason if not
func (s *DatabaseStore) CanChangeFilterMode() (bool, string) {
	if inMemoryProcess == nil {
		return true, ""
	}

	return inMemoryProcess.CanChangeMode()
}

// ClearCompletedProcess removes a completed filtration process
func (s *DatabaseStore) ClearCompletedProcess() {
	if inMemoryProcess != nil && inMemoryProcess.State == models.FiltrationStateCompleted {
		inMemoryProcess = nil
	}
}