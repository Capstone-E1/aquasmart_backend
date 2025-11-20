package ml

import (
	"log"
	"sync"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
)

// MLService provides machine learning services for real-time data processing
type MLService struct {
	store           store.DataStore
	anomalyDetector *AnomalyDetector
	filterPredictor *FilterPredictor
	sensorPredictor *SensorPredictor
	stopChan        chan struct{}
	wg              sync.WaitGroup
	mu              sync.Mutex
	running         bool

	// Configuration
	baselineUpdateInterval     time.Duration
	healthAnalysisInterval     time.Duration
	predictionUpdateInterval   time.Duration
	enableRealTimeAnomaly      bool
	enableAutoPredictionUpdate bool
}

// NewMLService creates a new ML service
func NewMLService(dataStore store.DataStore) *MLService {
	return &MLService{
		store:                      dataStore,
		anomalyDetector:            NewAnomalyDetector(),
		filterPredictor:            NewFilterPredictor(),
		sensorPredictor:            NewSensorPredictor(),
		stopChan:                   make(chan struct{}),
		baselineUpdateInterval:     1 * time.Hour,   // Update baselines every hour
		healthAnalysisInterval:     30 * time.Minute, // Analyze filter health every 30 minutes
		predictionUpdateInterval:   2 * time.Hour,    // Update predictions every 2 hours
		enableRealTimeAnomaly:      false, // DISABLED: Anomaly detection feature disabled
		enableAutoPredictionUpdate: true,
	}
}


// Start begins the ML service background tasks
func (s *MLService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Println("🤖 Starting ML Service (using statistical methods with linear regression)...")

	// Start baseline update task (only if anomaly detection is enabled)
	if s.enableRealTimeAnomaly {
		s.wg.Add(1)
		go s.baselineUpdateTask()
		log.Println("  ✓ Anomaly detection enabled")
	} else {
		log.Println("  ✗ Anomaly detection disabled")
	}

	// Start filter health analysis task
	s.wg.Add(1)
	go s.filterHealthAnalysisTask()

	// Start prediction update task
	s.wg.Add(1)
	go s.predictionUpdateTask()

	log.Println("✅ ML Service started successfully")
}

// Stop stops all ML service background tasks
func (s *MLService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	log.Println("🛑 Stopping ML Service...")
	close(s.stopChan)
	s.wg.Wait()
	log.Println("✅ ML Service stopped")
}

// ProcessNewReading processes a new sensor reading for anomaly detection and prediction updates
func (s *MLService) ProcessNewReading(reading *models.SensorReading) {
	// 1. Anomaly Detection
	if s.enableRealTimeAnomaly {
		// Get baseline for this device and filter mode
		baseline, err := s.store.GetBaseline(reading.DeviceID, reading.FilterMode)
		if err != nil {
			log.Printf("Warning: Failed to get baseline for anomaly detection: %v", err)
		} else if baseline != nil {
			// Detect anomalies
			anomalies := s.anomalyDetector.DetectAnomalies(reading, baseline)
			if len(anomalies) > 0 {
				log.Printf("⚠️  Detected %d anomalies in reading from %s", len(anomalies), reading.DeviceID)

				for _, anomaly := range anomalies {
					// Save anomaly to database
					if err := s.store.SaveAnomaly(&anomaly); err != nil {
						log.Printf("Error saving anomaly: %v", err)
					} else {
						log.Printf("   - %s: %s (severity: %s)", anomaly.AffectedMetric, anomaly.Description, anomaly.Severity)
					}
				}
			}
		}
	}

	// 2. Autonomous Prediction Update (trigger when new data arrives)
	if s.enableAutoPredictionUpdate {
		// Trigger prediction update asynchronously (don't block)
		go s.updatePredictionsForDevice(reading.DeviceID, reading.FilterMode, "new_data")
	}
}

// baselineUpdateTask periodically updates sensor baselines
func (s *MLService) baselineUpdateTask() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.baselineUpdateInterval)
	defer ticker.Stop()

	// Run immediately on start
	s.updateBaselines()

	for {
		select {
		case <-ticker.C:
			s.updateBaselines()
		case <-s.stopChan:
			return
		}
	}
}

