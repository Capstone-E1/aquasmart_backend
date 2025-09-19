package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
	"github.com/Capstone-E1/aquasmart_backend/internal/mqtt"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
)

// Handlers contains all HTTP request handlers
type Handlers struct {
	store      *store.Store
	mqttClient *mqtt.Client
}

// NewHandlers creates a new handlers instance
func NewHandlers(store *store.Store, mqttClient *mqtt.Client) *Handlers {
	return &Handlers{
		store:      store,
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

// SetFilterMode handles POST requests to set the water filter mode
func (h *Handlers) SetFilterMode(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Mode models.FilterMode `json:"mode"`
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

	// Update current filter mode in store
	h.store.SetCurrentFilterMode(request.Mode)

	// Send command via MQTT
	if err := h.mqttClient.PublishFilterCommand(filterCommand); err != nil {
		h.sendErrorResponse(w, "Failed to send filter command", http.StatusInternalServerError)
		return
	}

	// Return success response
	response := APIResponse{
		Success: true,
		Message: "Filter mode command sent successfully",
		Data: map[string]interface{}{
			"command": filterCommand.Command,
			"mode":    filterCommand.Mode,
			"sent_at": filterCommand.Timestamp,
		},
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

	response := APIResponse{
		Success: true,
		Data:    status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}