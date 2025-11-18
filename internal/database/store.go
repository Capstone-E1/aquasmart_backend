package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
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

// Ping checks if database connection is alive
func (s *DatabaseStore) Ping() error {
	return s.db.Ping()
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

	// Update device status (last_seen, total_readings) and accumulate flow
	s.updateDeviceStatus(reading.DeviceID)
	s.accumulateFlow(reading.DeviceID, reading.Flow, reading.Timestamp)
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

// accumulateFlow calculates and accumulates flow since last update
func (s *DatabaseStore) accumulateFlow(deviceID string, currentFlowRate float64, timestamp time.Time) {
	// Get last flow update time
	var lastUpdate *time.Time
	var totalFlow float64
	
	query := `SELECT last_flow_update_at, total_flow_liters FROM device_status WHERE device_id = $1`
	err := s.db.QueryRow(query, deviceID).Scan(&lastUpdate, &totalFlow)
	
	if err != nil {
		// First time or error, initialize
		log.Printf("⚠️  Warning: Could not get last flow update for %s: %v", deviceID, err)
		updateQuery := `
			UPDATE device_status 
			SET last_flow_update_at = $1 
			WHERE device_id = $2`
		s.db.Exec(updateQuery, timestamp, deviceID)
		return
	}
	
	// If no previous update or filter_mode_started_at is null, skip calculation
	if lastUpdate == nil {
		updateQuery := `
			UPDATE device_status 
			SET last_flow_update_at = $1 
			WHERE device_id = $2`
		s.db.Exec(updateQuery, timestamp, deviceID)
		return
	}
	
	// Calculate time difference in minutes
	timeDiff := timestamp.Sub(*lastUpdate).Minutes()
	
	// Avoid negative time or too large gaps (max 5 minutes between readings)
	if timeDiff < 0 || timeDiff > 5 {
		log.Printf("⚠️  Unusual time gap for flow calculation: %.2f minutes", timeDiff)
		updateQuery := `
			UPDATE device_status 
			SET last_flow_update_at = $1 
			WHERE device_id = $2`
		s.db.Exec(updateQuery, timestamp, deviceID)
		return
	}
	
	// Calculate flow volume: flow_rate (L/min) * time (min) = volume (L)
	flowVolume := currentFlowRate * timeDiff
	newTotalFlow := totalFlow + flowVolume
	
	// Update total flow
	updateQuery := `
		UPDATE device_status 
		SET total_flow_liters = $1, 
		    last_flow_update_at = $2 
		WHERE device_id = $3`
	
	_, err = s.db.Exec(updateQuery, newTotalFlow, timestamp, deviceID)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to update flow accumulation: %v", err)
	} else {
		log.Printf("📊 Flow accumulated for %s: +%.2fL (%.2f L/min × %.2f min) = Total: %.2fL", 
			deviceID, flowVolume, currentFlowRate, timeDiff, newTotalFlow)
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

// GetFilterModeTracking returns filter mode tracking information
func (s *DatabaseStore) GetFilterModeTracking() map[string]interface{} {
	// Get tracking from device with most recent data (prioritize devices with actual flow)
	query := `
		SELECT filter_mode_started_at, total_flow_liters 
		FROM device_status 
		WHERE filter_mode_started_at IS NOT NULL
		ORDER BY last_seen DESC, total_flow_liters DESC
		LIMIT 1`
	
	var startedAt *time.Time
	var totalFlow float64
	
	err := s.db.QueryRow(query).Scan(&startedAt, &totalFlow)
	if err != nil || startedAt == nil {
		return nil
	}
	
	// Calculate duration in seconds
	duration := time.Since(*startedAt).Seconds()
	
	// Get statistics for today, this week, this month
	stats := s.getFlowStatistics()
	
	result := map[string]interface{}{
		"started_at":        startedAt,
		"duration_seconds":  int(duration),
		"total_flow_liters": totalFlow,
		"statistics":        stats, // Always include stats, even if nil/empty
	}
	
	return result
}

// getFlowStatistics calculates flow statistics for different time periods
func (s *DatabaseStore) getFlowStatistics() map[string]interface{} {
	now := time.Now()
	
	// Today's stats
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	log.Printf("📊 Getting today's stats: %s to %s", todayStart.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"))
	todayStats := s.getFlowByPeriod(todayStart, now)
	
	// This week's stats (Monday to now)
	weekStart := todayStart
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	log.Printf("📊 Getting week's stats: %s to %s", weekStart.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"))
	weekStats := s.getFlowByPeriod(weekStart, now)
	
	// This month's stats
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	log.Printf("📊 Getting month's stats: %s to %s", monthStart.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"))
	monthStats := s.getFlowByPeriod(monthStart, now)
	
	if todayStats == nil && weekStats == nil && monthStats == nil {
		log.Printf("⚠️  All statistics are nil!")
		return map[string]interface{}{
			"today": map[string]interface{}{
				"drinking_water_liters": 0,
				"household_water_liters": 0,
				"total_liters": 0,
			},
			"this_week": map[string]interface{}{
				"drinking_water_liters": 0,
				"household_water_liters": 0,
				"total_liters": 0,
			},
			"this_month": map[string]interface{}{
				"drinking_water_liters": 0,
				"household_water_liters": 0,
				"total_liters": 0,
			},
		}
	}
	
	return map[string]interface{}{
		"today": todayStats,
		"this_week": weekStats,
		"this_month": monthStats,
	}
}

// getFlowByPeriod calculates total flow for each filter mode in a time period
func (s *DatabaseStore) getFlowByPeriod(start, end time.Time) map[string]interface{} {
	// Calculate average flow rate and multiply by time span to estimate volume
	// Note: This is an approximation since we track flow rate (L/min) not cumulative volume
	query := `
		SELECT 
			filter_mode,
			AVG(flow) as avg_flow_rate,
			COUNT(*) as reading_count,
			EXTRACT(EPOCH FROM (MAX(timestamp) - MIN(timestamp))) / 60.0 as duration_minutes
		FROM sensor_readings
		WHERE timestamp >= $1 AND timestamp <= $2
		GROUP BY filter_mode`
	
	rows, err := s.db.Query(query, start, end)
	if err != nil {
		log.Printf("⚠️  Failed to get flow statistics: %v", err)
		return nil
	}
	defer rows.Close()
	
	var drinkingFlow float64
	var householdFlow float64
	var drinkingCount int
	var householdCount int
	
	for rows.Next() {
		var mode string
		var avgFlowRate float64
		var count int
		var durationMinutes float64
		
		if err := rows.Scan(&mode, &avgFlowRate, &count, &durationMinutes); err != nil {
			log.Printf("⚠️  Error scanning flow stats: %v", err)
			continue
		}
		
		// Estimate volume: avg_flow_rate (L/min) × duration (min) = volume (L)
		estimatedVolume := avgFlowRate * durationMinutes
		
		if mode == "drinking_water" {
			drinkingFlow = estimatedVolume
			drinkingCount = count
		} else if mode == "household_water" {
			householdFlow = estimatedVolume
			householdCount = count
		}
	}
	
	totalFlow := drinkingFlow + householdFlow
	totalCount := drinkingCount + householdCount
	
	return map[string]interface{}{
		"drinking_water_liters": drinkingFlow,
		"household_water_liters": householdFlow,
		"total_liters": totalFlow,
		"drinking_water_readings": drinkingCount,
		"household_water_readings": householdCount,
		"total_readings": totalCount,
	}
}

// SetCurrentFilterMode sets the current filter mode for ALL devices
func (s *DatabaseStore) SetCurrentFilterMode(mode models.FilterMode) {
	log.Printf("Setting filter mode to: %s for all devices", mode)
	
	// Update filter mode for ALL devices and reset tracking
	query := `
		UPDATE device_status 
		SET current_filter_mode = $1, 
		    updated_at = NOW(),
		    filter_mode_started_at = NOW(),
		    total_flow_liters = 0,
		    last_flow_update_at = NOW()
		WHERE device_id IN ('stm32_main', 'stm32_pre', 'stm32_post')
	`
	
	result, err := s.db.Exec(query, string(mode))
	if err != nil {
		log.Printf("❌ Failed to set filter mode in database: %v", err)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	log.Printf("✅ Filter mode changed to %s for %d devices (tracking reset)", mode, rowsAffected)
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

// ===== Schedule Management Methods =====

// CreateSchedule creates a new filter schedule
func (s *DatabaseStore) CreateSchedule(schedule *models.FilterSchedule) error {
	query := `
		INSERT INTO filter_schedules (name, filter_mode, start_time, duration_minutes, days_of_week, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err := s.db.QueryRow(query,
		schedule.Name,
		schedule.FilterMode,
		schedule.StartTime,
		schedule.DurationMinutes,
		pq.Array(schedule.DaysOfWeek),
		schedule.IsActive,
	).Scan(&schedule.ID, &schedule.CreatedAt, &schedule.UpdatedAt)

	if err != nil {
		log.Printf("❌ Error creating schedule: %v", err)
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	log.Printf("✅ Created schedule: %s (ID: %d)", schedule.Name, schedule.ID)
	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *DatabaseStore) GetSchedule(id int) (*models.FilterSchedule, error) {
	query := `
		SELECT id, name, filter_mode, start_time, duration_minutes, days_of_week, is_active, created_at, updated_at
		FROM filter_schedules
		WHERE id = $1`

	var schedule models.FilterSchedule
	var startTime time.Time
	err := s.db.QueryRow(query, id).Scan(
		&schedule.ID,
		&schedule.Name,
		&schedule.FilterMode,
		&startTime,
		&schedule.DurationMinutes,
		pq.Array(&schedule.DaysOfWeek),
		&schedule.IsActive,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)
	
	// Convert time.Time to HH:MM:SS string format
	schedule.StartTime = startTime.Format("15:04:05")

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found")
	}
	if err != nil {
		log.Printf("❌ Error getting schedule: %v", err)
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return &schedule, nil
}

// GetAllSchedules retrieves all schedules, optionally filtered by active status
func (s *DatabaseStore) GetAllSchedules(activeOnly bool) ([]models.FilterSchedule, error) {
	query := `
		SELECT id, name, filter_mode, start_time, duration_minutes, days_of_week, is_active, created_at, updated_at
		FROM filter_schedules`

	if activeOnly {
		query += ` WHERE is_active = true`
	}

	query += ` ORDER BY start_time ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("❌ Error getting schedules: %v", err)
		return nil, fmt.Errorf("failed to get schedules: %w", err)
	}
	defer rows.Close()

	var schedules []models.FilterSchedule
	for rows.Next() {
		var schedule models.FilterSchedule
		var startTime time.Time
		err := rows.Scan(
			&schedule.ID,
			&schedule.Name,
			&schedule.FilterMode,
			&startTime,
			&schedule.DurationMinutes,
			pq.Array(&schedule.DaysOfWeek),
			&schedule.IsActive,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			log.Printf("❌ Error scanning schedule: %v", err)
			continue
		}
		
		// Convert time.Time to HH:MM:SS string format
		schedule.StartTime = startTime.Format("15:04:05")
		
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// UpdateSchedule updates an existing schedule
func (s *DatabaseStore) UpdateSchedule(schedule *models.FilterSchedule) error {
	query := `
		UPDATE filter_schedules
		SET name = $1, filter_mode = $2, start_time = $3, duration_minutes = $4, 
		    days_of_week = $5, is_active = $6, updated_at = NOW()
		WHERE id = $7`

	result, err := s.db.Exec(query,
		schedule.Name,
		schedule.FilterMode,
		schedule.StartTime,
		schedule.DurationMinutes,
		pq.Array(schedule.DaysOfWeek),
		schedule.IsActive,
		schedule.ID,
	)

	if err != nil {
		log.Printf("❌ Error updating schedule: %v", err)
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found")
	}

	log.Printf("✅ Updated schedule: %s (ID: %d)", schedule.Name, schedule.ID)
	return nil
}

// DeleteSchedule deletes a schedule by ID
func (s *DatabaseStore) DeleteSchedule(id int) error {
	query := `DELETE FROM filter_schedules WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		log.Printf("❌ Error deleting schedule: %v", err)
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found")
	}

	log.Printf("🗑️  Deleted schedule ID: %d", id)
	return nil
}

// ToggleSchedule enables or disables a schedule
func (s *DatabaseStore) ToggleSchedule(id int, isActive bool) error {
	query := `UPDATE filter_schedules SET is_active = $1, updated_at = NOW() WHERE id = $2`

	result, err := s.db.Exec(query, isActive, id)
	if err != nil {
		log.Printf("❌ Error toggling schedule: %v", err)
		return fmt.Errorf("failed to toggle schedule: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found")
	}

	status := "disabled"
	if isActive {
		status = "enabled"
	}
	log.Printf("✅ Schedule ID %d %s", id, status)
	return nil
}

// ===== Schedule Execution Methods =====

// CreateScheduleExecution creates a new execution record
func (s *DatabaseStore) CreateScheduleExecution(execution *models.ScheduleExecution) error {
	query := `
		INSERT INTO schedule_executions (schedule_id, executed_at, status)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	err := s.db.QueryRow(query,
		execution.ScheduleID,
		execution.ExecutedAt,
		execution.Status,
	).Scan(&execution.ID, &execution.CreatedAt)

	if err != nil {
		log.Printf("❌ Error creating schedule execution: %v", err)
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

// GetScheduleExecution retrieves a single execution by ID
func (s *DatabaseStore) GetScheduleExecution(id int) (*models.ScheduleExecution, error) {
	query := `
		SELECT id, schedule_id, executed_at, completed_at, status, override_reason, created_at
		FROM schedule_executions
		WHERE id = $1`

	var execution models.ScheduleExecution
	var completedAt sql.NullTime
	var overrideReason sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&execution.ID,
		&execution.ScheduleID,
		&execution.ExecutedAt,
		&completedAt,
		&execution.Status,
		&overrideReason,
		&execution.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	if completedAt.Valid {
		execution.CompletedAt = &completedAt.Time
	}
	if overrideReason.Valid {
		execution.OverrideReason = overrideReason.String
	}

	return &execution, nil
}

// GetScheduleExecutions retrieves executions for a specific schedule
func (s *DatabaseStore) GetScheduleExecutions(scheduleID int, limit int) ([]models.ScheduleExecution, error) {
	query := `
		SELECT id, schedule_id, executed_at, completed_at, status, override_reason, created_at
		FROM schedule_executions
		WHERE schedule_id = $1
		ORDER BY executed_at DESC
		LIMIT $2`

	rows, err := s.db.Query(query, scheduleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get executions: %w", err)
	}
	defer rows.Close()

	return s.scanExecutions(rows)
}

// GetAllScheduleExecutions retrieves all executions across all schedules
func (s *DatabaseStore) GetAllScheduleExecutions(limit int) ([]models.ScheduleExecution, error) {
	query := `
		SELECT id, schedule_id, executed_at, completed_at, status, override_reason, created_at
		FROM schedule_executions
		ORDER BY executed_at DESC
		LIMIT $1`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get executions: %w", err)
	}
	defer rows.Close()

	return s.scanExecutions(rows)
}

// UpdateScheduleExecution updates an execution record
func (s *DatabaseStore) UpdateScheduleExecution(execution *models.ScheduleExecution) error {
	query := `
		UPDATE schedule_executions
		SET completed_at = $1, status = $2, override_reason = $3
		WHERE id = $4`

	_, err := s.db.Exec(query,
		execution.CompletedAt,
		execution.Status,
		execution.OverrideReason,
		execution.ID,
	)

	if err != nil {
		log.Printf("❌ Error updating schedule execution: %v", err)
		return fmt.Errorf("failed to update execution: %w", err)
	}

	return nil
}

// scanExecutions is a helper to scan execution rows
func (s *DatabaseStore) scanExecutions(rows *sql.Rows) ([]models.ScheduleExecution, error) {
	var executions []models.ScheduleExecution

	for rows.Next() {
		var execution models.ScheduleExecution
		var completedAt sql.NullTime
		var overrideReason sql.NullString

		err := rows.Scan(
			&execution.ID,
			&execution.ScheduleID,
			&execution.ExecutedAt,
			&completedAt,
			&execution.Status,
			&overrideReason,
			&execution.CreatedAt,
		)
		if err != nil {
			log.Printf("❌ Error scanning execution: %v", err)
			continue
		}

		if completedAt.Valid {
			execution.CompletedAt = &completedAt.Time
		}
		if overrideReason.Valid {
			execution.OverrideReason = overrideReason.String
		}

		executions = append(executions, execution)
	}

	return executions, nil
}
