package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

// ML: Anomaly Detection Methods

// SaveAnomaly stores an anomaly detection in the database
func (s *DatabaseStore) SaveAnomaly(anomaly *models.AnomalyDetection) error {
	query := `
		INSERT INTO anomaly_detections (
			device_id, detected_at, anomaly_type, severity, affected_metric,
			expected_value, actual_value, deviation, filter_mode, description,
			is_false_positive, alert_sent, auto_resolved
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at`

	err := s.db.QueryRow(
		query,
		anomaly.DeviceID,
		anomaly.DetectedAt,
		anomaly.AnomalyType,
		anomaly.Severity,
		anomaly.AffectedMetric,
		anomaly.ExpectedValue,
		anomaly.ActualValue,
		anomaly.Deviation,
		anomaly.FilterMode,
		anomaly.Description,
		anomaly.IsFalsePositive,
		anomaly.AlertSent,
		anomaly.AutoResolved,
	).Scan(&anomaly.ID, &anomaly.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to save anomaly: %w", err)
	}

	log.Printf("✅ Saved anomaly: %s - %s (severity: %s)", anomaly.DeviceID, anomaly.AnomalyType, anomaly.Severity)
	return nil
}

// GetAnomalies retrieves recent anomalies
func (s *DatabaseStore) GetAnomalies(limit int) ([]models.AnomalyDetection, error) {
	query := `
		SELECT id, device_id, detected_at, anomaly_type, severity, affected_metric,
			   expected_value, actual_value, deviation, filter_mode, description,
			   is_false_positive, resolved_at, alert_sent, auto_resolved, created_at
		FROM anomaly_detections
		ORDER BY detected_at DESC
		LIMIT $1`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query anomalies: %w", err)
	}
	defer rows.Close()

	return s.scanAnomalies(rows)
}

// GetAnomaliesByDevice retrieves anomalies for a specific device
func (s *DatabaseStore) GetAnomaliesByDevice(deviceID string, limit int) ([]models.AnomalyDetection, error) {
	query := `
		SELECT id, device_id, detected_at, anomaly_type, severity, affected_metric,
			   expected_value, actual_value, deviation, filter_mode, description,
			   is_false_positive, resolved_at, alert_sent, auto_resolved, created_at
		FROM anomaly_detections
		WHERE device_id = $1
		ORDER BY detected_at DESC
		LIMIT $2`

	rows, err := s.db.Query(query, deviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query anomalies by device: %w", err)
	}
	defer rows.Close()

	return s.scanAnomalies(rows)
}

// GetAnomaliesBySeverity retrieves anomalies by severity level
func (s *DatabaseStore) GetAnomaliesBySeverity(severity string, limit int) ([]models.AnomalyDetection, error) {
	query := `
		SELECT id, device_id, detected_at, anomaly_type, severity, affected_metric,
			   expected_value, actual_value, deviation, filter_mode, description,
			   is_false_positive, resolved_at, alert_sent, auto_resolved, created_at
		FROM anomaly_detections
		WHERE severity = $1
		ORDER BY detected_at DESC
		LIMIT $2`

	rows, err := s.db.Query(query, severity, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query anomalies by severity: %w", err)
	}
	defer rows.Close()

	return s.scanAnomalies(rows)
}

// GetUnresolvedAnomalies retrieves all unresolved anomalies
func (s *DatabaseStore) GetUnresolvedAnomalies() ([]models.AnomalyDetection, error) {
	query := `
		SELECT id, device_id, detected_at, anomaly_type, severity, affected_metric,
			   expected_value, actual_value, deviation, filter_mode, description,
			   is_false_positive, resolved_at, alert_sent, auto_resolved, created_at
		FROM anomaly_detections
		WHERE resolved_at IS NULL AND is_false_positive = false
		ORDER BY detected_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unresolved anomalies: %w", err)
	}
	defer rows.Close()

	return s.scanAnomalies(rows)
}

// ResolveAnomaly marks an anomaly as resolved
func (s *DatabaseStore) ResolveAnomaly(id int) error {
	query := `UPDATE anomaly_detections SET resolved_at = NOW() WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to resolve anomaly: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("anomaly not found")
	}

	return nil
}

