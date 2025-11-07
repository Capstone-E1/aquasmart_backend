package store

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

// Store manages sensor data storage and retrieval for filtration system
type Store struct {
	mu                      sync.RWMutex
	sensorReadings          []models.SensorReading
	latestReading           *models.SensorReading           // Latest reading overall
	latestByMode            map[models.FilterMode]*models.SensorReading // Latest reading per filter mode
	latestByDevice          map[string]*models.SensorReading // Latest reading per device
	currentFilterMode       models.FilterMode               // Current active filter mode
	filtrationProcess       *models.FiltrationProcess       // Current filtration process state
	ledCommand              string                          // Current LED command (ON/OFF)
	maxReadings             int
	mlData                  *mlStore                        // ML-related data storage
}

// NewStore creates a new in-memory store
func NewStore(maxReadings int) *Store {
	if maxReadings <= 0 {
		maxReadings = 1000 // Default to store last 1000 readings
	}

	return &Store{
		sensorReadings:    make([]models.SensorReading, 0, maxReadings),
		latestReading:     nil,
		latestByMode:      make(map[models.FilterMode]*models.SensorReading),
		latestByDevice:    make(map[string]*models.SensorReading),
		currentFilterMode: models.FilterModeDrinking, // Default to drinking water mode
		ledCommand:        "OFF",                     // Default LED is OFF
		maxReadings:       maxReadings,
		mlData:            newMLStore(),              // Initialize ML data storage
	}
}

// AddSensorReading stores a new sensor reading
func (s *Store) AddSensorReading(reading models.SensorReading) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add to readings slice
	s.sensorReadings = append(s.sensorReadings, reading)

	// Maintain maximum size by removing oldest entries
	if len(s.sensorReadings) > s.maxReadings {
		s.sensorReadings = s.sensorReadings[1:]
	}

	// Update latest reading overall
	s.latestReading = &reading

	// Update latest reading for this filter mode
	readingCopy := reading
	s.latestByMode[reading.FilterMode] = &readingCopy

	// Update latest reading for this device
	if reading.DeviceID != "" {
		deviceCopy := reading
		s.latestByDevice[reading.DeviceID] = &deviceCopy
	}

	// Note: Do NOT update currentFilterMode here - it should only be set via SetCurrentFilterMode()
	// This allows manual filter mode changes via API to persist even when sensor data arrives
}

// GetLatestReading returns the most recent reading
func (s *Store) GetLatestReading() (*models.SensorReading, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latestReading == nil {
		return nil, false
	}

	// Return a copy to avoid race conditions
	reading := *s.latestReading
	return &reading, true
}

// GetAllLatestReadings returns the latest reading (for compatibility with multi-device APIs)
func (s *Store) GetAllLatestReadings() []models.SensorReading {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latestReading == nil {
		return []models.SensorReading{}
	}

	return []models.SensorReading{*s.latestReading}
}

// GetReadingsInRange returns sensor readings within a time range
func (s *Store) GetReadingsInRange(start, end time.Time) []models.SensorReading {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.SensorReading

	for _, reading := range s.sensorReadings {
		if reading.Timestamp.After(start) && reading.Timestamp.Before(end) {
			result = append(result, reading)
		}
	}

	// Sort by timestamp
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}

// GetRecentReadings returns the most recent N readings
func (s *Store) GetRecentReadings(limit int) []models.SensorReading {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get all readings
	readings := make([]models.SensorReading, len(s.sensorReadings))
	copy(readings, s.sensorReadings)

	// Sort by timestamp descending (most recent first)
	sort.Slice(readings, func(i, j int) bool {
		return readings[i].Timestamp.After(readings[j].Timestamp)
	})

	// Limit results
	if limit > 0 && len(readings) > limit {
		readings = readings[:limit]
	}

	return readings
}

// GetLatestReadingByMode returns the most recent reading for a specific filter mode
func (s *Store) GetLatestReadingByMode(mode models.FilterMode) (*models.SensorReading, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	reading, exists := s.latestByMode[mode]
	if !exists || reading == nil {
		return nil, false
	}

	// Return a copy to avoid race conditions
	readingCopy := *reading
	return &readingCopy, true
}

// GetLatestReadingByDevice returns the most recent reading for a specific device
func (s *Store) GetLatestReadingByDevice(deviceID string) (*models.SensorReading, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	reading, exists := s.latestByDevice[deviceID]
	if !exists || reading == nil {
		return nil, false
	}

	// Return a copy to avoid race conditions
	readingCopy := *reading
	return &readingCopy, true
}

