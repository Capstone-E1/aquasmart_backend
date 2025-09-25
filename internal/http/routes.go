package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/Capstone-E1/aquasmart_backend/internal/mqtt"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
	"github.com/Capstone-E1/aquasmart_backend/internal/ws"
)

// SetupRoutes configures all HTTP routes for the water purification API
func SetupRoutes(dataStore store.DataStore, wsHub *ws.Hub, mqttClient *mqtt.Client) *chi.Mux {
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

	// Create handlers
	handlers := NewHandlers(dataStore, mqttClient)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", handlers.HealthCheck)
		r.Get("/stats", handlers.GetSystemStats)

		// Sensor data routes
		r.Route("/sensors", func(r chi.Router) {
			// Latest readings
			r.Get("/latest", handlers.GetLatestReadings)

			// Recent readings with optional filtering
			r.Get("/recent", handlers.GetRecentReadings)

			// Historical data in time range
			r.Get("/history", handlers.GetReadingsInRange)

			// Water quality status
			r.Get("/quality", handlers.GetWaterQualityStatus)
		})

		// Device management routes
		r.Route("/devices", func(r chi.Router) {
			r.Get("/", handlers.GetActiveDevices)
		})

		// Command routes for filter control
		r.Route("/commands", func(r chi.Router) {
			r.Post("/filter", handlers.SetFilterMode)
			r.Get("/filter/status", handlers.GetFilterStatus)
		})

		// Filtration process monitoring routes
		r.Route("/filtration", func(r chi.Router) {
			r.Get("/status", handlers.GetFiltrationStatus)
		})
	})

	// WebSocket route for real-time updates
	r.HandleFunc("/ws", wsHub.HandleWebSocket)

	return r
}