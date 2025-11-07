package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

// ML-related data structures for in-memory storage
type mlStore struct {
	anomalies      []models.AnomalyDetection
	baselines      map[string]*models.SensorBaseline // key: "deviceID:filterMode"
	filterHealth   []models.FilterHealth
	predictions    []models.MLPrediction
	nextAnomalyID  int
	nextHealthID   int
	nextPredID     int
	mu             sync.RWMutex
}

func newMLStore() *mlStore {
	return &mlStore{
		anomalies:    []models.AnomalyDetection{},
		baselines:    make(map[string]*models.SensorBaseline),
		filterHealth: []models.FilterHealth{},
		predictions:  []models.MLPrediction{},
		nextAnomalyID: 1,
		nextHealthID: 1,
		nextPredID:   1,
	}
}

// ML: Anomaly Detection Methods

func (s *Store) SaveAnomaly(anomaly *models.AnomalyDetection) error {
	s.mlData.mu.Lock()
	defer s.mlData.mu.Unlock()

	anomaly.ID = s.mlData.nextAnomalyID
	s.mlData.nextAnomalyID++
	anomaly.CreatedAt = time.Now()

	s.mlData.anomalies = append(s.mlData.anomalies, *anomaly)
	return nil
}

func (s *Store) GetAnomalies(limit int) ([]models.AnomalyDetection, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	// Return most recent anomalies
	start := len(s.mlData.anomalies) - limit
	if start < 0 {
		start = 0
	}

	result := make([]models.AnomalyDetection, len(s.mlData.anomalies[start:]))
	copy(result, s.mlData.anomalies[start:])

	// Reverse to get newest first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

func (s *Store) GetAnomaliesByDevice(deviceID string, limit int) ([]models.AnomalyDetection, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	var result []models.AnomalyDetection
	for i := len(s.mlData.anomalies) - 1; i >= 0 && len(result) < limit; i-- {
		if s.mlData.anomalies[i].DeviceID == deviceID {
			result = append(result, s.mlData.anomalies[i])
		}
	}

	return result, nil
}

func (s *Store) GetAnomaliesBySeverity(severity string, limit int) ([]models.AnomalyDetection, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	var result []models.AnomalyDetection
	for i := len(s.mlData.anomalies) - 1; i >= 0 && len(result) < limit; i-- {
		if s.mlData.anomalies[i].Severity == severity {
			result = append(result, s.mlData.anomalies[i])
		}
	}

	return result, nil
}

func (s *Store) GetUnresolvedAnomalies() ([]models.AnomalyDetection, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	var result []models.AnomalyDetection
	for _, a := range s.mlData.anomalies {
		if a.ResolvedAt == nil && !a.IsFalsePositive {
			result = append(result, a)
		}
	}

	return result, nil
}

func (s *Store) ResolveAnomaly(id int) error {
	s.mlData.mu.Lock()
	defer s.mlData.mu.Unlock()

	for i := range s.mlData.anomalies {
		if s.mlData.anomalies[i].ID == id {
			now := time.Now()
			s.mlData.anomalies[i].ResolvedAt = &now
			return nil
		}
	}

	return fmt.Errorf("anomaly not found")
}

func (s *Store) MarkAnomalyFalsePositive(id int) error {
	s.mlData.mu.Lock()
	defer s.mlData.mu.Unlock()

	for i := range s.mlData.anomalies {
		if s.mlData.anomalies[i].ID == id {
			now := time.Now()
			s.mlData.anomalies[i].IsFalsePositive = true
			s.mlData.anomalies[i].ResolvedAt = &now
			return nil
		}
	}

	return fmt.Errorf("anomaly not found")
}

func (s *Store) GetAnomalyStats() (*models.AnomalyStats, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	stats := &models.AnomalyStats{
		BySeverity: make(map[string]int),
		ByType:     make(map[string]int),
		ByMetric:   make(map[string]int),
	}

	stats.TotalAnomalies = len(s.mlData.anomalies)

	now := time.Now()
	deviceCounts := make(map[string]int)
	fpCount := 0

	for _, a := range s.mlData.anomalies {
		// Last 24 hours
		if now.Sub(a.DetectedAt).Hours() < 24 {
			stats.Last24Hours++
		}

		// Last 7 days
		if now.Sub(a.DetectedAt).Hours() < 168 {
			stats.Last7Days++
		}

		// By severity
		stats.BySeverity[a.Severity]++

		// By type
		stats.ByType[a.AnomalyType]++

		// By metric
		stats.ByMetric[a.AffectedMetric]++

		// Device counts
		deviceCounts[a.DeviceID]++

		// False positives
		if a.IsFalsePositive {
			fpCount++
		}
	}

	// False positive rate
	if stats.TotalAnomalies > 0 {
		stats.FalsePositiveRate = float64(fpCount) / float64(stats.TotalAnomalies) * 100
	}

	// Most affected device
	maxCount := 0
	for device, count := range deviceCounts {
		if count > maxCount {
			maxCount = count
			stats.MostAffectedDevice = device
		}
	}

	return stats, nil
}

// ML: Sensor Baseline Methods

func (s *Store) SaveBaseline(baseline *models.SensorBaseline) error {
	s.mlData.mu.Lock()
	defer s.mlData.mu.Unlock()

	key := fmt.Sprintf("%s:%s", baseline.DeviceID, baseline.FilterMode)
	s.mlData.baselines[key] = baseline
	return nil
}

func (s *Store) GetBaseline(deviceID string, filterMode models.FilterMode) (*models.SensorBaseline, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", deviceID, filterMode)
	baseline, exists := s.mlData.baselines[key]
	if !exists {
		return nil, nil
	}

	return baseline, nil
}

func (s *Store) GetAllBaselines() ([]models.SensorBaseline, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	result := make([]models.SensorBaseline, 0, len(s.mlData.baselines))
	for _, baseline := range s.mlData.baselines {
		result = append(result, *baseline)
	}

	return result, nil
}

func (s *Store) UpdateBaseline(baseline *models.SensorBaseline) error {
	baseline.UpdatedAt = time.Now()
	return s.SaveBaseline(baseline)
}

// ML: Filter Health Methods

func (s *Store) SaveFilterHealth(health *models.FilterHealth) error {
	s.mlData.mu.Lock()
	defer s.mlData.mu.Unlock()

	health.ID = s.mlData.nextHealthID
	s.mlData.nextHealthID++
	health.CreatedAt = time.Now()
	health.UpdatedAt = time.Now()

	s.mlData.filterHealth = append(s.mlData.filterHealth, *health)
	return nil
}

func (s *Store) GetLatestFilterHealth(deviceID string) (*models.FilterHealth, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	// Search from the end (most recent)
	for i := len(s.mlData.filterHealth) - 1; i >= 0; i-- {
		if s.mlData.filterHealth[i].DeviceID == deviceID {
			health := s.mlData.filterHealth[i]
			return &health, nil
		}
	}

	return nil, nil
}

func (s *Store) GetFilterHealthHistory(deviceID string, limit int) ([]models.FilterHealth, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	var result []models.FilterHealth
	for i := len(s.mlData.filterHealth) - 1; i >= 0 && len(result) < limit; i-- {
		if s.mlData.filterHealth[i].DeviceID == deviceID {
			result = append(result, s.mlData.filterHealth[i])
		}
	}

	return result, nil
}

func (s *Store) GetAllFilterHealth() ([]models.FilterHealth, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	result := make([]models.FilterHealth, len(s.mlData.filterHealth))
	copy(result, s.mlData.filterHealth)

	return result, nil
}

// ML: Prediction Methods

func (s *Store) SavePrediction(prediction *models.MLPrediction) error {
	s.mlData.mu.Lock()
	defer s.mlData.mu.Unlock()

	prediction.ID = s.mlData.nextPredID
	s.mlData.nextPredID++
	prediction.CreatedAt = time.Now()

	s.mlData.predictions = append(s.mlData.predictions, *prediction)
	return nil
}

func (s *Store) GetPredictions(predictionType string, limit int) ([]models.MLPrediction, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	var result []models.MLPrediction
	for i := len(s.mlData.predictions) - 1; i >= 0 && len(result) < limit; i-- {
		if s.mlData.predictions[i].PredictionType == predictionType {
			result = append(result, s.mlData.predictions[i])
		}
	}

	return result, nil
}

func (s *Store) GetPredictionsByDevice(deviceID string, limit int) ([]models.MLPrediction, error) {
	s.mlData.mu.RLock()
	defer s.mlData.mu.RUnlock()

	var result []models.MLPrediction
	for i := len(s.mlData.predictions) - 1; i >= 0 && len(result) < limit; i-- {
		if s.mlData.predictions[i].DeviceID == deviceID {
			result = append(result, s.mlData.predictions[i])
		}
	}

	return result, nil
}