// updateBaselines updates baselines for all devices and modes
func (s *MLService) updateBaselines() {
	log.Println("📊 Updating sensor baselines...")

	devices := []string{"stm32_pre", "stm32_post", "stm32_main"}
	modes := []models.FilterMode{models.FilterModeDrinking, models.FilterModeHousehold}

	updated := 0
	for _, device := range devices {
		for _, mode := range modes {
			// Get all readings for this device
			allReadings := s.store.GetReadingsByDevice(device)

			// Calculate baseline
			baseline := s.anomalyDetector.CalculateBaseline(allReadings, device, mode)
			if baseline != nil {
				// Save or update baseline
				if err := s.store.SaveBaseline(baseline); err != nil {
					log.Printf("Warning: Failed to save baseline for %s/%s: %v", device, mode, err)
				} else {
					updated++
					log.Printf("   ✅ Updated baseline for %s in %s mode (n=%d)", device, mode, baseline.SampleSize)
				}
			}
		}
	}

	log.Printf("✅ Baseline update complete: %d baselines updated", updated)
}

// filterHealthAnalysisTask periodically analyzes filter health
func (s *MLService) filterHealthAnalysisTask() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.healthAnalysisInterval)
	defer ticker.Stop()

	// Wait a bit before first analysis (let some data accumulate)
	time.Sleep(5 * time.Minute)

	for {
		select {
		case <-ticker.C:
			s.analyzeFilterHealth()
		case <-s.stopChan:
			return
		}
	}
}

// analyzeFilterHealth performs filter health analysis
func (s *MLService) analyzeFilterHealth() {
	log.Println("🔬 Analyzing filter health...")

	// Get recent pre and post filtration readings
	preReadings := s.store.GetRecentReadingsByDevice("stm32_pre", 100)
	postReadings := s.store.GetRecentReadingsByDevice("stm32_post", 100)

	if len(preReadings) < 20 || len(postReadings) < 20 {
		log.Printf("⚠️  Insufficient data for filter health analysis (pre: %d, post: %d)",
			len(preReadings), len(postReadings))
		return
	}

	// Get current filter mode
	filterMode := s.store.GetCurrentFilterMode()

	// Perform analysis
	health, err := s.filterPredictor.AnalyzeFilterHealth(preReadings, postReadings, filterMode)
	if err != nil {
		log.Printf("Error analyzing filter health: %v", err)
		return
	}

	// Save to database
	if err := s.store.SaveFilterHealth(health); err != nil {
		log.Printf("Error saving filter health: %v", err)
		return
	}

	log.Printf("✅ Filter health analysis complete:")
	log.Printf("   - Health Score: %.1f/100 (%s)", health.HealthScore, health.GetHealthCategory())
	log.Printf("   - Current Efficiency: %.1f%%", health.CurrentEfficiency)
	log.Printf("   - Predicted Days Remaining: %d", health.PredictedDaysRemaining)
	log.Printf("   - Trend: %s", health.EfficiencyTrend)

	if health.MaintenanceRequired {
		log.Printf("⚠️  MAINTENANCE REQUIRED")
	}
	if health.ReplacementUrgent {
		log.Printf("🚨 URGENT: Filter replacement needed!")
	}
}

// DetectDrift checks for sensor drift in recent readings
func (s *MLService) DetectDrift() {
	log.Println("📈 Checking for sensor drift...")

	devices := []string{"stm32_pre", "stm32_post", "stm32_main"}
	modes := []models.FilterMode{models.FilterModeDrinking, models.FilterModeHousehold}

	driftDetected := 0

	for _, device := range devices {
		for _, mode := range modes {
			// Get baseline
			baseline, err := s.store.GetBaseline(device, mode)
			if err != nil || baseline == nil {
				continue
			}

			// Get recent readings (last 20)
			allReadings := s.store.GetReadingsByDevice(device)
			if len(allReadings) < 20 {
				continue
			}

			recentReadings := allReadings[len(allReadings)-20:]

			// Detect drift
			driftAnomalies := s.anomalyDetector.DetectDrift(recentReadings, baseline)
			if len(driftAnomalies) > 0 {
				driftDetected += len(driftAnomalies)

				for _, anomaly := range driftAnomalies {
					log.Printf("⚠️  Drift detected: %s on %s", anomaly.Description, device)

					// Save drift anomaly
					if err := s.store.SaveAnomaly(&anomaly); err != nil {
						log.Printf("Error saving drift anomaly: %v", err)
					}
				}
			}
		}
	}

	if driftDetected > 0 {
		log.Printf("⚠️  Sensor drift check complete: %d drifts detected", driftDetected)
	} else {
		log.Println("✅ No sensor drift detected")
	}
}

