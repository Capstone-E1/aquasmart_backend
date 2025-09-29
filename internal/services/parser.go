package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

// SensorParser handles parsing of sensor data from various sources
type SensorParser struct{}

// NewSensorParser creates a new instance of SensorParser
func NewSensorParser() *SensorParser {
	return &SensorParser{}
}

// ParseSensorJSON parses JSON payload from STM32/ESP8266 device
func (sp *SensorParser) ParseSensorJSON(payload []byte, deviceID string, filterMode models.FilterMode) (*models.SensorReading, error) {
	var sensorData models.SensorData

	// Parse the JSON payload
	if err := json.Unmarshal(payload, &sensorData); err != nil {
		return nil, fmt.Errorf("failed to parse sensor JSON: %w", err)
	}

	// Create sensor reading with current timestamp and filter mode
	reading := &models.SensorReading{
		DeviceID:   deviceID,
		Timestamp:  time.Now(),
		FilterMode: filterMode,
		Flow:       sensorData.Flow,
		Ph:         sensorData.Ph,
		Turbidity:  sensorData.Turbidity,
		TDS:        sensorData.TDS,
	}

	// Validate the reading
	if !reading.ValidateReading() {
		return nil, fmt.Errorf("invalid sensor reading values: Flow=%.2f, pH=%.2f, Turbidity=%.2f, TDS=%.2f",
			reading.Flow, reading.Ph, reading.Turbidity, reading.TDS)
	}

	return reading, nil
}

// ParseSensorString parses comma-separated sensor values (fallback format)
// Expected format: "flow,ph,turbidity,tds"
func (sp *SensorParser) ParseSensorString(payload string, deviceID string, filterMode models.FilterMode) (*models.SensorReading, error) {
	var flow, ph, turbidity, tds float64

	// Parse comma-separated values
	n, err := fmt.Sscanf(payload, "%f,%f,%f,%f", &flow, &ph, &turbidity, &tds)
	if err != nil || n != 4 {
		return nil, fmt.Errorf("failed to parse sensor string: expected 4 values (flow,ph,turbidity,tds), got %d", n)
	}

	// Create sensor reading
	reading := &models.SensorReading{
		DeviceID:   deviceID,
		Timestamp:  time.Now(),
		FilterMode: filterMode,
		Flow:       flow,
		Ph:         ph,
		Turbidity:  turbidity,
		TDS:        tds,
	}

	// Validate the reading
	if !reading.ValidateReading() {
		return nil, fmt.Errorf("invalid sensor reading values: Flow=%.2f, pH=%.2f, Turbidity=%.2f, TDS=%.2f",
			reading.Flow, reading.Ph, reading.Turbidity, reading.TDS)
	}

	return reading, nil
}

// FormatSensorReading formats sensor reading for logging or debugging
func (sp *SensorParser) FormatSensorReading(reading *models.SensorReading) string {
	return fmt.Sprintf("Device: %s, Time: %s, Filter: %s, Flow: %.2f L/min, pH: %.2f, Turbidity: %.2f NTU, TDS: %.2f ppm",
		reading.DeviceID,
		reading.Timestamp.Format("2006-01-02 15:04:05"),
		reading.FilterMode,
		reading.Flow,
		reading.Ph,
		reading.Turbidity,
		reading.TDS)
}