// MarkAnomalyFalsePositive marks an anomaly as a false positive
func (s *DatabaseStore) MarkAnomalyFalsePositive(id int) error {
	query := `UPDATE anomaly_detections SET is_false_positive = true, resolved_at = NOW() WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to mark anomaly as false positive: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("anomaly not found")
	}

	return nil
}

// GetAnomalyStats calculates anomaly statistics
func (s *DatabaseStore) GetAnomalyStats() (*models.AnomalyStats, error) {
	stats := &models.AnomalyStats{
		BySeverity: make(map[string]int),
		ByType:     make(map[string]int),
		ByMetric:   make(map[string]int),
	}

	// Total anomalies
	err := s.db.QueryRow("SELECT COUNT(*) FROM anomaly_detections").Scan(&stats.TotalAnomalies)
	if err != nil {
		return nil, fmt.Errorf("failed to get total anomalies: %w", err)
	}

	// Last 24 hours
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM anomaly_detections
		WHERE detected_at > NOW() - INTERVAL '24 hours'
	`).Scan(&stats.Last24Hours)
	if err != nil {
		return nil, err
	}

	// Last 7 days
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM anomaly_detections
		WHERE detected_at > NOW() - INTERVAL '7 days'
	`).Scan(&stats.Last7Days)
	if err != nil {
		return nil, err
	}

	// By severity
	rows, err := s.db.Query("SELECT severity, COUNT(*) FROM anomaly_detections GROUP BY severity")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err == nil {
			stats.BySeverity[severity] = count
		}
	}

	// By type
	rows, err = s.db.Query("SELECT anomaly_type, COUNT(*) FROM anomaly_detections GROUP BY anomaly_type")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var aType string
		var count int
		if err := rows.Scan(&aType, &count); err == nil {
			stats.ByType[aType] = count
		}
	}

	// By metric
	rows, err = s.db.Query("SELECT affected_metric, COUNT(*) FROM anomaly_detections GROUP BY affected_metric")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var metric string
		var count int
		if err := rows.Scan(&metric, &count); err == nil {
			stats.ByMetric[metric] = count
		}
	}

	// False positive rate
	var totalCount, fpCount int
	s.db.QueryRow("SELECT COUNT(*) FROM anomaly_detections").Scan(&totalCount)
	s.db.QueryRow("SELECT COUNT(*) FROM anomaly_detections WHERE is_false_positive = true").Scan(&fpCount)
	if totalCount > 0 {
		stats.FalsePositiveRate = float64(fpCount) / float64(totalCount) * 100
	}

	// Most affected device
	err = s.db.QueryRow(`
		SELECT device_id FROM anomaly_detections
		GROUP BY device_id
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`).Scan(&stats.MostAffectedDevice)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return stats, nil
}

// scanAnomalies is a helper to scan anomaly rows
func (s *DatabaseStore) scanAnomalies(rows *sql.Rows) ([]models.AnomalyDetection, error) {
	var anomalies []models.AnomalyDetection

	for rows.Next() {
		var a models.AnomalyDetection
		err := rows.Scan(
			&a.ID,
			&a.DeviceID,
			&a.DetectedAt,
			&a.AnomalyType,
			&a.Severity,
			&a.AffectedMetric,
			&a.ExpectedValue,
			&a.ActualValue,
			&a.Deviation,
			&a.FilterMode,
			&a.Description,
			&a.IsFalsePositive,
			&a.ResolvedAt,
			&a.AlertSent,
			&a.AutoResolved,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan anomaly: %w", err)
		}
		anomalies = append(anomalies, a)
	}

	return anomalies, nil
}

// ML: Sensor Baseline Methods

// SaveBaseline stores a sensor baseline
func (s *DatabaseStore) SaveBaseline(baseline *models.SensorBaseline) error {
	query := `
		INSERT INTO sensor_baselines (
			device_id, filter_mode, flow_mean, flow_std_dev, flow_min, flow_max,
			ph_mean, ph_std_dev, ph_min, ph_max,
			turbidity_mean, turbidity_std_dev, turbidity_min, turbidity_max,
			tds_mean, tds_std_dev, tds_min, tds_max,
			sample_size, calculated_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		ON CONFLICT (device_id, filter_mode) DO UPDATE SET
			flow_mean = EXCLUDED.flow_mean,
			flow_std_dev = EXCLUDED.flow_std_dev,
			flow_min = EXCLUDED.flow_min,
			flow_max = EXCLUDED.flow_max,
			ph_mean = EXCLUDED.ph_mean,
			ph_std_dev = EXCLUDED.ph_std_dev,
			ph_min = EXCLUDED.ph_min,
			ph_max = EXCLUDED.ph_max,
			turbidity_mean = EXCLUDED.turbidity_mean,
			turbidity_std_dev = EXCLUDED.turbidity_std_dev,
			turbidity_min = EXCLUDED.turbidity_min,
			turbidity_max = EXCLUDED.turbidity_max,
			tds_mean = EXCLUDED.tds_mean,
			tds_std_dev = EXCLUDED.tds_std_dev,
			tds_min = EXCLUDED.tds_min,
			tds_max = EXCLUDED.tds_max,
			sample_size = EXCLUDED.sample_size,
			updated_at = EXCLUDED.updated_at
		RETURNING id`

	err := s.db.QueryRow(
		query,
		baseline.DeviceID,
		baseline.FilterMode,
		baseline.FlowMean, baseline.FlowStdDev, baseline.FlowMin, baseline.FlowMax,
		baseline.PhMean, baseline.PhStdDev, baseline.PhMin, baseline.PhMax,
		baseline.TurbidityMean, baseline.TurbidityStdDev, baseline.TurbidityMin, baseline.TurbidityMax,
		baseline.TDSMean, baseline.TDSStdDev, baseline.TDSMin, baseline.TDSMax,
		baseline.SampleSize, baseline.CalculatedAt, baseline.UpdatedAt,
	).Scan(&baseline.DeviceID)

	if err != nil {
		return fmt.Errorf("failed to save baseline: %w", err)
	}

	log.Printf("✅ Saved baseline for %s in %s mode", baseline.DeviceID, baseline.FilterMode)
	return nil
}

// GetBaseline retrieves baseline for device and filter mode
func (s *DatabaseStore) GetBaseline(deviceID string, filterMode models.FilterMode) (*models.SensorBaseline, error) {
	query := `
		SELECT device_id, filter_mode, flow_mean, flow_std_dev, flow_min, flow_max,
			   ph_mean, ph_std_dev, ph_min, ph_max,
			   turbidity_mean, turbidity_std_dev, turbidity_min, turbidity_max,
			   tds_mean, tds_std_dev, tds_min, tds_max,
			   sample_size, calculated_at, updated_at
		FROM sensor_baselines
		WHERE device_id = $1 AND filter_mode = $2`

	var baseline models.SensorBaseline
	err := s.db.QueryRow(query, deviceID, filterMode).Scan(
		&baseline.DeviceID,
		&baseline.FilterMode,
		&baseline.FlowMean, &baseline.FlowStdDev, &baseline.FlowMin, &baseline.FlowMax,
		&baseline.PhMean, &baseline.PhStdDev, &baseline.PhMin, &baseline.PhMax,
		&baseline.TurbidityMean, &baseline.TurbidityStdDev, &baseline.TurbidityMin, &baseline.TurbidityMax,
		&baseline.TDSMean, &baseline.TDSStdDev, &baseline.TDSMin, &baseline.TDSMax,
		&baseline.SampleSize, &baseline.CalculatedAt, &baseline.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get baseline: %w", err)
	}

	return &baseline, nil
}

// GetAllBaselines retrieves all baselines
func (s *DatabaseStore) GetAllBaselines() ([]models.SensorBaseline, error) {
	query := `
		SELECT device_id, filter_mode, flow_mean, flow_std_dev, flow_min, flow_max,
			   ph_mean, ph_std_dev, ph_min, ph_max,
			   turbidity_mean, turbidity_std_dev, turbidity_min, turbidity_max,
			   tds_mean, tds_std_dev, tds_min, tds_max,
			   sample_size, calculated_at, updated_at
		FROM sensor_baselines
		ORDER BY updated_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query baselines: %w", err)
	}
	defer rows.Close()

	var baselines []models.SensorBaseline
	for rows.Next() {
		var b models.SensorBaseline
		err := rows.Scan(
			&b.DeviceID, &b.FilterMode,
			&b.FlowMean, &b.FlowStdDev, &b.FlowMin, &b.FlowMax,
			&b.PhMean, &b.PhStdDev, &b.PhMin, &b.PhMax,
			&b.TurbidityMean, &b.TurbidityStdDev, &b.TurbidityMin, &b.TurbidityMax,
			&b.TDSMean, &b.TDSStdDev, &b.TDSMin, &b.TDSMax,
			&b.SampleSize, &b.CalculatedAt, &b.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan baseline: %w", err)
		}
		baselines = append(baselines, b)
	}

	return baselines, nil
}

