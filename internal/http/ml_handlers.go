package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/ml"
	"github.com/Capstone-E1/aquasmart_backend/internal/models"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
	"github.com/go-chi/chi/v5"
)

// MLHandlers provides HTTP handlers for ML features
type MLHandlers struct {
	store            store.DataStore
	anomalyDetector  *ml.AnomalyDetector
	filterPredictor  *ml.FilterPredictor
	sensorPredictor  *ml.SensorPredictor
	mlService        *ml.MLService
}

// NewMLHandlers creates a new ML handlers instance
func NewMLHandlers(dataStore store.DataStore, mlService *ml.MLService) *MLHandlers {
	return &MLHandlers{
		store:           dataStore,
		anomalyDetector: ml.NewAnomalyDetector(),
		filterPredictor: ml.NewFilterPredictor(),
		sensorPredictor: ml.NewSensorPredictor(),
		mlService:       mlService,
	}
}

// GetFilterHealth returns the latest filter health assessment
func (h *MLHandlers) GetFilterHealth(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		deviceID = "filter_system"
	}

	health, err := h.store.GetLatestFilterHealth(deviceID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get filter health", err)
		return
	}

	if health == nil {
		respondWithJSON(w, http.StatusOK, map[string]string{
			"message": "No filter health data available yet. Analysis requires pre and post filtration readings.",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, health)
}

// AnalyzeFilterHealth triggers a new filter health analysis
func (h *MLHandlers) AnalyzeFilterHealth(w http.ResponseWriter, r *http.Request) {
	// Get recent pre and post filtration readings
	preReadings := h.store.GetRecentReadingsByDevice("stm32_pre", 100)
	postReadings := h.store.GetRecentReadingsByDevice("stm32_post", 100)

	if len(preReadings) < 20 || len(postReadings) < 20 {
		respondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Insufficient data for analysis. Need at least 20 pre and post filtration readings.",
			"pre_readings": strconv.Itoa(len(preReadings)),
			"post_readings": strconv.Itoa(len(postReadings)),
		})
		return
	}

	// Get current filter mode
	filterMode := h.store.GetCurrentFilterMode()

	// Perform analysis
	health, err := h.filterPredictor.AnalyzeFilterHealth(preReadings, postReadings, filterMode)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to analyze filter health", err)
		return
	}

	// Save to database
	if err := h.store.SaveFilterHealth(health); err != nil {
		log.Printf("Warning: Failed to save filter health: %v", err)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Filter health analysis completed",
		"health":  health,
	})
}

// GetAnomalies returns detected anomalies
func (h *MLHandlers) GetAnomalies(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	deviceID := r.URL.Query().Get("device_id")
	severity := r.URL.Query().Get("severity")

	var anomalies []models.AnomalyDetection
	var err error

	if deviceID != "" {
		anomalies, err = h.store.GetAnomaliesByDevice(deviceID, limit)
	} else if severity != "" {
		anomalies, err = h.store.GetAnomaliesBySeverity(severity, limit)
	} else {
		anomalies, err = h.store.GetAnomalies(limit)
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get anomalies", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"count":     len(anomalies),
		"anomalies": anomalies,
	})
}

// GetUnresolvedAnomalies returns all unresolved anomalies
func (h *MLHandlers) GetUnresolvedAnomalies(w http.ResponseWriter, r *http.Request) {
	anomalies, err := h.store.GetUnresolvedAnomalies()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get unresolved anomalies", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"count":     len(anomalies),
		"anomalies": anomalies,
	})
}

// ResolveAnomaly marks an anomaly as resolved
func (h *MLHandlers) ResolveAnomaly(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid anomaly ID", err)
		return
	}

	if err := h.store.ResolveAnomaly(id); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to resolve anomaly", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Anomaly marked as resolved",
	})
}

// MarkAnomalyFalsePositive marks an anomaly as false positive
func (h *MLHandlers) MarkAnomalyFalsePositive(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid anomaly ID", err)
		return
	}

	if err := h.store.MarkAnomalyFalsePositive(id); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to mark anomaly as false positive", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Anomaly marked as false positive",
	})
}

