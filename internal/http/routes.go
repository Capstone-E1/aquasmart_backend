package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/Capstone-E1/aquasmart_backend/internal/ml"
	"github.com/Capstone-E1/aquasmart_backend/internal/services"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
	"github.com/Capstone-E1/aquasmart_backend/internal/ws"
)

// SetupRoutes configures all HTTP routes for the water purification API
func SetupRoutes(dataStore store.DataStore, wsHub *ws.Hub, scheduler *services.Scheduler, mlService *ml.MLService) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // In production, specify allowed origins
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Create handlers with scheduler and ML service support
	handlers := NewHandlers(dataStore, scheduler, mlService)
	mlHandlers := NewMLHandlers(dataStore, mlService)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// System stats
		r.Get("/stats", handlers.GetSystemStats)

		// Sensor data routes
		r.Route("/sensors", func(r chi.Router) {
			// Latest readings
			r.Get("/latest", handlers.GetLatestReadings)

			// Recent readings with optional filtering
			r.Get("/recent", handlers.GetRecentReadings)

			// Get all sensor data (with pagination and filters)
			r.Get("/all", handlers.GetAllSensorData)

			// Get all sensor data (simple format)
			r.Get("/all/simple", handlers.GetAllSensorDataSimple)

			// Get statistics about all sensor data
			r.Get("/stats", handlers.GetSensorDataStats)

			// Historical data in time range
			r.Get("/history", handlers.GetReadingsInRange)

			// Water quality status
			r.Get("/quality", handlers.GetWaterQualityStatus)

			// Add sensor data manually (for testing)
			r.Post("/data", handlers.AddSensorData)

			// Best daily values for today
			r.Get("/best-daily", handlers.GetBestDailyValues)

			// Worst daily values for today
			r.Get("/worst-daily", handlers.GetWorstDailyValues)

			// Device-specific routes
			r.Get("/devices/latest", handlers.GetAllDevicesLatest)  // Get latest reading for all devices
			r.Get("/devices/{deviceID}", handlers.GetDeviceReadings) // Get all readings for a specific device

			// STM32/ESP32 specific endpoints
			r.Post("/stm32", handlers.AddSTM32SensorData)           // POST data from STM32
			r.Get("/stm32/command", handlers.GetSTM32Command)       // GET commands for STM32
			r.Get("/stm32/mode", handlers.GetSTM32FilterModeSimple) // Simple text-only filter mode
			r.Get("/stm32/led", handlers.GetSTM32LEDStatus)         // GET LED command: ON/OFF (for ESP32 polling)
			r.Post("/stm32/led", handlers.SetLEDCommand)            // POST to set LED command (from Postman/Frontend)
		})
		
		// Command routes for filter control
		r.Route("/commands", func(r chi.Router) {
			r.Post("/filter", handlers.SetFilterMode)
		})

		// Schedule management routes
		r.Route("/schedules", func(r chi.Router) {
			r.Get("/", handlers.GetAllSchedules)                  // List all schedules
			r.Post("/", handlers.CreateSchedule)                  // Create new schedule
			r.Get("/{id}", handlers.GetSchedule)                  // Get specific schedule
			r.Put("/{id}", handlers.UpdateSchedule)               // Update schedule
			r.Delete("/{id}", handlers.DeleteSchedule)            // Delete schedule
			r.Post("/{id}/toggle", handlers.ToggleSchedule)       // Enable/disable schedule
			r.Get("/executions", handlers.GetScheduleExecutionHistory) // Execution history
		})

		// ML Features - Anomaly Detection & Filter Lifespan Prediction
		r.Route("/ml", func(r chi.Router) {
			// Dashboard - Overall ML metrics
			r.Get("/dashboard", mlHandlers.GetMLDashboard)

			// Filter Health & Lifespan Prediction
			r.Get("/filter/health", mlHandlers.GetFilterHealth)
			r.Post("/filter/analyze", mlHandlers.AnalyzeFilterHealth)

			// Anomaly Detection
			r.Get("/anomalies", mlHandlers.GetAnomalies)
			r.Get("/anomalies/unresolved", mlHandlers.GetUnresolvedAnomalies)
			r.Get("/anomalies/stats", mlHandlers.GetAnomalyStats)
			r.Post("/anomalies/detect", mlHandlers.DetectAnomaliesNow)
			r.Post("/anomalies/{id}/resolve", mlHandlers.ResolveAnomaly)
			r.Post("/anomalies/{id}/false-positive", mlHandlers.MarkAnomalyFalsePositive)

			// Baselines for anomaly detection
			r.Get("/baselines", mlHandlers.GetBaselines)
			r.Post("/baselines/calculate", mlHandlers.CalculateBaselines)

			// Sensor Value Predictions (NEW)
			r.Get("/predictions", mlHandlers.GetPredictions)
			r.Post("/predictions/generate", mlHandlers.GeneratePredictions)
			r.Get("/predictions/accuracy", mlHandlers.GetPredictionAccuracy)
			r.Post("/predictions/update", mlHandlers.TriggerPredictionUpdate)
			r.Get("/predictions/status", mlHandlers.GetPredictionStatus)
		})

		// Export routes for data history
		r.Route("/export", func(r chi.Router) {
			r.Get("/history.xlsx", handlers.ExportHistoryExcel)
			r.Get("/history.csv", handlers.ExportHistoryCSV)
		})
	})

	// WebSocket route for real-time updates
	r.HandleFunc("/ws", wsHub.HandleWebSocket)

	return r
}