// UpdateBaseline updates an existing baseline
func (s *DatabaseStore) UpdateBaseline(baseline *models.SensorBaseline) error {
	baseline.UpdatedAt = time.Now()
	return s.SaveBaseline(baseline) // Uses ON CONFLICT DO UPDATE
}

// ML: Filter Health Methods

// SaveFilterHealth stores filter health assessment
func (s *DatabaseStore) SaveFilterHealth(health *models.FilterHealth) error {
	// Convert recommendations slice to JSON
	recsJSON, err := json.Marshal(health.Recommendations)
	if err != nil {
		return fmt.Errorf("failed to marshal recommendations: %w", err)
	}

	query := `
		INSERT INTO filter_health (
			device_id, filter_mode, health_score, predicted_days_remaining,
			estimated_replacement, current_efficiency, average_efficiency, efficiency_trend,
			turbidity_reduction, tds_reduction, ph_stabilization,
			maintenance_required, replacement_urgent, recommendations,
			last_calculated, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id`

	err = s.db.QueryRow(
		query,
		health.DeviceID, health.FilterMode, health.HealthScore, health.PredictedDaysRemaining,
		health.EstimatedReplacement, health.CurrentEfficiency, health.AverageEfficiency, health.EfficiencyTrend,
		health.TurbidityReduction, health.TDSReduction, health.PhStabilization,
		health.MaintenanceRequired, health.ReplacementUrgent, recsJSON,
		health.LastCalculated, health.CreatedAt, health.UpdatedAt,
	).Scan(&health.ID)

	if err != nil {
		return fmt.Errorf("failed to save filter health: %w", err)
	}

	log.Printf("✅ Saved filter health: Score %.1f, Days remaining: %d", health.HealthScore, health.PredictedDaysRemaining)
	return nil
}