// GetAnomalyStats returns anomaly statistics
func (h *MLHandlers) GetAnomalyStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetAnomalyStats()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get anomaly stats", err)
		return
	}

	respondWithJSON(w, http.StatusOK, stats)
}

// CalculateBaselines calculates sensor baselines for anomaly detection
func (h *MLHandlers) CalculateBaselines(w http.ResponseWriter, r *http.Request) {
	devices := []string{"stm32_pre", "stm32_post", "stm32_main"}
	modes := []models.FilterMode{models.FilterModeDrinking, models.FilterModeHousehold}

	baselinesCreated := 0

	for _, device := range devices {
		for _, mode := range modes {
			// Get recent readings for this device/mode combination
			allReadings := h.store.GetReadingsByDevice(device)

			baseline := h.anomalyDetector.CalculateBaseline(allReadings, device, mode)
			if baseline != nil {
				if err := h.store.SaveBaseline(baseline); err != nil {
					log.Printf("Warning: Failed to save baseline for %s/%s: %v", device, mode, err)
				} else {
					baselinesCreated++
				}
			}
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":           "Baseline calculation completed",
		"baselines_created": baselinesCreated,
	})
}

// GetBaselines returns all sensor baselines
func (h *MLHandlers) GetBaselines(w http.ResponseWriter, r *http.Request) {
	baselines, err := h.store.GetAllBaselines()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get baselines", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"count":     len(baselines),
		"baselines": baselines,
	})
}

// DetectAnomaliesNow performs real-time anomaly detection on latest readings
func (h *MLHandlers) DetectAnomaliesNow(w http.ResponseWriter, r *http.Request) {
	devices := []string{"stm32_pre", "stm32_post", "stm32_main"}
	totalAnomalies := 0

	for _, device := range devices {
		// Get latest reading
		reading, exists := h.store.GetLatestReadingByDevice(device)
		if !exists {
			continue
		}

		// Get baseline for this device/mode
		baseline, err := h.store.GetBaseline(device, reading.FilterMode)
		if err != nil {
			log.Printf("Warning: Failed to get baseline for %s: %v", device, err)
			continue
		}

		if baseline == nil {
			log.Printf("No baseline available for %s in %s mode", device, reading.FilterMode)
			continue
		}

		// Detect anomalies
		anomalies := h.anomalyDetector.DetectAnomalies(reading, baseline)
		for _, anomaly := range anomalies {
			if err := h.store.SaveAnomaly(&anomaly); err != nil {
				log.Printf("Warning: Failed to save anomaly: %v", err)
			} else {
				totalAnomalies++
			}
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":          "Anomaly detection completed",
		"anomalies_found":  totalAnomalies,
		"timestamp":        time.Now(),
	})
}

// GetMLDashboard returns a comprehensive ML dashboard with all metrics
func (h *MLHandlers) GetMLDashboard(w http.ResponseWriter, r *http.Request) {
	// Get filter health
	filterHealth, _ := h.store.GetLatestFilterHealth("filter_system")

	// Get unresolved anomalies
	unresolvedAnomalies, _ := h.store.GetUnresolvedAnomalies()

	// Get anomaly stats
	anomalyStats, _ := h.store.GetAnomalyStats()

	// Get recent anomalies
	recentAnomalies, _ := h.store.GetAnomalies(10)

	dashboard := map[string]interface{}{
		"filter_health": filterHealth,
		"anomalies": map[string]interface{}{
			"unresolved_count": len(unresolvedAnomalies),
			"unresolved":       unresolvedAnomalies,
			"recent":           recentAnomalies,
			"stats":            anomalyStats,
		},
		"system_status": map[string]interface{}{
			"ml_features_enabled": true,
			"last_updated":        time.Now(),
		},
	}

	respondWithJSON(w, http.StatusOK, dashboard)
}