// GetAllLatestReadingsByDevice returns the latest reading for each device
func (s *Store) GetAllLatestReadingsByDevice() map[string]models.SensorReading {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]models.SensorReading)
	for deviceID, reading := range s.latestByDevice {
		if reading != nil {
			result[deviceID] = *reading
		}
	}
	return result
}

// GetCurrentFilterMode returns the current active filter mode
func (s *Store) GetCurrentFilterMode() models.FilterMode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.currentFilterMode
}

// SetCurrentFilterMode sets the current filter mode
func (s *Store) SetCurrentFilterMode(mode models.FilterMode) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentFilterMode = mode
}

// GetFilterModeTracking returns filter mode tracking (in-memory store doesn't track this)
func (s *Store) GetFilterModeTracking() map[string]interface{} {
	return nil // Not supported in in-memory store
}

// GetReadingsByMode returns all readings for a specific filter mode
func (s *Store) GetReadingsByMode(mode models.FilterMode) []models.SensorReading {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.SensorReading
	for _, reading := range s.sensorReadings {
		if reading.FilterMode == mode {
			result = append(result, reading)
		}
	}

	// Sort by timestamp
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}

// GetRecentReadingsByMode returns the most recent N readings for a specific filter mode
func (s *Store) GetRecentReadingsByMode(mode models.FilterMode, limit int) []models.SensorReading {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter readings by mode
	var readings []models.SensorReading
	for _, reading := range s.sensorReadings {
		if reading.FilterMode == mode {
			readings = append(readings, reading)
		}
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(readings, func(i, j int) bool {
		return readings[i].Timestamp.After(readings[j].Timestamp)
	})

	// Limit results
	if limit > 0 && len(readings) > limit {
		readings = readings[:limit]
	}

	return readings
}

// GetReadingsByDevice returns all readings for a specific device
func (s *Store) GetReadingsByDevice(deviceID string) []models.SensorReading {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.SensorReading
	for _, reading := range s.sensorReadings {
		if reading.DeviceID == deviceID {
			result = append(result, reading)
		}
	}

	// Sort by timestamp
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}

// GetRecentReadingsByDevice returns the most recent N readings for a specific device
func (s *Store) GetRecentReadingsByDevice(deviceID string, limit int) []models.SensorReading {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter readings by device
	var readings []models.SensorReading
	for _, reading := range s.sensorReadings {
		if reading.DeviceID == deviceID {
			readings = append(readings, reading)
		}
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(readings, func(i, j int) bool {
		return readings[i].Timestamp.After(readings[j].Timestamp)
	})

	// Limit results
	if limit > 0 && len(readings) > limit {
		readings = readings[:limit]
	}

	return readings
}

// GetActiveDevices returns device status indicator (for compatibility)
func (s *Store) GetActiveDevices() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latestReading != nil {
		return []string{"aquasmart_filtration_system"}
	}
	return []string{}
}

// GetWaterQualityStatus returns the latest water quality assessment
func (s *Store) GetWaterQualityStatus() (*models.WaterQualityStatus, bool) {
	reading, exists := s.GetLatestReading()
	if !exists {
		return nil, false
	}

	status := reading.ToWaterQualityStatus()
	return &status, true
}

// GetWaterQualityStatusByMode returns the latest water quality assessment for a specific filter mode
func (s *Store) GetWaterQualityStatusByMode(mode models.FilterMode) (*models.WaterQualityStatus, bool) {
	reading, exists := s.GetLatestReadingByMode(mode)
	if !exists {
		return nil, false
	}

	status := reading.ToWaterQualityStatus()
	return &status, true
}

// GetAllWaterQualityStatus returns water quality status for all filter modes
func (s *Store) GetAllWaterQualityStatus() []models.WaterQualityStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var statuses []models.WaterQualityStatus
	for _, reading := range s.latestByMode {
		if reading != nil {
			status := reading.ToWaterQualityStatus()
			statuses = append(statuses, status)
		}
	}

	return statuses
}

// GetReadingCount returns the total number of stored readings
func (s *Store) GetReadingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.sensorReadings)
}

// ClearReadings removes all stored readings (useful for testing)
func (s *Store) ClearReadings() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sensorReadings = make([]models.SensorReading, 0, s.maxReadings)
	s.latestReading = nil
}

// === Filtration Process Methods ===

// GetFiltrationProcess returns the current filtration process state
func (s *Store) GetFiltrationProcess() (*models.FiltrationProcess, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.filtrationProcess == nil {
		return nil, false
	}

	// Return a copy to avoid race conditions
	processCopy := *s.filtrationProcess
	return &processCopy, true
}

