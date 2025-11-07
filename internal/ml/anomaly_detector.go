package ml

import (
	"fmt"
	"math"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

// AnomalyDetector provides anomaly detection capabilities for sensor readings
type AnomalyDetector struct {
	zScoreThreshold float64 // Number of standard deviations for anomaly
	spikeMultiplier float64 // Multiplier for spike detection
}

// NewAnomalyDetector creates a new anomaly detector with default thresholds
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		zScoreThreshold: 3.0, // 3 sigma rule (99.7% confidence)
		spikeMultiplier: 2.5, // Spike if value is 2.5x normal range
	}
}

// DetectAnomalies analyzes a sensor reading against baseline and detects anomalies
func (ad *AnomalyDetector) DetectAnomalies(reading *models.SensorReading, baseline *models.SensorBaseline) []models.AnomalyDetection {
	anomalies := []models.AnomalyDetection{}

	if baseline == nil || baseline.SampleSize < 10 {
		// Not enough baseline data
		return anomalies
	}

	now := time.Now()

	// Check Flow anomalies
	if flowAnomaly := ad.checkMetricAnomaly(
		"flow",
		reading.Flow,
		baseline.FlowMean,
		baseline.FlowStdDev,
		baseline.FlowMin,
		baseline.FlowMax,
		reading,
	); flowAnomaly != nil {
		flowAnomaly.DetectedAt = now
		anomalies = append(anomalies, *flowAnomaly)
	}

	// Check pH anomalies
	if phAnomaly := ad.checkMetricAnomaly(
		"ph",
		reading.Ph,
		baseline.PhMean,
		baseline.PhStdDev,
		baseline.PhMin,
		baseline.PhMax,
		reading,
	); phAnomaly != nil {
		phAnomaly.DetectedAt = now
		anomalies = append(anomalies, *phAnomaly)
	}

	// Check Turbidity anomalies
	if turbAnomaly := ad.checkMetricAnomaly(
		"turbidity",
		reading.Turbidity,
		baseline.TurbidityMean,
		baseline.TurbidityStdDev,
		baseline.TurbidityMin,
		baseline.TurbidityMax,
		reading,
	); turbAnomaly != nil {
		turbAnomaly.DetectedAt = now
		anomalies = append(anomalies, *turbAnomaly)
	}

	// Check TDS anomalies
	if tdsAnomaly := ad.checkMetricAnomaly(
		"tds",
		reading.TDS,
		baseline.TDSMean,
		baseline.TDSStdDev,
		baseline.TDSMin,
		baseline.TDSMax,
		reading,
	); tdsAnomaly != nil {
		tdsAnomaly.DetectedAt = now
		anomalies = append(anomalies, *tdsAnomaly)
	}

	return anomalies
}

// checkMetricAnomaly checks a single metric for anomalies
func (ad *AnomalyDetector) checkMetricAnomaly(
	metricName string,
	actualValue float64,
	mean float64,
	stdDev float64,
	min float64,
	max float64,
	reading *models.SensorReading,
) *models.AnomalyDetection {

	// Calculate z-score
	zScore := 0.0
	if stdDev > 0 {
		zScore = (actualValue - mean) / stdDev
	}

	// Detect different types of anomalies
	var anomalyType string
	var severity string
	var description string
	deviation := math.Abs((actualValue - mean) / mean * 100)

	// 1. Check for sudden spikes (value way above normal)
	if actualValue > mean+ad.spikeMultiplier*stdDev && math.Abs(zScore) > ad.zScoreThreshold {
		anomalyType = "spike"
		severity = ad.calculateSeverity(math.Abs(zScore))
		description = fmt.Sprintf("%s spike detected: %.2f (expected ~%.2f)", metricName, actualValue, mean)

	// 2. Check for sudden drops (value way below normal)
	} else if actualValue < mean-ad.spikeMultiplier*stdDev && math.Abs(zScore) > ad.zScoreThreshold {
		anomalyType = "sudden_drop"
		severity = ad.calculateSeverity(math.Abs(zScore))
		description = fmt.Sprintf("%s sudden drop detected: %.2f (expected ~%.2f)", metricName, actualValue, mean)

	// 3. Check for general outliers (outside normal range)
	} else if math.Abs(zScore) > ad.zScoreThreshold {
		anomalyType = "outlier"
		severity = ad.calculateSeverity(math.Abs(zScore))
		description = fmt.Sprintf("%s outlier detected: %.2f (expected ~%.2f)", metricName, actualValue, mean)

	// 4. Check for sensor failures (impossible values)
	} else if ad.isPossibleSensorFailure(metricName, actualValue) {
		anomalyType = "sensor_failure"
		severity = "critical"
		description = fmt.Sprintf("%s sensor failure suspected: value %.2f is outside possible range", metricName, actualValue)

	} else {
		// No anomaly detected
		return nil
	}

	return &models.AnomalyDetection{
		DeviceID:       reading.DeviceID,
		AnomalyType:    anomalyType,
		Severity:       severity,
		AffectedMetric: metricName,
		ExpectedValue:  mean,
		ActualValue:    actualValue,
		Deviation:      deviation,
		FilterMode:     reading.FilterMode,
		Description:    description,
		AlertSent:      false,
		AutoResolved:   false,
		CreatedAt:      time.Now(),
	}
}