// GetPredictions returns sensor value predictions
func (h *MLHandlers) GetPredictions(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		deviceID = "stm32_pre"
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 24 // Default to 24 hours
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// TODO: Implement GetSensorPredictions in store
	// For now, return placeholder
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Prediction retrieval not yet fully implemented",
		"device_id": deviceID,
		"requested_limit": limit,
		"note": "Store methods need to be implemented for database access",
	})
}

// GeneratePredictions triggers new prediction generation
func (h *MLHandlers) GeneratePredictions(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	filterModeStr := r.URL.Query().Get("filter_mode")

	if deviceID == "" {
		deviceID = "stm32_pre"
	}

	var filterMode models.FilterMode
	if filterModeStr == "household_water" {
		filterMode = models.FilterModeHousehold
	} else {
		filterMode = models.FilterModeDrinking
	}

	// Get historical readings
	historicalReadings := h.store.GetRecentReadingsByDevice(deviceID, 200)

	if len(historicalReadings) < 50 {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Insufficient historical data for predictions",
			"device_id": deviceID,
			"readings_available": len(historicalReadings),
			"readings_required": 50,
		})
		return
	}

	// Generate predictions
	startTime := time.Now()
	predictions, err := h.sensorPredictor.PredictSensorValues(historicalReadings, deviceID, filterMode)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate predictions", err)
		return
	}

	executionTime := time.Since(startTime).Milliseconds()

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Predictions generated successfully",
		"device_id": deviceID,
		"filter_mode": filterMode,
		"predictions_count": len(predictions),
		"execution_time_ms": executionTime,
		"predictions": predictions,
		"historical_data_used": len(historicalReadings),
	})
}

// GetPredictionAccuracy returns prediction accuracy metrics
func (h *MLHandlers) GetPredictionAccuracy(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")

	// TODO: Implement GetPredictionAccuracySummary in store
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Prediction accuracy tracking not yet fully implemented",
		"device_id": deviceID,
		"note": "Store methods need to be implemented for accuracy retrieval",
	})
}

// TriggerPredictionUpdate manually triggers prediction update for all devices
func (h *MLHandlers) TriggerPredictionUpdate(w http.ResponseWriter, r *http.Request) {
	log.Println("Manual prediction update triggered via API")

	// Trigger update asynchronously
	go func() {
		devices := []string{"stm32_pre", "stm32_post", "stm32_main"}
		modes := []models.FilterMode{models.FilterModeDrinking, models.FilterModeHousehold}

		updated := 0
		for _, device := range devices {
			for _, mode := range modes {
				historicalReadings := h.store.GetRecentReadingsByDevice(device, 200)
				if len(historicalReadings) >= 50 {
					_, err := h.sensorPredictor.PredictSensorValues(historicalReadings, device, mode)
					if err == nil {
						updated++
					}
				}
			}
		}
		log.Printf("✅ Manual prediction update complete: %d device/mode combinations updated", updated)
	}()

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Prediction update triggered",
		"status": "running_in_background",
		"timestamp": time.Now(),
	})
}

// GetPredictionStatus returns the current status of prediction system
func (h *MLHandlers) GetPredictionStatus(w http.ResponseWriter, r *http.Request) {
	if h.mlService == nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"message": "ML Service not available",
			"prediction_enabled": false,
		})
		return
	}

	status := h.mlService.GetMLServiceStatus()

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"prediction_system_status": status,
		"features": map[string]interface{}{
			"autonomous_updates": "enabled",
			"trigger_on_new_data": "enabled",
			"scheduled_updates": "enabled",
			"forecast_horizon": "24 time periods",
			"prediction_method": "exponential_smoothing_with_trend",
		},
	})
}

// Helper function to respond with JSON
func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// Helper function to respond with error
func respondWithError(w http.ResponseWriter, statusCode int, message string, err error) {
	log.Printf("Error: %s - %v", message, err)
	respondWithJSON(w, statusCode, map[string]string{
		"error":   message,
		"details": err.Error(),
	})
}
