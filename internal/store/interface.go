package store

import (
	"time"
	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

// DataStore defines the interface for data storage operations
type DataStore interface {
	AddSensorReading(models.SensorReading)
	GetLatestReading() (*models.SensorReading, bool)
	GetLatestReadingByMode(models.FilterMode) (*models.SensorReading, bool)
	GetAllLatestReadings() []models.SensorReading
	GetRecentReadings(int) []models.SensorReading
	GetRecentReadingsByMode(models.FilterMode, int) []models.SensorReading
	GetReadingsInRange(time.Time, time.Time) []models.SensorReading
	GetReadingCount() int
	GetActiveDevices() []string
	GetCurrentFilterMode() models.FilterMode
	SetCurrentFilterMode(models.FilterMode)
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

	// LED control commands
	SetLEDCommand(string)
	GetLEDCommand() string
}