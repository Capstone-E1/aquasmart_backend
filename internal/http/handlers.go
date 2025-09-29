package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
	"github.com/Capstone-E1/aquasmart_backend/internal/mqtt"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
)

// Handlers contains all HTTP request handlers
type Handlers struct {
	store      store.DataStore
	mqttClient *mqtt.Client
}

// NewHandlers creates a new handlers instance
func NewHandlers(dataStore store.DataStore, mqttClient *mqtt.Client) *Handlers {
	return &Handlers{
		store:      dataStore,
		mqttClient: mqttClient,
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
		"total_readings":   h.store.GetReadingCount(),
		"active_devices":   len(h.store.GetActiveDevices()),
		"server_time":      time.Now(),
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

// AddSensorData handles POST requests to manually add sensor data (for testing)
func (h *Handlers) AddSensorData(w http.ResponseWriter, r *http.Request) {
	var request struct {
		FilterMode string  `json:"filter_mode"`
		Flow       float64 `json:"flow"`
		Ph         float64 `json:"ph"`
		Turbidity  float64 `json:"turbidity"`
		TDS        float64 `json:"tds"`
	}

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate filter mode
	filterMode := models.FilterMode(request.FilterMode)
	if filterMode != models.FilterModeDrinking && filterMode != models.FilterModeHousehold {
		h.sendErrorResponse(w, "Invalid filter_mode. Use 'drinking_water' or 'household_water'", http.StatusBadRequest)
		return
	}

	// Create sensor reading
	reading := models.SensorReading{
		Timestamp:  time.Now(),
		FilterMode: filterMode,
		Flow:       request.Flow,
		Ph:         request.Ph,
		Turbidity:  request.Turbidity,
		TDS:        request.TDS,
	}

	// Validate the reading
	if !reading.ValidateReading() {
		h.sendErrorResponse(w, "Invalid sensor reading values", http.StatusBadRequest)
		return
	}

	// Store the reading
	h.store.AddSensorReading(reading)

	// Return success response
	response := APIResponse{
		Success: true,
		Message: "Sensor data added successfully",
		Data:    reading,
	}

	w.Header().Set("Content-Type", "application/json")
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

	// Send command via MQTT (optional - skip if MQTT not connected)
	if h.mqttClient.IsConnected() {
		if err := h.mqttClient.PublishFilterCommand(filterCommand); err != nil {
			log.Printf("⚠️  Warning: Failed to send MQTT command: %v", err)
		}
	} else {
		log.Printf("ℹ️  MQTT not connected, filter mode change processed locally only")
	}

	// Start new filtration process (default 50L, can be made configurable)
	h.store.StartFiltrationProcess(request.Mode, 50.0)

	// Return success response
	responseData := map[string]interface{}{
		"command":  filterCommand.Command,
		"mode":     filterCommand.Mode,
		"sent_at":  filterCommand.Timestamp,
		"forced":   request.Force,
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