// GetLatestFilterHealth retrieves the most recent filter health for a device
func (s *DatabaseStore) GetLatestFilterHealth(deviceID string) (*models.FilterHealth, error) {
	query := `
		SELECT id, device_id, filter_mode, health_score, predicted_days_remaining,
			   estimated_replacement, current_efficiency, average_efficiency, efficiency_trend,
			   turbidity_reduction, tds_reduction, ph_stabilization,
			   maintenance_required, replacement_urgent, recommendations,
			   last_calculated, created_at, updated_at
		FROM filter_health
		WHERE device_id = $1
		ORDER BY last_calculated DESC
		LIMIT 1`

	var health models.FilterHealth
	var recsJSON []byte

	err := s.db.QueryRow(query, deviceID).Scan(
		&health.ID, &health.DeviceID, &health.FilterMode, &health.HealthScore, &health.PredictedDaysRemaining,
		&health.EstimatedReplacement, &health.CurrentEfficiency, &health.AverageEfficiency, &health.EfficiencyTrend,
		&health.TurbidityReduction, &health.TDSReduction, &health.PhStabilization,
		&health.MaintenanceRequired, &health.ReplacementUrgent, &recsJSON,
		&health.LastCalculated, &health.CreatedAt, &health.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get filter health: %w", err)
	}

	// Unmarshal recommendations
	if err := json.Unmarshal(recsJSON, &health.Recommendations); err != nil {
		health.Recommendations = []string{}
	}

	return &health, nil
}

// GetFilterHealthHistory retrieves filter health history
func (s *DatabaseStore) GetFilterHealthHistory(deviceID string, limit int) ([]models.FilterHealth, error) {
	query := `
		SELECT id, device_id, filter_mode, health_score, predicted_days_remaining,
			   estimated_replacement, current_efficiency, average_efficiency, efficiency_trend,
			   turbidity_reduction, tds_reduction, ph_stabilization,
			   maintenance_required, replacement_urgent, recommendations,
			   last_calculated, created_at, updated_at
		FROM filter_health
		WHERE device_id = $1
		ORDER BY last_calculated DESC
		LIMIT $2`

	rows, err := s.db.Query(query, deviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query filter health history: %w", err)
	}
	defer rows.Close()

	return s.scanFilterHealth(rows)
}

