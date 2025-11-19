package http

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/export"
	"github.com/Capstone-E1/aquasmart_backend/internal/ml"
	"github.com/Capstone-E1/aquasmart_backend/internal/models"
	"github.com/Capstone-E1/aquasmart_backend/internal/mqtt"
	"github.com/Capstone-E1/aquasmart_backend/internal/services"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
	"github.com/go-chi/chi/v5"
)

// Handlers contains all HTTP request handlers
type Handlers struct {
	store         store.DataStore
	exportService *export.ExportService
	scheduler     *services.Scheduler
	mqtt          *mqtt.Client
  mlService     *ml.MLService
}

// NewHandlers creates a new handlers instance
func NewHandlers(dataStore store.DataStore, scheduler *services.Scheduler, mqttClient *mqtt.Client, mlService *ml.MLService) *Handlers {
	return &Handlers{
		store:         dataStore,
		exportService: export.NewExportService(),
		scheduler:     scheduler,
		mqtt:          mqttClient,
		mlService:     mlService,
	}
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// HealthCheck returns health status of the application
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "aquasmart-backend",
	}

	// Optional: Add database health check
	if err := h.store.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["database"] = "error"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		health["database"] = "connected"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// GetLatestReadings returns the latest sensor readings (optionally filtered by mode or device)
