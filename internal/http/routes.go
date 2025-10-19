package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
	"github.com/Capstone-E1/aquasmart_backend/internal/ws"
)

// SetupRoutes configures all HTTP routes for the water purification API
func SetupRoutes(dataStore store.DataStore, wsHub *ws.Hub) *chi.Mux {
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

	// Create handlers (HTTP-only, no MQTT)
	handlers := NewHandlers(dataStore)

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

			// STM32 specific endpoints
			r.Post("/stm32", handlers.AddSTM32SensorData)          // POST data from STM32
			r.Get("/stm32/command", handlers.GetSTM32Command)      // GET commands for STM32
			r.Get("/stm32/mode", handlers.GetSTM32FilterModeSimple) // Simple text-only filter mode
			r.Get("/stm32/led", handlers.GetSTM32LEDStatus)        // Simple LED status: ON or OFF
		})

		// Command routes for filter control
		r.Route("/commands", func(r chi.Router) {
			r.Post("/filter", handlers.SetFilterMode)
		})

		// Control routes for LED and other devices
		r.Route("/control", func(r chi.Router) {
			r.Post("/led", handlers.ControlLED)              // POST to control LED (on/off)
			r.Get("/led/command", handlers.GetLEDCommand)    // GET LED command for STM32 polling
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