// GetAllFilterHealth retrieves all filter health records
func (s *DatabaseStore) GetAllFilterHealth() ([]models.FilterHealth, error) {
	query := `
		SELECT id, device_id, filter_mode, health_score, predicted_days_remaining,
			   estimated_replacement, current_efficiency, average_efficiency, efficiency_trend,
			   turbidity_reduction, tds_reduction, ph_stabilization,
			   maintenance_required, replacement_urgent, recommendations,
			   last_calculated, created_at, updated_at
		FROM filter_health
		ORDER BY last_calculated DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all filter health: %w", err)
	}
	defer rows.Close()

	return s.scanFilterHealth(rows)
}

// scanFilterHealth is a helper to scan filter health rows
func (s *DatabaseStore) scanFilterHealth(rows *sql.Rows) ([]models.FilterHealth, error) {
	var healthRecords []models.FilterHealth

	for rows.Next() {
		var h models.FilterHealth
		var recsJSON []byte

		err := rows.Scan(
			&h.ID, &h.DeviceID, &h.FilterMode, &h.HealthScore, &h.PredictedDaysRemaining,
			&h.EstimatedReplacement, &h.CurrentEfficiency, &h.AverageEfficiency, &h.EfficiencyTrend,
			&h.TurbidityReduction, &h.TDSReduction, &h.PhStabilization,
			&h.MaintenanceRequired, &h.ReplacementUrgent, &recsJSON,
			&h.LastCalculated, &h.CreatedAt, &h.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan filter health: %w", err)
		}

		// Unmarshal recommendations
		if err := json.Unmarshal(recsJSON, &h.Recommendations); err != nil {
			h.Recommendations = []string{}
		}

		healthRecords = append(healthRecords, h)
	}

	return healthRecords, nil
}

// ML: Prediction Methods

// SavePrediction stores an ML prediction
func (s *DatabaseStore) SavePrediction(prediction *models.MLPrediction) error {
	query := `
		INSERT INTO ml_predictions (
			prediction_type, device_id, predicted_value, confidence_score,
			predicted_for, actual_value, accuracy, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err := s.db.QueryRow(
		query,
		prediction.PredictionType,
		prediction.DeviceID,
		prediction.PredictedValue,
		prediction.ConfidenceScore,
		prediction.PredictedFor,
		prediction.ActualValue,
		prediction.Accuracy,
		prediction.CreatedAt,
	).Scan(&prediction.ID)

	if err != nil {
		return fmt.Errorf("failed to save prediction: %w", err)
	}

	return nil
}

// GetPredictions retrieves predictions by type
func (s *DatabaseStore) GetPredictions(predictionType string, limit int) ([]models.MLPrediction, error) {
	query := `
		SELECT id, prediction_type, device_id, predicted_value, confidence_score,
			   predicted_for, actual_value, accuracy, created_at
		FROM ml_predictions
		WHERE prediction_type = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := s.db.Query(query, predictionType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query predictions: %w", err)
	}
	defer rows.Close()

	return s.scanPredictions(rows)
}

// GetPredictionsByDevice retrieves predictions for a device
func (s *DatabaseStore) GetPredictionsByDevice(deviceID string, limit int) ([]models.MLPrediction, error) {
	query := `
		SELECT id, prediction_type, device_id, predicted_value, confidence_score,
			   predicted_for, actual_value, accuracy, created_at
		FROM ml_predictions
		WHERE device_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := s.db.Query(query, deviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query predictions by device: %w", err)
	}
	defer rows.Close()

	return s.scanPredictions(rows)
}

// scanPredictions is a helper to scan prediction rows
func (s *DatabaseStore) scanPredictions(rows *sql.Rows) ([]models.MLPrediction, error) {
	var predictions []models.MLPrediction

	for rows.Next() {
		var p models.MLPrediction
		err := rows.Scan(
			&p.ID,
			&p.PredictionType,
			&p.DeviceID,
			&p.PredictedValue,
			&p.ConfidenceScore,
			&p.PredictedFor,
			&p.ActualValue,
			&p.Accuracy,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan prediction: %w", err)
		}
		predictions = append(predictions, p)
	}

	return predictions, nil
}
