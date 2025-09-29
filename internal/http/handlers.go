package http

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/export"
	"github.com/Capstone-E1/aquasmart_backend/internal/models"
	"github.com/Capstone-E1/aquasmart_backend/internal/mqtt"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
)

// Handlers contains all HTTP request handlers
type Handlers struct {
	store         store.DataStore
	mqttClient    *mqtt.Client
	exportService *export.ExportService
}

// NewHandlers creates a new handlers instance
func NewHandlers(dataStore store.DataStore, mqttClient *mqtt.Client) *Handlers {
	return &Handlers{
		store:         dataStore,
		mqttClient:    mqttClient,
		exportService: export.NewExportService(),
	}
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// GetLatestReadings returns the latest sensor readings (optionally filtered by mode)
func (h *Handlers) GetLatestReadings(w http.ResponseWriter, r *http.Request) {
	filterModeStr := r.URL.Query().Get("filter_mode")

	if filterModeStr != "" {
		// Return reading for specific filter mode
		filterMode := models.FilterMode(filterModeStr)
		if filterMode != models.FilterModeDrinking && filterMode != models.FilterModeHousehold {
			h.sendErrorResponse(w, "Invalid filter_mode. Use 'drinking_water' or 'household_water'", http.StatusBadRequest)
			return
		}

		reading, exists := h.store.GetLatestReadingByMode(filterMode)
		if !exists {
			h.sendErrorResponse(w, "No sensor data available for specified filter mode", http.StatusNotFound)
			return
		}

		response := APIResponse{
			Success: true,
			Data:    reading,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return latest reading overall or all latest readings by mode
	readings := h.store.GetAllLatestReadings()

	response := APIResponse{
		Success: true,
		Data:    readings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetLatestReadingByDevice returns the latest sensor reading (single device system)
func (h *Handlers) GetLatestReadingByDevice(w http.ResponseWriter, r *http.Request) {
	reading, exists := h.store.GetLatestReading()
	if !exists {
		h.sendErrorResponse(w, "No sensor data available", http.StatusNotFound)
		return
	}

	response := APIResponse{
		Success: true,
		Data:    reading,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetWaterQualityStatus returns water quality assessment (optionally filtered by mode)
func (h *Handlers) GetWaterQualityStatus(w http.ResponseWriter, r *http.Request) {
	filterModeStr := r.URL.Query().Get("filter_mode")

	if filterModeStr != "" {
		// Return status for specific filter mode
		filterMode := models.FilterMode(filterModeStr)
		if filterMode != models.FilterModeDrinking && filterMode != models.FilterModeHousehold {
			h.sendErrorResponse(w, "Invalid filter_mode. Use 'drinking_water' or 'household_water'", http.StatusBadRequest)
			return
		}

		status, exists := h.store.GetWaterQualityStatusByMode(filterMode)
		if !exists {
			h.sendErrorResponse(w, "No sensor data available for specified filter mode", http.StatusNotFound)
			return
		}

		response := APIResponse{
			Success: true,
			Data:    status,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return status for all filter modes
	statuses := h.store.GetAllWaterQualityStatus()

	response := APIResponse{
		Success: true,
		Data:    statuses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetWaterQualityStatusByDevice returns water quality assessment (single device system)
func (h *Handlers) GetWaterQualityStatusByDevice(w http.ResponseWriter, r *http.Request) {
	status, exists := h.store.GetWaterQualityStatus()
	if !exists {
		h.sendErrorResponse(w, "No sensor data available", http.StatusNotFound)
		return
	}

	response := APIResponse{
		Success: true,
		Data:    status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRecentReadings returns recent sensor readings (optionally filtered by mode)
func (h *Handlers) GetRecentReadings(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	filterModeStr := r.URL.Query().Get("filter_mode")

	limit := 50 // Default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var readings []models.SensorReading

	if filterModeStr != "" {
		// Return readings for specific filter mode
		filterMode := models.FilterMode(filterModeStr)
		if filterMode != models.FilterModeDrinking && filterMode != models.FilterModeHousehold {
			h.sendErrorResponse(w, "Invalid filter_mode. Use 'drinking_water' or 'household_water'", http.StatusBadRequest)
			return
		}

		readings = h.store.GetRecentReadingsByMode(filterMode, limit)
	} else {
		// Return all recent readings
		readings = h.store.GetRecentReadings(limit)
	}

	response := APIResponse{
		Success: true,
		Data:    readings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetReadingsInRange returns sensor readings within a time range
func (h *Handlers) GetReadingsInRange(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		h.sendErrorResponse(w, "Both start and end time parameters are required", http.StatusBadRequest)
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		h.sendErrorResponse(w, "Invalid start time format. Use RFC3339 format", http.StatusBadRequest)
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		h.sendErrorResponse(w, "Invalid end time format. Use RFC3339 format", http.StatusBadRequest)
		return
	}

	if end.Before(start) {
		h.sendErrorResponse(w, "End time must be after start time", http.StatusBadRequest)
		return
	}

	readings := h.store.GetReadingsInRange(start, end)

	response := APIResponse{
		Success: true,
		Data:    readings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetActiveDevices returns a list of all active device IDs
func (h *Handlers) GetActiveDevices(w http.ResponseWriter, r *http.Request) {
	devices := h.store.GetActiveDevices()

	response := APIResponse{
		Success: true,
		Data:    devices,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSystemStats returns system statistics
func (h *Handlers) GetSystemStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"total_readings": h.store.GetReadingCount(),
		"active_devices": len(h.store.GetActiveDevices()),
		"server_time":    time.Now(),
	}

	response := APIResponse{
		Success: true,
		Data:    stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HealthCheck returns the health status of the API
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
	}

	response := APIResponse{
		Success: true,
		Data:    health,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendErrorResponse sends a standardized error response
func (h *Handlers) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := APIResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// SetFilterMode handles POST requests to set the water filter mode
func (h *Handlers) SetFilterMode(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Mode  models.FilterMode `json:"mode"`
		Force bool              `json:"force,omitempty"` // Optional: force mode change during filtration
	}

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create filter command
	filterCommand := models.NewFilterCommand(request.Mode)

	// Validate filter mode
	if !filterCommand.ValidateFilterMode() {
		h.sendErrorResponse(w, "Invalid filter mode. Use 'drinking_water' or 'household_water'", http.StatusBadRequest)
		return
	}

	// Check if filter mode change is allowed
	canChange, reason := h.store.CanChangeFilterMode()
	if !canChange && !request.Force {
		// Get current filtration process details for error response
		process, exists := h.store.GetFiltrationProcess()
		if exists {
			errorData := map[string]interface{}{
				"error_code":           reason,
				"current_state":        process.State,
				"progress":             process.Progress,
				"processed_volume":     process.ProcessedVolume,
				"target_volume":        process.TargetVolume,
				"estimated_completion": process.EstimatedCompletion,
				"can_force":            process.CanInterrupt,
				"status_message":       process.GetStatusMessage(),
			}

			response := APIResponse{
				Success: false,
				Message: "Cannot change filter mode",
				Error:   reason,
				Data:    errorData,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict) // 409 Conflict
			json.NewEncoder(w).Encode(response)
			return
		}

		h.sendErrorResponse(w, "Cannot change filter mode: "+reason, http.StatusConflict)
		return
	}

	// If forced or process can be interrupted, handle the transition
	if !canChange && request.Force {
		// Check if current process can be interrupted
		if process, exists := h.store.GetFiltrationProcess(); exists && !process.CanInterrupt {
			h.sendErrorResponse(w, "Current filtration process cannot be interrupted", http.StatusConflict)
			return
		}

		// Set process to switching state
		if process, exists := h.store.GetFiltrationProcess(); exists {
			process.State = models.FiltrationStateSwitching
			h.store.SetFiltrationProcess(process)
		}
	}

	// Update current filter mode in store
	h.store.SetCurrentFilterMode(request.Mode)

	// Send command via MQTT
	if err := h.mqttClient.PublishFilterCommand(filterCommand); err != nil {
		h.sendErrorResponse(w, "Failed to send filter command", http.StatusInternalServerError)
		return
	}

	// Determine target volume based on mode (both set to 5L for now)
	var targetVolume float64 = 5.0 // 5L for both drinking and household water

	// Start new filtration process
	h.store.StartFiltrationProcess(request.Mode, targetVolume)

	// Return success response
	responseData := map[string]interface{}{
		"command": filterCommand.Command,
		"mode":    filterCommand.Mode,
		"sent_at": filterCommand.Timestamp,
		"forced":  request.Force,
	}

	if request.Force {
		responseData["message"] = "Filter mode changed (previous process interrupted)"
	}

	response := APIResponse{
		Success: true,
		Message: "Filter mode command sent successfully",
		Data:    responseData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFilterStatus handles GET requests to get the current filter status
func (h *Handlers) GetFilterStatus(w http.ResponseWriter, r *http.Request) {
	currentMode := h.store.GetCurrentFilterMode()

	// Check if we have recent data for this mode to determine if it's active
	_, hasData := h.store.GetLatestReadingByMode(currentMode)

	status := models.NewFilterStatus(currentMode, hasData)

	// Include filtration process info if available
	if process, exists := h.store.GetFiltrationProcess(); exists {
		status.FiltrationState = process.State
		status.ProcessStartedAt = &process.StartedAt
		status.EstimatedCompletion = &process.EstimatedCompletion
	}

	response := APIResponse{
		Success: true,
		Data:    status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFiltrationStatus handles GET requests to get detailed filtration process status
func (h *Handlers) GetFiltrationStatus(w http.ResponseWriter, r *http.Request) {
	process, exists := h.store.GetFiltrationProcess()
	if !exists {
		// No active filtration process
		idleData := map[string]interface{}{
			"state":           models.FiltrationStateIdle,
			"can_change_mode": true,
			"message":         "No active filtration process",
		}

		response := APIResponse{
			Success: true,
			Data:    idleData,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Calculate whether mode can be changed
	canChange, reason := process.CanChangeMode()

	// Prepare detailed response
	responseData := map[string]interface{}{
		"state":                process.State,
		"current_mode":         process.CurrentMode,
		"started_at":           process.StartedAt,
		"last_updated":         process.LastUpdated,
		"progress":             process.Progress,
		"processed_volume":     process.ProcessedVolume,
		"target_volume":        process.TargetVolume,
		"current_flow_rate":    process.CurrentFlowRate,
		"estimated_completion": process.EstimatedCompletion,
		"can_change_mode":      canChange,
		"can_interrupt":        process.CanInterrupt,
		"status_message":       process.GetStatusMessage(),
	}

	if !canChange {
		responseData["block_reason"] = reason
	}

	response := APIResponse{
		Success: true,
		Data:    responseData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ExportHistoryExcel handles GET requests to export purification history as Excel
func (h *Handlers) ExportHistoryExcel(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for date range filtering
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	filterMode := r.URL.Query().Get("filter_mode")

	var start, end time.Time
	var err error

	// Set default time range (last 30 days if not specified)
	if startStr == "" {
		start = time.Now().AddDate(0, 0, -30)
	} else {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			h.sendErrorResponse(w, "Invalid start date format. Use RFC3339 format", http.StatusBadRequest)
			return
		}
	}

	if endStr == "" {
		end = time.Now()
	} else {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			h.sendErrorResponse(w, "Invalid end date format. Use RFC3339 format", http.StatusBadRequest)
			return
		}
	}

	// Get sensor readings from the store
	readings := h.store.GetReadingsInRange(start, end)

	// Filter by mode if specified
	if filterMode != "" {
		filteredReadings := []models.SensorReading{}
		for _, reading := range readings {
			if string(reading.FilterMode) == filterMode {
				filteredReadings = append(filteredReadings, reading)
			}
		}
		readings = filteredReadings
	}

	// Generate water quality assessments
	waterQualityStatuses := []models.WaterQualityStatus{}
	for _, reading := range readings {
		status := reading.ToWaterQualityStatus()
		waterQualityStatuses = append(waterQualityStatuses, status)
	}

	// Create mock filtration history (in real implementation, this would come from database)
	filtrationHistory := h.generateFiltrationHistory(readings)

	// Prepare export data
	exportData := export.ExportData{
		SensorReadings:          readings,
		WaterQualityAssessments: waterQualityStatuses,
		FiltrationHistory:       filtrationHistory,
		ExportMetadata: export.ExportMetadata{
			GeneratedAt:   time.Now(),
			DateRange:     fmt.Sprintf("%s to %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
			TotalReadings: len(readings),
			FilterModes:   []string{"drinking_water", "household_water"},
			DeviceInfo:    "AquaSmart IoT Device",
		},
	}

	// Generate Excel file
	excelFile, err := h.exportService.GenerateExcel(exportData)
	if err != nil {
		h.sendErrorResponse(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}

	// Set response headers
	filename := fmt.Sprintf("aquasmart_history_%s_to_%s.xlsx",
		start.Format("2006-01-02"), end.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Write Excel file to response
	if err := excelFile.Write(w); err != nil {
		h.sendErrorResponse(w, "Failed to write Excel file", http.StatusInternalServerError)
		return
	}
}

// ExportHistoryCSV handles GET requests to export purification history as CSV
func (h *Handlers) ExportHistoryCSV(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for date range filtering
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	filterMode := r.URL.Query().Get("filter_mode")

	var start, end time.Time
	var err error

	// Set default time range (last 30 days if not specified)
	if startStr == "" {
		start = time.Now().AddDate(0, 0, -30)
	} else {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			h.sendErrorResponse(w, "Invalid start date format. Use RFC3339 format", http.StatusBadRequest)
			return
		}
	}

	if endStr == "" {
		end = time.Now()
	} else {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			h.sendErrorResponse(w, "Invalid end date format. Use RFC3339 format", http.StatusBadRequest)
			return
		}
	}

	// Get sensor readings from the store
	readings := h.store.GetReadingsInRange(start, end)

	// Filter by mode if specified
	if filterMode != "" {
		filteredReadings := []models.SensorReading{}
		for _, reading := range readings {
			if string(reading.FilterMode) == filterMode {
				filteredReadings = append(filteredReadings, reading)
			}
		}
		readings = filteredReadings
	}

	// Generate CSV data
	csvData, err := h.exportService.GenerateCSV(readings)
	if err != nil {
		h.sendErrorResponse(w, "Failed to generate CSV data", http.StatusInternalServerError)
		return
	}

	// Set response headers
	filename := fmt.Sprintf("aquasmart_history_%s_to_%s.csv",
		start.Format("2006-01-02"), end.Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Write CSV data to response
	csvWriter := csv.NewWriter(w)
	if err := h.exportService.WriteCSV(csvWriter, csvData); err != nil {
		h.sendErrorResponse(w, "Failed to write CSV data", http.StatusInternalServerError)
		return
	}
}

// generateFiltrationHistory creates mock filtration history from sensor readings
// In a real implementation, this would query a dedicated filtration_sessions table
func (h *Handlers) generateFiltrationHistory(readings []models.SensorReading) []export.FiltrationRecord {
	history := []export.FiltrationRecord{}

	if len(readings) == 0 {
		return history
	}

	// Group readings by day and mode to simulate filtration sessions
	sessions := make(map[string][]models.SensorReading)
	for _, reading := range readings {
		key := fmt.Sprintf("%s_%s", reading.Timestamp.Format("2006-01-02"), reading.FilterMode)
		sessions[key] = append(sessions[key], reading)
	}

	id := 1
	for _, sessionReadings := range sessions {
		if len(sessionReadings) == 0 {
			continue
		}

		startTime := sessionReadings[0].Timestamp
		endTime := sessionReadings[len(sessionReadings)-1].Timestamp
		duration := endTime.Sub(startTime)

		// Calculate processed volume based on average flow
		var totalFlow float64
		for _, reading := range sessionReadings {
			totalFlow += reading.Flow
		}
		avgFlow := totalFlow / float64(len(sessionReadings))
		processedVolume := avgFlow * duration.Minutes()

		// Target volume is 5L for both modes (as updated earlier)
		targetVolume := 5.0
		progress := (processedVolume / targetVolume) * 100
		if progress > 100 {
			progress = 100
		}

		status := "completed"
		if progress < 100 {
			status = "incomplete"
		}

		record := export.FiltrationRecord{
			ID:              id,
			StartTime:       startTime,
			EndTime:         endTime,
			FilterMode:      string(sessionReadings[0].FilterMode),
			TargetVolume:    targetVolume,
			ProcessedVolume: processedVolume,
			Progress:        progress,
			Status:          status,
			Duration:        duration.Round(time.Second).String(),
		}

		history = append(history, record)
		id++
	}

	return history
}
