package store

import (
	"time"
	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

// DataStore defines the interface for data storage operations
type DataStore interface {
	// Health check
	Ping() error
	
	AddSensorReading(models.SensorReading)
	GetLatestReading() (*models.SensorReading, bool)
	GetLatestReadingByMode(models.FilterMode) (*models.SensorReading, bool)
	GetLatestReadingByDevice(string) (*models.SensorReading, bool)
	GetAllLatestReadings() []models.SensorReading
	GetAllLatestReadingsByDevice() map[string]models.SensorReading
	GetRecentReadings(int) []models.SensorReading
	GetRecentReadingsByMode(models.FilterMode, int) []models.SensorReading
	GetRecentReadingsByDevice(string, int) []models.SensorReading
	GetReadingsByDevice(string) []models.SensorReading
	GetReadingsInRange(time.Time, time.Time) []models.SensorReading
	GetReadingCount() int
	GetActiveDevices() []string
	GetCurrentFilterMode() models.FilterMode
	SetCurrentFilterMode(models.FilterMode)
	GetFilterModeTracking() map[string]interface{}
	GetWaterQualityStatus() (*models.WaterQualityStatus, bool)
	GetWaterQualityStatusByMode(models.FilterMode) (*models.WaterQualityStatus, bool)
	GetAllWaterQualityStatus() []models.WaterQualityStatus

	// Filtration process tracking
	GetFiltrationProcess() (*models.FiltrationProcess, bool)
	SetFiltrationProcess(*models.FiltrationProcess)
	UpdateFiltrationProgress(currentFlowRate float64)
	StartFiltrationProcess(mode models.FilterMode, targetVolume float64)
	CompleteFiltrationProcess()
	ClearFiltrationProcess()  // Force clear any filtration process
	CanChangeFilterMode() (bool, string)
	ClearCompletedProcess()

	// Schedule management
	CreateSchedule(*models.FilterSchedule) error
	GetSchedule(int) (*models.FilterSchedule, error)
	GetAllSchedules(activeOnly bool) ([]models.FilterSchedule, error)
	UpdateSchedule(*models.FilterSchedule) error
	DeleteSchedule(int) error
	ToggleSchedule(int, bool) error

	// Schedule execution tracking
	CreateScheduleExecution(*models.ScheduleExecution) error
	GetScheduleExecution(int) (*models.ScheduleExecution, error)
	GetScheduleExecutions(scheduleID int, limit int) ([]models.ScheduleExecution, error)
	GetAllScheduleExecutions(limit int) ([]models.ScheduleExecution, error)
	UpdateScheduleExecution(*models.ScheduleExecution) error
}