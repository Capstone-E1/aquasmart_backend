package models

import (
	"time"
)

// SensorReading represents a complete sensor reading from the STM32/ESP8266 device
type SensorReading struct {
	Timestamp  time.Time  `json:"timestamp"`
	FilterMode FilterMode `json:"filter_mode"`
	Flow       float64    `json:"flow"`
	Ph         float64    `json:"ph"`
	Turbidity  float64    `json:"turbidity"`
	TDS        float64    `json:"tds"`
}

// SensorData represents the raw JSON structure received from the device
type SensorData struct {
	Flow      float64 `json:"flow"`
	Ph        float64 `json:"ph"`
	Turbidity float64 `json:"turbidity"`
	TDS       float64 `json:"tds"`
}

// WaterQualityStatus represents the overall water quality assessment
type WaterQualityStatus struct {
	Timestamp      time.Time  `json:"timestamp"`
	FilterMode     FilterMode `json:"filter_mode"`
	Flow           float64    `json:"flow"`
	Ph             float64    `json:"ph"`
	PhStatus       string     `json:"ph_status"`
	Turbidity      float64    `json:"turbidity"`
	TurbStatus     string     `json:"turbidity_status"`
	TDS            float64    `json:"tds"`
	TDSStatus      string     `json:"tds_status"`
	OverallQuality string     `json:"overall_quality"`
}

// ValidateReading checks if sensor values are within acceptable ranges
func (s *SensorReading) ValidateReading() bool {
	// Flow should be non-negative (L/min units)
	if s.Flow < 0 {
		return false
	}
	// Ph should be between 0-14
	if s.Ph < 0 || s.Ph > 14 {
		return false
	}
	// Turbidity should be non-negative (NTU units)
	if s.Turbidity < 0 {
		return false
	}
	// TDS should be non-negative (ppm units)
	if s.TDS < 0 {
		return false
	}
	// FilterMode should be valid
	if s.FilterMode != FilterModeDrinking && s.FilterMode != FilterModeHousehold {
		return false
	}
	return true
}

// GetPhStatus returns the pH status based on water quality standards
func (s *SensorReading) GetPhStatus() string {
	switch {
	case s.Ph < 6.5:
		return "acidic"
	case s.Ph > 8.5:
		return "alkaline"
	default:
		return "normal"
	}
}

// GetTurbidityStatus returns turbidity status based on water quality standards
func (s *SensorReading) GetTurbidityStatus() string {
	switch {
	case s.Turbidity > 4.0:
		return "high"
	case s.Turbidity > 1.0:
		return "moderate"
	default:
		return "low"
	}
}

// GetTDSStatus returns TDS status based on water quality standards
func (s *SensorReading) GetTDSStatus() string {
	switch {
	case s.TDS > 500:
		return "high"
	case s.TDS > 300:
		return "moderate"
	default:
		return "low"
	}
}

// ToWaterQualityStatus converts a SensorReading to WaterQualityStatus with assessments
func (s *SensorReading) ToWaterQualityStatus() WaterQualityStatus {
	phStatus := s.GetPhStatus()
	turbStatus := s.GetTurbidityStatus()
	tdsStatus := s.GetTDSStatus()

	// Determine overall quality
	overallQuality := "good"
	if phStatus != "normal" || turbStatus == "high" || tdsStatus == "high" {
		overallQuality = "poor"
	} else if turbStatus == "moderate" || tdsStatus == "moderate" {
		overallQuality = "moderate"
	}

	return WaterQualityStatus{
		Timestamp:      s.Timestamp,
		FilterMode:     s.FilterMode,
		Flow:           s.Flow,
		Ph:             s.Ph,
		PhStatus:       phStatus,
		Turbidity:      s.Turbidity,
		TurbStatus:     turbStatus,
		TDS:            s.TDS,
		TDSStatus:      tdsStatus,
		OverallQuality: overallQuality,
	}
}

// FilterMode represents the available water filtration modes
type FilterMode string

const (
	FilterModeDrinking  FilterMode = "drinking_water"
	FilterModeHousehold FilterMode = "household_water"
)

// FilterCommand represents a command to control the water filter
type FilterCommand struct {
	Command   string     `json:"command"`
	Mode      FilterMode `json:"mode"`
	Timestamp time.Time  `json:"timestamp"`
}

// FilterStatus represents the current status of the water filter
type FilterStatus struct {
	CurrentMode FilterMode `json:"current_mode"`
	IsActive    bool       `json:"is_active"`
	LastChanged time.Time  `json:"last_changed"`
	Timestamp   time.Time  `json:"timestamp"`
}

// CommandResponse represents a response from the STM32 device
type CommandResponse struct {
	Command   string    `json:"command"`
	Status    string    `json:"status"` // "success", "error", "processing"
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ValidateFilterMode checks if the filter mode is valid
func (fc *FilterCommand) ValidateFilterMode() bool {
	return fc.Mode == FilterModeDrinking || fc.Mode == FilterModeHousehold
}

// NewFilterCommand creates a new filter command
func NewFilterCommand(mode FilterMode) *FilterCommand {
	return &FilterCommand{
		Command:   "set_filter_mode",
		Mode:      mode,
		Timestamp: time.Now(),
	}
}

// NewFilterStatus creates a new filter status
func NewFilterStatus(mode FilterMode, isActive bool) *FilterStatus {
	return &FilterStatus{
		CurrentMode: mode,
		IsActive:    isActive,
		LastChanged: time.Now(),
		Timestamp:   time.Now(),
	}
}