// calculateSeverity determines severity based on z-score
func (ad *AnomalyDetector) calculateSeverity(absZScore float64) string {
	switch {
	case absZScore >= 6.0:
		return "critical" // Extremely rare (beyond 6 sigma)
	case absZScore >= 4.5:
		return "high" // Very rare (beyond 4.5 sigma)
	case absZScore >= 3.5:
		return "medium" // Rare (beyond 3.5 sigma)
	default:
		return "low" // Uncommon but possible
	}
}

// isPossibleSensorFailure checks if value indicates sensor failure
func (ad *AnomalyDetector) isPossibleSensorFailure(metricName string, value float64) bool {
	switch metricName {
	case "flow":
		// Flow should be 0-50 L/min (extreme upper bound)
		return value < 0 || value > 50
	case "ph":
		// pH should be 0-14
		return value < 0 || value > 14
	case "turbidity":
		// Turbidity should be 0-1000 NTU
		return value < 0 || value > 1000
	case "tds":
		// TDS should be 0-1000 PPM
		return value < 0 || value > 1000
	default:
		return false
	}
}

// CalculateBaseline computes statistical baseline from historical readings
func (ad *AnomalyDetector) CalculateBaseline(readings []models.SensorReading, deviceID string, filterMode models.FilterMode) *models.SensorBaseline {
	if len(readings) < 10 {
		return nil // Need at least 10 samples for meaningful statistics
	}

	// Filter readings for specific device and mode
	var filteredReadings []models.SensorReading
	for _, r := range readings {
		if r.DeviceID == deviceID && r.FilterMode == filterMode {
			filteredReadings = append(filteredReadings, r)
		}
	}

	if len(filteredReadings) < 10 {
		return nil
	}

	baseline := &models.SensorBaseline{
		DeviceID:     deviceID,
		FilterMode:   filterMode,
		SampleSize:   len(filteredReadings),
		CalculatedAt: time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Calculate statistics for each metric
	baseline.FlowMean, baseline.FlowStdDev, baseline.FlowMin, baseline.FlowMax = ad.calculateStats(filteredReadings, "flow")
	baseline.PhMean, baseline.PhStdDev, baseline.PhMin, baseline.PhMax = ad.calculateStats(filteredReadings, "ph")
	baseline.TurbidityMean, baseline.TurbidityStdDev, baseline.TurbidityMin, baseline.TurbidityMax = ad.calculateStats(filteredReadings, "turbidity")
	baseline.TDSMean, baseline.TDSStdDev, baseline.TDSMin, baseline.TDSMax = ad.calculateStats(filteredReadings, "tds")

	return baseline
}

// calculateStats calculates mean, std dev, min, max for a metric
func (ad *AnomalyDetector) calculateStats(readings []models.SensorReading, metric string) (mean, stdDev, min, max float64) {
	if len(readings) == 0 {
		return 0, 0, 0, 0
	}

	// Extract values
	var values []float64
	for _, r := range readings {
		switch metric {
		case "flow":
			values = append(values, r.Flow)
		case "ph":
			values = append(values, r.Ph)
		case "turbidity":
			values = append(values, r.Turbidity)
		case "tds":
			values = append(values, r.TDS)
		}
	}

	// Calculate mean
	sum := 0.0
	min = values[0]
	max = values[0]

	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	mean = sum / float64(len(values))

	// Calculate standard deviation
	varianceSum := 0.0
	for _, v := range values {
		diff := v - mean
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(len(values))
	stdDev = math.Sqrt(variance)

	return mean, stdDev, min, max
}

// DetectDrift checks for gradual sensor drift over time
func (ad *AnomalyDetector) DetectDrift(recentReadings []models.SensorReading, baseline *models.SensorBaseline) []models.AnomalyDetection {
	anomalies := []models.AnomalyDetection{}

	if len(recentReadings) < 5 || baseline == nil {
		return anomalies
	}

	// Calculate recent baseline from last readings
	recentBaseline := ad.CalculateBaseline(recentReadings, baseline.DeviceID, baseline.FilterMode)
	if recentBaseline == nil {
		return anomalies
	}

	// Check if recent mean has drifted significantly from historical baseline
	driftThreshold := 15.0 // 15% drift is concerning

	metrics := []struct {
		name         string
		recentMean   float64
		baselineMean float64
	}{
		{"flow", recentBaseline.FlowMean, baseline.FlowMean},
		{"ph", recentBaseline.PhMean, baseline.PhMean},
		{"turbidity", recentBaseline.TurbidityMean, baseline.TurbidityMean},
		{"tds", recentBaseline.TDSMean, baseline.TDSMean},
	}

	for _, m := range metrics {
		if m.baselineMean == 0 {
			continue
		}

		driftPercent := math.Abs((m.recentMean - m.baselineMean) / m.baselineMean * 100)

		if driftPercent > driftThreshold {
			severity := "medium"
			if driftPercent > 30 {
				severity = "high"
			}

			anomaly := models.AnomalyDetection{
				DeviceID:       baseline.DeviceID,
				DetectedAt:     time.Now(),
				AnomalyType:    "drift",
				Severity:       severity,
				AffectedMetric: m.name,
				ExpectedValue:  m.baselineMean,
				ActualValue:    m.recentMean,
				Deviation:      driftPercent,
				FilterMode:     baseline.FilterMode,
				Description:    fmt.Sprintf("%s sensor drift detected: %.1f%% change from baseline", m.name, driftPercent),
				AlertSent:      false,
				AutoResolved:   false,
				CreatedAt:      time.Now(),
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}