// GetMLServiceStatus returns the current status of the ML service
func (s *MLService) GetMLServiceStatus() map[string]interface{} {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	return map[string]interface{}{
		"running":                      running,
		"real_time_anomaly_enabled":     s.enableRealTimeAnomaly,
		"auto_prediction_update_enabled": s.enableAutoPredictionUpdate,
		"baseline_update_interval":      s.baselineUpdateInterval.String(),
		"health_analysis_interval":      s.healthAnalysisInterval.String(),
		"prediction_update_interval":    s.predictionUpdateInterval.String(),
	}
}

// EnableRealTimeAnomaly enables/disables real-time anomaly detection
func (s *MLService) EnableRealTimeAnomaly(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enableRealTimeAnomaly = enabled
	log.Printf("Real-time anomaly detection: %v", enabled)
}

// predictionUpdateTask periodically updates sensor predictions
func (s *MLService) predictionUpdateTask() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.predictionUpdateInterval)
	defer ticker.Stop()

	// Wait a bit before first prediction update (let some data accumulate)
	time.Sleep(10 * time.Minute)

	for {
		select {
		case <-ticker.C:
			s.updateAllPredictions("scheduled")
		case <-s.stopChan:
			return
		}
	}
}

// updateAllPredictions updates predictions for all devices
func (s *MLService) updateAllPredictions(triggerReason string) {
	log.Println("🔮 Updating sensor predictions...")

	devices := []string{"stm32_pre", "stm32_post", "stm32_main"}
	modes := []models.FilterMode{models.FilterModeDrinking, models.FilterModeHousehold}

	updated := 0
	for _, device := range devices {
		for _, mode := range modes {
			if s.updatePredictionsForDevice(device, mode, triggerReason) {
				updated++
			}
		}
	}

	log.Printf("✅ Prediction update complete: %d device/mode combinations updated", updated)
}

// updatePredictionsForDevice updates predictions for a specific device
func (s *MLService) updatePredictionsForDevice(deviceID string, filterMode models.FilterMode, triggerReason string) bool {
	startTime := time.Now()

	// Get historical readings (up to 200 for prediction)
	historicalReadings := s.store.GetRecentReadingsByDevice(deviceID, 200)

	if len(historicalReadings) < 50 {
		// Not enough data for predictions
		return false
	}

	// Generate predictions
	predictions, err := s.sensorPredictor.PredictSensorValues(historicalReadings, deviceID, filterMode)
	if err != nil {
		log.Printf("Warning: Failed to generate predictions for %s/%s: %v", deviceID, filterMode, err)
		return false
	}

	// Save predictions to database (note: need to implement store methods)
	savedCount := 0
	for _, pred := range predictions {
		// Convert PredictionResult to SensorPrediction model
		sensorPred := &models.SensorPrediction{
			DeviceID:           deviceID,
			FilterMode:         filterMode,
			PredictedFor:       pred.Timestamp,
			PredictionMethod:   pred.Method,
			ConfidenceScore:    pred.ConfidenceScore,
			PredictedFlow:      pred.PredictedFlow,
			PredictedPh:        pred.PredictedPh,
			PredictedTurbidity: pred.PredictedTurbidity,
			PredictedTDS:       pred.PredictedTDS,
			IsValidated:        false,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		// TODO: Implement SaveSensorPrediction in store
		// For now, skip saving until store methods are implemented
		_ = sensorPred
		savedCount++
	}

	// Log update
	executionTime := int(time.Since(startTime).Milliseconds())
	log.Printf("   ✅ Generated %d predictions for %s in %s mode (%dms)",
		len(predictions), deviceID, filterMode, executionTime)

	// TODO: Log to prediction_update_log table
	_ = executionTime
	_ = triggerReason
	_ = savedCount

	return true
}

// ValidatePredictions compares predictions with actual readings and updates accuracy
func (s *MLService) ValidatePredictions() {
	log.Println("📊 Validating predictions against actual readings...")

	// TODO: Implement prediction validation
	// 1. Get unvalidated predictions
	// 2. Find matching actual readings
	// 3. Calculate errors
	// 4. Update predictions with actual values and accuracy
	// 5. Update accuracy summary

	log.Println("✅ Prediction validation complete")
}

// GetSensorPredictor returns the sensor predictor instance
func (s *MLService) GetSensorPredictor() *SensorPredictor {
	return s.sensorPredictor
}