func (h *Handlers) GetLatestReadings(w http.ResponseWriter, r *http.Request) {
	filterModeStr := r.URL.Query().Get("filter_mode")
	deviceID := r.URL.Query().Get("device_id")

	// If device_id is specified, return reading for that device
	if deviceID != "" {
		reading, exists := h.store.GetLatestReadingByDevice(deviceID)
		if !exists {
			h.sendErrorResponse(w, "No sensor data available for specified device", http.StatusNotFound)
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

// GetRecentReadings returns recent sensor readings (optionally filtered by mode or device)
func (h *Handlers) GetRecentReadings(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	filterModeStr := r.URL.Query().Get("filter_mode")
	deviceID := r.URL.Query().Get("device_id")

	limit := 50 // Default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var readings []models.SensorReading

	// If device_id is specified, filter by device
	if deviceID != "" {
		readings = h.store.GetRecentReadingsByDevice(deviceID, limit)
	} else if filterModeStr != "" {
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

	// Process reading for ML analysis (anomaly detection & prediction updates)
	if h.mlService != nil {
		go h.mlService.ProcessNewReading(&reading)
	}

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
		Mode           models.FilterMode `json:"mode"`
		Force          bool              `json:"force,omitempty"`           // Optional: force mode change during filtration
		OverrideReason string            `json:"override_reason,omitempty"` // Optional: reason for manual override
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

	// If forced, handle the transition (override CanInterrupt check for testing)
	if !canChange && request.Force {
		log.Printf("⚠️  Force flag enabled - interrupting filtration process")
		
		// Set process to switching state or clear it
		if process, exists := h.store.GetFiltrationProcess(); exists {
			// Check if process naturally allows interruption
			if process.CanInterrupt {
				log.Printf("   Process can be interrupted naturally (progress: %.1f%%)", process.Progress)
				process.State = models.FiltrationStateSwitching
				h.store.SetFiltrationProcess(process)
			} else {
				// Force override - clear the filtration process entirely
				log.Printf("   Force override: clearing filtration process (progress: %.1f%%)", process.Progress)
				h.store.ClearFiltrationProcess()
			}
		}
	}

	// Update current filter mode in store
	h.store.SetCurrentFilterMode(request.Mode)

	// Publish filter command via MQTT
	if h.mqtt != nil {
		if err := h.mqtt.PublishFilterCommand(request.Mode); err != nil {
			log.Printf("⚠️  Failed to publish filter command via MQTT: %v", err)
		}
	}

	// Notify scheduler about manual override
	if h.scheduler != nil {
		overrideReason := request.OverrideReason
		if overrideReason == "" {
			overrideReason = fmt.Sprintf("Manual mode change to %s", request.Mode)
		}
		h.scheduler.HandleManualOverride(overrideReason)
	}

	// Note: With HTTP-only communication, STM32 will poll for commands via GET /api/v1/sensors/stm32/command
	log.Printf("✅ Filter mode changed to %s (STM32 will poll for this command)", request.Mode)

	// Optional: Start filtration process only if start_filtration flag is true
	var startFiltration bool = false // Default: don't start filtration automatically
	
	if startFiltration {
		// Determine target volume based on mode (both set to 5L for now)
		var targetVolume float64 = 5.0 // 5L for both drinking and household water
		// Start new filtration process
		h.store.StartFiltrationProcess(request.Mode, targetVolume)
		log.Printf("🌊 Started filtration process: mode=%s, target=%.1fL", request.Mode, targetVolume)
	}

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

// GetFilterStatus handles GET requests to get current filter mode and statistics
func (h *Handlers) GetFilterStatus(w http.ResponseWriter, r *http.Request) {
	// Get current filter mode from all active devices
	currentMode := h.store.GetCurrentFilterMode()
	
	// Get filter mode tracking with statistics
	tracking := h.store.GetFilterModeTracking()
	
	// Build response with full tracking info
	responseData := map[string]interface{}{
		"current_mode":          currentMode,
		"filter_mode_tracking":  tracking,
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

// GetAllSensorData returns all sensor data with optional filtering and pagination
func (h *Handlers) GetAllSensorData(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	filterModeStr := r.URL.Query().Get("filter_mode")
	deviceID := r.URL.Query().Get("device_id")
	sortOrder := r.URL.Query().Get("sort") // "asc" or "desc"

	// Set default values
	limit := 100 // Default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	offset := 0 // Default offset
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	if sortOrder == "" {
		sortOrder = "desc" // Default to newest first
	}

	// Get all readings
	allReadings := h.store.GetRecentReadings(10000) // Get a large number to simulate "all"

	// Filter by device if specified
	var filteredReadings []models.SensorReading
	if deviceID != "" {
		for _, reading := range allReadings {
			if reading.DeviceID == deviceID {
				filteredReadings = append(filteredReadings, reading)
			}
		}
		allReadings = filteredReadings
		filteredReadings = []models.SensorReading{} // Reset for mode filtering
	}

	// Filter by mode if specified
	if filterModeStr != "" {
		filterMode := models.FilterMode(filterModeStr)
		if filterMode != models.FilterModeDrinking && filterMode != models.FilterModeHousehold {
			h.sendErrorResponse(w, "Invalid filter_mode. Use 'drinking_water' or 'household_water'", http.StatusBadRequest)
			return
		}

		for _, reading := range allReadings {
			if reading.FilterMode == filterMode {
				filteredReadings = append(filteredReadings, reading)
			}
		}
	} else {
		filteredReadings = allReadings
	}

	// Sort readings
	if sortOrder == "asc" {
		// Sort oldest first
		for i := 0; i < len(filteredReadings)-1; i++ {
			for j := i + 1; j < len(filteredReadings); j++ {
				if filteredReadings[i].Timestamp.After(filteredReadings[j].Timestamp) {
					filteredReadings[i], filteredReadings[j] = filteredReadings[j], filteredReadings[i]
				}
			}
		}
	}
	// Default is already desc (newest first)

	// Apply pagination
	totalRecords := len(filteredReadings)
	start := offset
	end := offset + limit

	if start >= totalRecords {
		filteredReadings = []models.SensorReading{}
	} else {
		if end > totalRecords {
			end = totalRecords
		}
		filteredReadings = filteredReadings[start:end]
	}

	// Prepare response with metadata
	responseData := map[string]interface{}{
		"data": filteredReadings,
		"pagination": map[string]interface{}{
			"total_records":    totalRecords,
			"current_page":     (offset / limit) + 1,
			"per_page":         limit,
			"total_pages":      (totalRecords + limit - 1) / limit,
			"has_next":         end < totalRecords,
			"has_previous":     offset > 0,
		},
		"filters": map[string]interface{}{
			"device_id":   deviceID,
			"filter_mode": filterModeStr,
			"sort_order":  sortOrder,
		},
	}

	response := APIResponse{
		Success: true,
		Data:    responseData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetDeviceReadings returns all readings for a specific device (path parameter)
func (h *Handlers) GetDeviceReadings(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")
	
	if deviceID == "" {
		h.sendErrorResponse(w, "device_id parameter is required", http.StatusBadRequest)
		return
	}

	// Get readings for this device
	readings := h.store.GetReadingsByDevice(deviceID)

	if len(readings) == 0 {
		h.sendErrorResponse(w, "No readings found for device: "+deviceID, http.StatusNotFound)
		return
	}

	response := APIResponse{
		Success: true,
		Data:    readings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAllDevicesLatest returns the latest reading for each device
func (h *Handlers) GetAllDevicesLatest(w http.ResponseWriter, r *http.Request) {
	latestReadings := h.store.GetAllLatestReadingsByDevice()

	if len(latestReadings) == 0 {
		h.sendErrorResponse(w, "No devices with readings found", http.StatusNotFound)
		return
	}

	response := APIResponse{
		Success: true,
		Data:    latestReadings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAllSensorDataSimple returns all sensor data in simple format (just array)
func (h *Handlers) GetAllSensorDataSimple(w http.ResponseWriter, r *http.Request) {
	filterModeStr := r.URL.Query().Get("filter_mode")
	sortOrder := r.URL.Query().Get("sort") // "asc" or "desc"

	// Get all readings
	allReadings := h.store.GetRecentReadings(10000) // Get a large number

	// Filter by mode if specified
	var filteredReadings []models.SensorReading
	if filterModeStr != "" {
		filterMode := models.FilterMode(filterModeStr)
		if filterMode != models.FilterModeDrinking && filterMode != models.FilterModeHousehold {
			h.sendErrorResponse(w, "Invalid filter_mode. Use 'drinking_water' or 'household_water'", http.StatusBadRequest)
			return
		}

		for _, reading := range allReadings {
			if reading.FilterMode == filterMode {
				filteredReadings = append(filteredReadings, reading)
			}
		}
	} else {
		filteredReadings = allReadings
	}

	// Sort readings if specified
	if sortOrder == "asc" {
		// Sort oldest first
		for i := 0; i < len(filteredReadings)-1; i++ {
			for j := i + 1; j < len(filteredReadings); j++ {
				if filteredReadings[i].Timestamp.After(filteredReadings[j].Timestamp) {
					filteredReadings[i], filteredReadings[j] = filteredReadings[j], filteredReadings[i]
				}
			}
		}
	}

	response := APIResponse{
		Success: true,
		Data:    filteredReadings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSensorDataStats returns statistics about all sensor data
func (h *Handlers) GetSensorDataStats(w http.ResponseWriter, r *http.Request) {
	filterModeStr := r.URL.Query().Get("filter_mode")

	// Get all readings
	allReadings := h.store.GetRecentReadings(10000)

	// Filter by mode if specified
	var filteredReadings []models.SensorReading
	if filterModeStr != "" {
		filterMode := models.FilterMode(filterModeStr)
		if filterMode != models.FilterModeDrinking && filterMode != models.FilterModeHousehold {
			h.sendErrorResponse(w, "Invalid filter_mode. Use 'drinking_water' or 'household_water'", http.StatusBadRequest)
			return
		}

		for _, reading := range allReadings {
			if reading.FilterMode == filterMode {
				filteredReadings = append(filteredReadings, reading)
			}
		}
	} else {
		filteredReadings = allReadings
	}

	if len(filteredReadings) == 0 {
		h.sendErrorResponse(w, "No sensor data found", http.StatusNotFound)
		return
	}

	// Calculate statistics
	stats := calculateSensorStats(filteredReadings)

	response := APIResponse{
		Success: true,
		Data:    stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SensorDataStats represents statistics about sensor data
type SensorDataStats struct {
	TotalReadings int                    `json:"total_readings"`
	DateRange     map[string]string      `json:"date_range"`
	PhStats       map[string]float64     `json:"ph_stats"`
	TDSStats      map[string]float64     `json:"tds_stats"`
	TurbidityStats map[string]float64    `json:"turbidity_stats"`
	FlowStats     map[string]float64     `json:"flow_stats"`
	FilterModes   map[string]int         `json:"filter_modes"`
	QualityBreakdown map[string]int      `json:"quality_breakdown"`
}

// calculateSensorStats calculates comprehensive statistics
func calculateSensorStats(readings []models.SensorReading) SensorDataStats {
	if len(readings) == 0 {
		return SensorDataStats{}
	}

	// Initialize stats
	stats := SensorDataStats{
		TotalReadings: len(readings),
		DateRange:     make(map[string]string),
		PhStats:       make(map[string]float64),
		TDSStats:      make(map[string]float64),
		TurbidityStats: make(map[string]float64),
		FlowStats:     make(map[string]float64),
		FilterModes:   make(map[string]int),
		QualityBreakdown: make(map[string]int),
	}

	// Initialize values with first reading
	minPh, maxPh := readings[0].Ph, readings[0].Ph
	minTDS, maxTDS := readings[0].TDS, readings[0].TDS
	minTurbidity, maxTurbidity := readings[0].Turbidity, readings[0].Turbidity
	minFlow, maxFlow := readings[0].Flow, readings[0].Flow
	sumPh, sumTDS, sumTurbidity, sumFlow := 0.0, 0.0, 0.0, 0.0

	earliestTime := readings[0].Timestamp
	latestTime := readings[0].Timestamp

	// Calculate stats
	for _, reading := range readings {
		// Date range
		if reading.Timestamp.Before(earliestTime) {
			earliestTime = reading.Timestamp
		}
		if reading.Timestamp.After(latestTime) {
			latestTime = reading.Timestamp
		}

		// pH stats
		if reading.Ph < minPh {
			minPh = reading.Ph
		}
		if reading.Ph > maxPh {
			maxPh = reading.Ph
		}
		sumPh += reading.Ph

		// TDS stats
		if reading.TDS < minTDS {
			minTDS = reading.TDS
		}
		if reading.TDS > maxTDS {
			maxTDS = reading.TDS
		}
		sumTDS += reading.TDS

		// Turbidity stats
		if reading.Turbidity < minTurbidity {
			minTurbidity = reading.Turbidity
		}
		if reading.Turbidity > maxTurbidity {
			maxTurbidity = reading.Turbidity
		}
		sumTurbidity += reading.Turbidity

		// Flow stats
		if reading.Flow < minFlow {
			minFlow = reading.Flow
		}
		if reading.Flow > maxFlow {
			maxFlow = reading.Flow
		}
		sumFlow += reading.Flow

		// Filter modes count
		stats.FilterModes[string(reading.FilterMode)]++

		// Quality breakdown
		quality := reading.ToWaterQualityStatus().OverallQuality
		stats.QualityBreakdown[quality]++
	}

	count := float64(len(readings))

	// Fill stats
	stats.DateRange["earliest"] = earliestTime.Format("2006-01-02 15:04:05")
	stats.DateRange["latest"] = latestTime.Format("2006-01-02 15:04:05")

	stats.PhStats["min"] = minPh
	stats.PhStats["max"] = maxPh
	stats.PhStats["average"] = sumPh / count

	stats.TDSStats["min"] = minTDS
	stats.TDSStats["max"] = maxTDS
	stats.TDSStats["average"] = sumTDS / count

	stats.TurbidityStats["min"] = minTurbidity
	stats.TurbidityStats["max"] = maxTurbidity
	stats.TurbidityStats["average"] = sumTurbidity / count

	stats.FlowStats["min"] = minFlow
	stats.FlowStats["max"] = maxFlow
	stats.FlowStats["average"] = sumFlow / count

	return stats
}

// GetBestDailyValues returns the best pH, TDS, and Turbidity values for today
func (h *Handlers) GetBestDailyValues(w http.ResponseWriter, r *http.Request) {
	// Get today's date
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Get all readings for today
	readings := h.store.GetReadingsInRange(startOfDay, endOfDay)

	var bestValues BestDailyValues

	if len(readings) == 0 {
		// Return default values instead of 404 error when no data exists
		bestValues = BestDailyValues{
			Date:            now.Format("2006-01-02"),
			FilterMode:      "drinking_water",
			BestPH:          7.0,
			BestTDS:         400,
			BestTurbidity:   0.5,
			BestFlow:        2.5,
			TotalReadings:   0,
			OverallQuality:  "No Data",
			Summary:         "No sensor data available for today.",
		}
	} else {
		// Calculate best values from actual readings
		bestValues = calculateBestValues(readings)
	}

	response := APIResponse{
		Success: true,
		Data:    bestValues,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// BestDailyValues represents the best values for a day
type BestDailyValues struct {
	Date          string  `json:"date"`
	BestPH        float64 `json:"best_ph"`
	BestTDS       float64 `json:"best_tds"`
	BestTurbidity float64 `json:"best_turbidity"`
	TotalReadings int     `json:"total_readings"`
}

// calculateBestValues calculates the best pH, TDS, and Turbidity values from readings
func calculateBestValues(readings []models.SensorReading) BestDailyValues {
	if len(readings) == 0 {
		return BestDailyValues{}
	}

	// For pH: ideal range is 6.5-8.5, so best is closest to 7.0
	// For TDS: lower is better for drinking water (ideal < 300 ppm)
	// For Turbidity: lower is better (ideal < 1 NTU)

	bestPH := readings[0].Ph
	bestTDS := readings[0].TDS
	bestTurbidity := readings[0].Turbidity

	for _, reading := range readings {
		// Best pH is closest to 7.0
		if abs(reading.Ph-7.0) < abs(bestPH-7.0) {
			bestPH = reading.Ph
		}

		// Best TDS is lowest
		if reading.TDS < bestTDS {
			bestTDS = reading.TDS
		}

		// Best Turbidity is lowest
		if reading.Turbidity < bestTurbidity {
			bestTurbidity = reading.Turbidity
		}
	}

	return BestDailyValues{
		Date:          time.Now().Format("2006-01-02"),
		BestPH:        bestPH,
		BestTDS:       bestTDS,
		BestTurbidity: bestTurbidity,
		TotalReadings: len(readings),
	}
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetWorstDailyValues returns the worst pH, TDS, and Turbidity values for today
func (h *Handlers) GetWorstDailyValues(w http.ResponseWriter, r *http.Request) {
	// Get today's date
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Get all readings for today
	readings := h.store.GetReadingsInRange(startOfDay, endOfDay)

	var worstValues WorstDailyValues

	if len(readings) == 0 {
		// Return default values instead of 404 error when no data exists
		worstValues = WorstDailyValues{
			Date:          now.Format("2006-01-02"),
			WorstPH:       7.0,
			WorstTDS:      400,
			WorstTurbidity: 0.5,
			TotalReadings: 0,
		}
	} else {
		// Calculate worst values from actual readings
		worstValues = calculateWorstValues(readings)
	}

	response := APIResponse{
		Success: true,
		Data:    worstValues,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// WorstDailyValues represents the worst values for a day
type WorstDailyValues struct {
	Date           string  `json:"date"`
	WorstPH        float64 `json:"worst_ph"`
	WorstTDS       float64 `json:"worst_tds"`
	WorstTurbidity float64 `json:"worst_turbidity"`
	TotalReadings  int     `json:"total_readings"`
}

// calculateWorstValues calculates the worst pH, TDS, and Turbidity values from readings
func calculateWorstValues(readings []models.SensorReading) WorstDailyValues {
	if len(readings) == 0 {
		return WorstDailyValues{}
	}

	// For pH: worst is farthest from 7.0 (ideal)
	// For TDS: higher is worse for drinking water
	// For Turbidity: higher is worse (more cloudy)

	worstPH := readings[0].Ph
	worstTDS := readings[0].TDS
	worstTurbidity := readings[0].Turbidity

	for _, reading := range readings {
		// Worst pH is farthest from 7.0
		if abs(reading.Ph-7.0) > abs(worstPH-7.0) {
			worstPH = reading.Ph
		}

		// Worst TDS is highest
		if reading.TDS > worstTDS {
			worstTDS = reading.TDS
		}

		// Worst Turbidity is highest
		if reading.Turbidity > worstTurbidity {
			worstTurbidity = reading.Turbidity
		}
	}

	return WorstDailyValues{
		Date:           time.Now().Format("2006-01-02"),
		WorstPH:        worstPH,
		WorstTDS:       worstTDS,
		WorstTurbidity: worstTurbidity,
		TotalReadings:  len(readings),
	}
}

// ===== Schedule Management Handlers =====

// CreateSchedule handles POST /api/v1/schedules
func (h *Handlers) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var request models.CreateScheduleRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := request.Validate(); err != nil {
		h.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Normalize data
	normalizedTime := models.NormalizeTimeFormat(request.StartTime)
	normalizedDays := models.NormalizeDaysOfWeek(request.DaysOfWeek)

	// Create schedule object
	schedule := &models.FilterSchedule{
		Name:            request.Name,
		FilterMode:      request.FilterMode,
		StartTime:       normalizedTime,
		DurationMinutes: request.DurationMinutes,
		DaysOfWeek:      normalizedDays,
		IsActive:        request.IsActive,
	}

	// Save to database
	if err := h.store.CreateSchedule(schedule); err != nil {
		h.sendErrorResponse(w, "Failed to create schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate next execution
	nextExecution := schedule.CalculateNextExecution()

	response := APIResponse{
		Success: true,
		Message: "Schedule created successfully",
		Data: map[string]interface{}{
			"schedule":       schedule,
			"next_execution": nextExecution,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetAllSchedules handles GET /api/v1/schedules
func (h *Handlers) GetAllSchedules(w http.ResponseWriter, r *http.Request) {
	// Check query parameter for active_only filter
	activeOnlyStr := r.URL.Query().Get("active_only")
	activeOnly := activeOnlyStr == "true"

	schedules, err := h.store.GetAllSchedules(activeOnly)
	if err != nil {
		h.sendErrorResponse(w, "Failed to get schedules: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Add next execution time for each schedule
	schedulesWithNext := make([]map[string]interface{}, len(schedules))
	for i, schedule := range schedules {
		nextExecution := schedule.CalculateNextExecution()
		schedulesWithNext[i] = map[string]interface{}{
			"schedule":       schedule,
			"next_execution": nextExecution,
		}
	}

	response := APIResponse{
		Success: true,
		Data:    schedulesWithNext,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSchedule handles GET /api/v1/schedules/{id}
func (h *Handlers) GetSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendErrorResponse(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	schedule, err := h.store.GetSchedule(id)
	if err != nil {
		h.sendErrorResponse(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get recent executions
	executions, _ := h.store.GetScheduleExecutions(id, 5)

	// Calculate next execution
	nextExecution := schedule.CalculateNextExecution()

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"schedule":         schedule,
			"next_execution":   nextExecution,
			"recent_executions": executions,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateSchedule handles PUT /api/v1/schedules/{id}
func (h *Handlers) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendErrorResponse(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	// Get existing schedule
	existing, err := h.store.GetSchedule(id)
	if err != nil {
		h.sendErrorResponse(w, "Schedule not found", http.StatusNotFound)
		return
	}

	var request models.UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := request.Validate(); err != nil {
		h.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update fields if provided
	if request.Name != nil {
		existing.Name = *request.Name
	}
	if request.FilterMode != nil {
		existing.FilterMode = *request.FilterMode
	}
	if request.StartTime != nil {
		existing.StartTime = models.NormalizeTimeFormat(*request.StartTime)
	}
	if request.DurationMinutes != nil {
		existing.DurationMinutes = *request.DurationMinutes
	}
	if request.DaysOfWeek != nil {
		existing.DaysOfWeek = models.NormalizeDaysOfWeek(request.DaysOfWeek)
	}
	if request.IsActive != nil {
		existing.IsActive = *request.IsActive
	}

	// Save updated schedule
	if err := h.store.UpdateSchedule(existing); err != nil {
		h.sendErrorResponse(w, "Failed to update schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success: true,
		Message: "Schedule updated successfully",
		Data:    existing,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteSchedule handles DELETE /api/v1/schedules/{id}
func (h *Handlers) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendErrorResponse(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteSchedule(id); err != nil {
		h.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success: true,
		Message: "Schedule deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ToggleSchedule handles POST /api/v1/schedules/{id}/toggle
func (h *Handlers) ToggleSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendErrorResponse(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	var request struct {
		IsActive bool `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.store.ToggleSchedule(id, request.IsActive); err != nil {
		h.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := "disabled"
	if request.IsActive {
		status = "enabled"
	}

	response := APIResponse{
		Success: true,
		Message: fmt.Sprintf("Schedule %s successfully", status),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetScheduleExecutionHistory handles GET /api/v1/schedules/executions
func (h *Handlers) GetScheduleExecutionHistory(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	scheduleIDStr := r.URL.Query().Get("schedule_id")
	var executions []models.ScheduleExecution
	var err error

	if scheduleIDStr != "" {
		// Get executions for specific schedule
		scheduleID, err := strconv.Atoi(scheduleIDStr)
		if err != nil {
			h.sendErrorResponse(w, "Invalid schedule_id", http.StatusBadRequest)
			return
		}
		executions, err = h.store.GetScheduleExecutions(scheduleID, limit)
	} else {
		// Get all executions
		executions, err = h.store.GetAllScheduleExecutions(limit)
	}

	if err != nil {
		h.sendErrorResponse(w, "Failed to get execution history: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success: true,
		Data:    executions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
