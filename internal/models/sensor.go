package models

import (
	"fmt"
	"strings"
	"time"
)

// SensorReading represents a complete sensor reading from the STM32/ESP8266 device
type SensorReading struct {
	DeviceID   string     `json:"device_id"`
	Timestamp  time.Time  `json:"timestamp"`
	FilterMode FilterMode `json:"filter_mode"`
	Flow       float64    `json:"flow"`
	Ph         float64    `json:"ph"`
	Turbidity  float64    `json:"turbidity"`
	TDS        float64    `json:"tds"`
}

func ConvertVoltageToTurbidity(voltage float64) float64 {
	if voltage < 0 {
		return 0
	}
	if voltage > 3.0 {
		voltage = 3.0
	}
	return (-5 * 1000 / 3.3 * voltage) + 1005.0
}

func ConvertVoltageToTDS(voltage float64) float64 {
	if voltage < 0 {
		return 0
	}
	if voltage > 2.3 {
		voltage = 2.3
	}
	return (voltage / 2.3) * 1000.0
}

func ConvertVoltageToPh(voltage float64) float64 {
	if voltage < 0 {
		return 0
	}
	if voltage > 3.3 {
		voltage = 3.3
	}
	return (voltage / 3.3) * 14.0
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
	DeviceID       string     `json:"device_id"`
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

// GetDeviceType determines if device is pre-filtration or post-filtration based on device ID
func (s *SensorReading) GetDeviceType() string {
	deviceID := strings.ToLower(s.DeviceID)
	if strings.Contains(deviceID, "pre") {
		return "pre_filtration"
	}
	if strings.Contains(deviceID, "post") {
		return "post_filtration"
	}
	return "unknown"
}

// IsPreFiltration returns true if this is a pre-filtration device
func (s *SensorReading) IsPreFiltration() bool {
	return s.GetDeviceType() == "pre_filtration"
}

// IsPostFiltration returns true if this is a post-filtration device
func (s *SensorReading) IsPostFiltration() bool {
	return s.GetDeviceType() == "post_filtration"
}

// ValidateReading checks if sensor values are within acceptable ranges
func (s *SensorReading) ValidateReading() bool {
	// DeviceID must be specified and valid
	if s.DeviceID == "" {
		return false
	}
	if !s.IsValidDeviceID() {
		return false
	}
	// Flow should be non-negative (L/min units)
	if s.Flow < 0 {
		return false
	}
	// Ph should be between 0-14 (pH scale)
	if s.Ph < 0 || s.Ph > 14 {
		return false
	}
	// Turbidity should be between 0-1000 NTU
	if s.Turbidity < 0 || s.Turbidity > 1000 {
		return false
	}
	// TDS should be between 0-1000 PPM
	if s.TDS < 0 || s.TDS > 1000 {
		return false
	}
	// FilterMode should be valid
	if s.FilterMode != FilterModeDrinking && s.FilterMode != FilterModeHousehold {
		return false
	}
	return true
}

// IsValidDeviceID checks if the device_id is one of the allowed values
func (s *SensorReading) IsValidDeviceID() bool {
	deviceID := strings.ToLower(s.DeviceID)
	return deviceID == "stm32_pre" || deviceID == "stm32_post" || deviceID == "stm32_main"
}

// GetPhStatus returns the pH status based on water quality standards
func (s *SensorReading) GetPhStatus() string {
	switch {
	case s.Ph < 7.0:
		return "Dangerously Acidic"
	case s.Ph > 8.5:
		return "Dangerously Alkaline"
	default:
		return "Normal"
	}
}

// GetTurbidityStatus returns turbidity status based on water quality standards
func (s *SensorReading) GetTurbidityStatus() string {
	switch {
	case s.Turbidity > 4.0:
		return "Poor"
	case s.Turbidity > 1.0:
		return "Good"
	default:
		return "Excellent"
	}
}

// GetTDSStatus returns TDS status based on water quality standards
func (s *SensorReading) GetTDSStatus() string {
	switch {
	case s.TDS > 900:
		return "Poor"
	case s.TDS < 900 && s.TDS > 600:
		return "Fair"
	case s.TDS < 600 && s.TDS > 300:
		return "Good"
	default:
		return "Excellent"
	}
}

// ToWaterQualityStatus converts a SensorReading to WaterQualityStatus with assessments
func (s *SensorReading) ToWaterQualityStatus() WaterQualityStatus {
	phStatus := s.GetPhStatus()
	turbStatus := s.GetTurbidityStatus()
	tdsStatus := s.GetTDSStatus()

	// Determine overall quality
	overallQuality := "Good"
	if phStatus != "Normal" || turbStatus == "Poor" || tdsStatus == "Poor" {
		overallQuality = "Danger"
	} else if (turbStatus == "Good" || tdsStatus == "Fair") && phStatus == "Normal" {
		overallQuality = "Good"
	} else if turbStatus == "Excellent" || tdsStatus == "Excellent" || phStatus == "Normal" {
		overallQuality = "Excellent"
	}

	return WaterQualityStatus{
		DeviceID:       s.DeviceID,
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
	CurrentMode         FilterMode      `json:"current_mode"`
	IsActive            bool            `json:"is_active"`
	LastChanged         time.Time       `json:"last_changed"`
	Timestamp           time.Time       `json:"timestamp"`
	FiltrationState     FiltrationState `json:"filtration_state"`
	ProcessStartedAt    *time.Time      `json:"process_started_at,omitempty"`
	EstimatedCompletion *time.Time      `json:"estimated_completion,omitempty"`
}

// FiltrationState represents the current state of the filtration process
type FiltrationState string

const (
	FiltrationStateIdle       FiltrationState = "idle"
	FiltrationStateProcessing FiltrationState = "processing"
	FiltrationStateCompleted  FiltrationState = "completed"
	FiltrationStateSwitching  FiltrationState = "switching"
)

// FiltrationProcess represents the current filtration process details
type FiltrationProcess struct {
	State       FiltrationState `json:"state"`
	CurrentMode FilterMode      `json:"current_mode"`
	StartedAt   time.Time       `json:"started_at"`
	LastUpdated time.Time       `json:"last_updated"`

	// Flow-based tracking (PRIMARY)
	TargetVolume    float64 `json:"target_volume"`     // Total liters to filter
	ProcessedVolume float64 `json:"processed_volume"`  // Liters already processed
	CurrentFlowRate float64 `json:"current_flow_rate"` // Current L/min from sensor

	// Time-based estimation (SECONDARY)
	EstimatedDuration   time.Duration `json:"estimated_duration"`
	EstimatedCompletion time.Time     `json:"estimated_completion"`

	// Progress calculation
	Progress     float64 `json:"progress"` // 0-100%
	CanInterrupt bool    `json:"can_interrupt"`
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
		CurrentMode:     mode,
		IsActive:        isActive,
		LastChanged:     time.Now(),
		Timestamp:       time.Now(),
		FiltrationState: FiltrationStateIdle,
	}
}

// NewFiltrationProcess creates a new filtration process
func NewFiltrationProcess(mode FilterMode, targetVolume float64) *FiltrationProcess {
	now := time.Now()
	return &FiltrationProcess{
		State:           FiltrationStateProcessing,
		CurrentMode:     mode,
		StartedAt:       now,
		LastUpdated:     now,
		TargetVolume:    targetVolume,
		ProcessedVolume: 0.0,
		CurrentFlowRate: 0.0,
		Progress:        0.0,
		CanInterrupt:    false, // Default: cannot interrupt during initial phase
	}
}

// UpdateProgress calculates and updates the filtration progress
func (fp *FiltrationProcess) UpdateProgress(currentFlowRate float64) {
	fp.CurrentFlowRate = currentFlowRate
	now := time.Now()

	// Calculate volume processed since last update based on flow rate
	if !fp.LastUpdated.IsZero() {
		timeDelta := now.Sub(fp.LastUpdated).Minutes()
		volumeDelta := currentFlowRate * timeDelta
		fp.ProcessedVolume += volumeDelta
	}

	fp.LastUpdated = now

	// Calculate progress (flow-based is primary)
	if fp.TargetVolume > 0 {
		fp.Progress = (fp.ProcessedVolume / fp.TargetVolume) * 100
		if fp.Progress > 100 {
			fp.Progress = 100
		}

		// Estimate completion time based on current flow rate
		if currentFlowRate > 0 {
			remainingVolume := fp.TargetVolume - fp.ProcessedVolume
			if remainingVolume > 0 {
				remainingMinutes := remainingVolume / currentFlowRate
				fp.EstimatedCompletion = now.Add(time.Duration(remainingMinutes) * time.Minute)
			} else {
				fp.EstimatedCompletion = now // Completed
			}
		}
	} else {
		// Fallback: Time-based progress if volume not available
		if fp.EstimatedDuration > 0 {
			elapsed := now.Sub(fp.StartedAt)
			fp.Progress = (elapsed.Seconds() / fp.EstimatedDuration.Seconds()) * 100
			if fp.Progress > 100 {
				fp.Progress = 100
			}
		}
	}

	// Mark as completed if progress reaches 100%
	if fp.Progress >= 100 {
		fp.State = FiltrationStateCompleted
	}

	// Allow interruption after 10% progress (configurable)
	if fp.Progress >= 10 {
		fp.CanInterrupt = true
	}
}

// IsProcessingOrSwitching returns true if the filtration is in a state that blocks mode changes
func (fp *FiltrationProcess) IsProcessingOrSwitching() bool {
	return fp.State == FiltrationStateProcessing || fp.State == FiltrationStateSwitching
}

// CanChangeMode returns whether filter mode can be changed and the reason if not
func (fp *FiltrationProcess) CanChangeMode() (bool, string) {
	switch fp.State {
	case FiltrationStateIdle, FiltrationStateCompleted:
		return true, ""
	case FiltrationStateProcessing:
		if fp.CanInterrupt {
			return true, "filtration_interruptible"
		}
		return false, "filtration_in_progress"
	case FiltrationStateSwitching:
		return false, "mode_change_in_progress"
	default:
		return false, "unknown_state"
	}
}

// GetStatusMessage returns a human-readable status message
func (fp *FiltrationProcess) GetStatusMessage() string {
	switch fp.State {
	case FiltrationStateIdle:
		return "System is idle"
	case FiltrationStateProcessing:
		return fmt.Sprintf("Filtering %.1fL of %.1fL (%.1f%% complete)",
			fp.ProcessedVolume, fp.TargetVolume, fp.Progress)
	case FiltrationStateCompleted:
		return "Filtration completed successfully"
	case FiltrationStateSwitching:
		return "Switching filter mode"
	default:
		return "Unknown status"
	}
}