// SetFiltrationProcess sets the current filtration process state
func (s *Store) SetFiltrationProcess(process *models.FiltrationProcess) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if process == nil {
		s.filtrationProcess = nil
		return
	}

	// Store a copy to avoid external modifications
	processCopy := *process
	s.filtrationProcess = &processCopy
}

// UpdateFiltrationProgress updates the current filtration progress with flow rate
func (s *Store) UpdateFiltrationProgress(currentFlowRate float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.filtrationProcess == nil {
		return
	}

	s.filtrationProcess.UpdateProgress(currentFlowRate)
}

// StartFiltrationProcess starts a new filtration process
func (s *Store) StartFiltrationProcess(mode models.FilterMode, targetVolume float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.filtrationProcess = models.NewFiltrationProcess(mode, targetVolume)
	s.currentFilterMode = mode
}

// CompleteFiltrationProcess marks the current filtration process as completed
func (s *Store) CompleteFiltrationProcess() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.filtrationProcess != nil {
		s.filtrationProcess.State = models.FiltrationStateCompleted
		s.filtrationProcess.Progress = 100.0
	}
}

// CanChangeFilterMode returns whether filter mode can be changed and the reason if not
func (s *Store) CanChangeFilterMode() (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.filtrationProcess == nil {
		return true, ""
	}

	return s.filtrationProcess.CanChangeMode()
}

// ClearCompletedProcess removes a completed filtration process
func (s *Store) ClearCompletedProcess() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.filtrationProcess != nil && s.filtrationProcess.State == models.FiltrationStateCompleted {
		s.filtrationProcess = nil
	}
}

// ClearFiltrationProcess force clears any filtration process (for testing/force mode changes)
func (s *Store) ClearFiltrationProcess() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.filtrationProcess = nil
}

// SetLEDCommand sets the LED command for STM32 to poll
func (s *Store) SetLEDCommand(command string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ledCommand = strings.ToUpper(command)
}

// GetLEDCommand retrieves the current LED command
func (s *Store) GetLEDCommand() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.ledCommand
}

// ===== Schedule Management Methods (Stub - Not implemented for in-memory store) =====

// CreateSchedule is not implemented for in-memory store
func (s *Store) CreateSchedule(schedule *models.FilterSchedule) error {
	return fmt.Errorf("schedule management not supported in memory store")
}

// GetSchedule is not implemented for in-memory store
func (s *Store) GetSchedule(id int) (*models.FilterSchedule, error) {
	return nil, fmt.Errorf("schedule management not supported in memory store")
}

// GetAllSchedules is not implemented for in-memory store
func (s *Store) GetAllSchedules(activeOnly bool) ([]models.FilterSchedule, error) {
	return nil, fmt.Errorf("schedule management not supported in memory store")
}

// UpdateSchedule is not implemented for in-memory store
func (s *Store) UpdateSchedule(schedule *models.FilterSchedule) error {
	return fmt.Errorf("schedule management not supported in memory store")
}

// DeleteSchedule is not implemented for in-memory store
func (s *Store) DeleteSchedule(id int) error {
	return fmt.Errorf("schedule management not supported in memory store")
}

// ToggleSchedule is not implemented for in-memory store
func (s *Store) ToggleSchedule(id int, isActive bool) error {
	return fmt.Errorf("schedule management not supported in memory store")
}

// CreateScheduleExecution is not implemented for in-memory store
func (s *Store) CreateScheduleExecution(execution *models.ScheduleExecution) error {
	return fmt.Errorf("schedule execution tracking not supported in memory store")
}

// GetScheduleExecution is not implemented for in-memory store
func (s *Store) GetScheduleExecution(id int) (*models.ScheduleExecution, error) {
	return nil, fmt.Errorf("schedule execution tracking not supported in memory store")
}

// GetScheduleExecutions is not implemented for in-memory store
func (s *Store) GetScheduleExecutions(scheduleID int, limit int) ([]models.ScheduleExecution, error) {
	return nil, fmt.Errorf("schedule execution tracking not supported in memory store")
}

// GetAllScheduleExecutions is not implemented for in-memory store
func (s *Store) GetAllScheduleExecutions(limit int) ([]models.ScheduleExecution, error) {
	return nil, fmt.Errorf("schedule execution tracking not supported in memory store")
}

// UpdateScheduleExecution is not implemented for in-memory store
func (s *Store) UpdateScheduleExecution(execution *models.ScheduleExecution) error {
	return fmt.Errorf("schedule execution tracking not supported in memory store")
}