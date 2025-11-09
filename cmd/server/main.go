package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/Capstone-E1/aquasmart_backend/config"
	"github.com/Capstone-E1/aquasmart_backend/internal/database"
	httphandlers "github.com/Capstone-E1/aquasmart_backend/internal/http"
	"github.com/Capstone-E1/aquasmart_backend/internal/ml"
	"github.com/Capstone-E1/aquasmart_backend/internal/mqtt"
	"github.com/Capstone-E1/aquasmart_backend/internal/services"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
	"github.com/Capstone-E1/aquasmart_backend/internal/ws"
)

func main() {
	log.Println("🌊 Starting AquaSmart Water Purification IoT Backend...")

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️  Warning: No .env file found: %v", err)
	} else {
		log.Println("✅ Loaded .env file")
	}

	// Load configuration
	cfg := config.Load()
	log.Printf("📋 Loaded configuration: Server port=%s, DB host=%s", 
		cfg.Server.Port, cfg.Database.Host)

	// Initialize data store with Aiven database or fallback to in-memory
	var dataStore store.DataStore
	
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to connect to database: %v", err)
		log.Println("📱 Falling back to in-memory storage")
		// Fallback to in-memory store
		dataStore = store.NewStore(1000)
		log.Println("💾 Initialized in-memory data store")
	} else {
		log.Println("✅ Connected to Aiven PostgreSQL database")
		
		// Run migrations from migrations/ directory
		if err := database.RunMigrations(db.DB); err != nil {
			log.Fatalf("❌ Failed to run migrations: %v", err)
		}
		
		// Use database store
		dataStore = database.NewDatabaseStore(db.DB)
		log.Println("💾 Initialized database data store with Aiven PostgreSQL")
	}

	// Initialize WebSocket hub
	wsHub := ws.NewHub()
	go wsHub.Run()
	log.Println("🔌 Started WebSocket hub")

	// Initialize MQTT client (skip if no broker URL configured)
	var mqttClient *mqtt.Client
	if cfg.MQTT.BrokerURL != "" && cfg.MQTT.BrokerURL != "tcp://localhost:1883" {
		log.Println("📡 Attempting to connect to MQTT broker...")
		mqttTopics := map[string]string{
			"sensor_data":    cfg.MQTT.TopicSensorData,
			"filter_command": cfg.MQTT.TopicFilterCommand,
		}
		
		client, err := mqtt.NewClient(
			cfg.MQTT.BrokerURL,
			cfg.MQTT.ClientID,
			cfg.MQTT.Username,
			cfg.MQTT.Password,
			dataStore,
			mqttTopics,
		)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to connect to MQTT broker: %v", err)
			log.Println("📡 Continuing without MQTT support")
			mqttClient = nil
		} else {
			log.Printf("📡 MQTT client connected - Broker: %s", cfg.MQTT.BrokerURL)
			mqttClient = client
			defer mqttClient.Disconnect()
		}
	} else {
		log.Println("📡 MQTT broker not configured, skipping MQTT initialization")
		mqttClient = nil
	}

	// Initialize and start scheduler
	scheduler := services.NewScheduler(dataStore)
	scheduler.Start()
	log.Println("🕐 Started automated filter mode scheduler")

	// Initialize ML service
	mlService := ml.NewMLService(dataStore)
	mlService.Start()
	defer mlService.Stop()
	log.Println("🤖 ML service initialized and started")

	// Setup HTTP routes with scheduler, MQTT and ML support
	router := httphandlers.SetupRoutes(dataStore, wsHub, scheduler, mqttClient, mlService)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("🚀 Starting HTTP server on port %s", cfg.Server.Port)
		log.Println("📡 API endpoints available:")
		log.Println("  GET /api/v1/stats - System statistics")
		log.Println("  GET /api/v1/sensors/latest - Latest readings from all devices")
		log.Println("  GET /api/v1/sensors/recent?limit=50 - Recent readings")
		log.Println("  GET /api/v1/sensors/all - All sensor data with pagination")
		log.Println("  GET /api/v1/sensors/all/simple - All sensor data simple format")
		log.Println("  GET /api/v1/sensors/stats - Sensor data statistics")
		log.Println("  GET /api/v1/sensors/history - Historical data in time range")
		log.Println("  GET /api/v1/sensors/quality - Water quality status")
		log.Println("  GET /api/v1/sensors/best-daily - Best daily values for today")
		log.Println("  GET /api/v1/sensors/worst-daily - Worst daily values for today")
		log.Println("  POST /api/v1/sensors/data - Add sensor data (testing)")
		log.Println("  POST /api/v1/commands/filter - Set filter mode")
		log.Println("  GET /api/v1/schedules - List all schedules")
		log.Println("  POST /api/v1/schedules - Create new schedule")
		log.Println("  GET /api/v1/schedules/{id} - Get schedule details")
		log.Println("  PUT /api/v1/schedules/{id} - Update schedule")
		log.Println("  DELETE /api/v1/schedules/{id} - Delete schedule")
		log.Println("  POST /api/v1/schedules/{id}/toggle - Enable/disable schedule")
		log.Println("  GET /api/v1/schedules/executions - Schedule execution history")
		log.Println("  GET /api/v1/export/history.xlsx - Export history to Excel")
		log.Println("  GET /api/v1/export/history.csv - Export history to CSV")
		log.Println("  WS /ws - WebSocket for real-time updates")
		log.Printf("🌐 Server running at http://localhost:%s", cfg.Server.Port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ HTTP server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Stop scheduler
	scheduler.Stop()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server shutdown complete")
}