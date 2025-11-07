package ml

import (
	"fmt"
	"math"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

// SensorPredictor provides time-series prediction for sensor values
type SensorPredictor struct {
	minHistoricalData int     // Minimum readings needed for prediction
	forecastHorizon   int     // How many time steps to predict ahead
	smoothingAlpha    float64 // Exponential smoothing parameter (0-1)
}

// NewSensorPredictor creates a new sensor predictor
func NewSensorPredictor() *SensorPredictor {
	return &SensorPredictor{
		minHistoricalData: 50,  // Need at least 50 historical readings
		forecastHorizon:   24,  // Predict next 24 readings (e.g., 24 hours)
		smoothingAlpha:    0.3, // Weight for exponential smoothing
	}
}

// PredictionResult holds prediction details for a single time point
type PredictionResult struct {
	Timestamp        time.Time
	PredictedFlow    float64
	PredictedPh      float64
	PredictedTurbidity float64
	PredictedTDS     float64
	ConfidenceScore  float64 // 0-1, based on prediction variance
	Method           string  // "exponential_smoothing", "trend_analysis", etc.
}

// PredictSensorValues predicts future sensor values based on historical data
func (sp *SensorPredictor) PredictSensorValues(
	historicalReadings []models.SensorReading,
	deviceID string,
	filterMode models.FilterMode,
) ([]PredictionResult, error) {

	if len(historicalReadings) < sp.minHistoricalData {
		return nil, fmt.Errorf("insufficient historical data: need at least %d readings, got %d",
			sp.minHistoricalData, len(historicalReadings))
	}

	// Filter readings for specific device and mode
	var filteredReadings []models.SensorReading
	for _, r := range historicalReadings {
		if r.DeviceID == deviceID && r.FilterMode == filterMode {
			filteredReadings = append(filteredReadings, r)
		}
	}

	if len(filteredReadings) < sp.minHistoricalData {
		return nil, fmt.Errorf("insufficient data for device %s in mode %s", deviceID, filterMode)
	}

	// Analyze patterns
	patterns := sp.analyzePatterns(filteredReadings)

	// Generate predictions
	predictions := sp.generatePredictions(filteredReadings, patterns)

	return predictions, nil
}

// Pattern holds analyzed patterns from historical data
type Pattern struct {
	FlowTrend        float64 // Per-reading trend
	PhTrend          float64
	TurbidityTrend   float64
	TDSTrend         float64

	FlowMean         float64
	PhMean           float64
	TurbidityMean    float64
	TDSMean          float64

	FlowStdDev       float64
	PhStdDev         float64
	TurbidityStdDev  float64
	TDSStdDev        float64

	HasCyclicPattern bool
	CyclePeriod      int // Number of readings per cycle

	IsStable         bool // Low variance indicates stability
}

// analyzePatterns extracts patterns from historical data
func (sp *SensorPredictor) analyzePatterns(readings []models.SensorReading) Pattern {
	pattern := Pattern{}

	if len(readings) == 0 {
		return pattern
	}

	// Calculate means and standard deviations
	flowVals := make([]float64, len(readings))
	phVals := make([]float64, len(readings))
	turbidityVals := make([]float64, len(readings))
	tdsVals := make([]float64, len(readings))

	for i, r := range readings {
		flowVals[i] = r.Flow
		phVals[i] = r.Ph
		turbidityVals[i] = r.Turbidity
		tdsVals[i] = r.TDS
	}

	pattern.FlowMean, pattern.FlowStdDev = sp.calcMeanStdDev(flowVals)
	pattern.PhMean, pattern.PhStdDev = sp.calcMeanStdDev(phVals)
	pattern.TurbidityMean, pattern.TurbidityStdDev = sp.calcMeanStdDev(turbidityVals)
	pattern.TDSMean, pattern.TDSStdDev = sp.calcMeanStdDev(tdsVals)

	// Calculate trends (linear regression slope)
	pattern.FlowTrend = sp.calculateTrend(flowVals)
	pattern.PhTrend = sp.calculateTrend(phVals)
	pattern.TurbidityTrend = sp.calculateTrend(turbidityVals)
	pattern.TDSTrend = sp.calculateTrend(tdsVals)

	// Detect stability (low variance relative to mean)
	avgStdDev := (pattern.FlowStdDev + pattern.PhStdDev + pattern.TurbidityStdDev + pattern.TDSStdDev) / 4
	avgMean := (pattern.FlowMean + pattern.PhMean + pattern.TurbidityMean + pattern.TDSMean) / 4

	if avgMean > 0 {
		coefficientOfVariation := avgStdDev / avgMean
		pattern.IsStable = coefficientOfVariation < 0.2 // Less than 20% variation
	}

	// Detect cyclic patterns (simplified - check for autocorrelation)
	pattern.HasCyclicPattern = sp.detectCyclicPattern(flowVals)
	if pattern.HasCyclicPattern {
		pattern.CyclePeriod = 24 // Assume daily cycle (24 hours)
	}

	return pattern
}

// generatePredictions creates future predictions based on patterns
func (sp *SensorPredictor) generatePredictions(
	readings []models.SensorReading,
	patterns Pattern,
) []PredictionResult {

	predictions := make([]PredictionResult, sp.forecastHorizon)

	// Get last reading timestamp
	lastReading := readings[len(readings)-1]
	lastTimestamp := lastReading.Timestamp

	// Determine time interval between readings
	timeInterval := sp.estimateTimeInterval(readings)

	// Use exponential smoothing with trend
	flowSmoothed := lastReading.Flow
	phSmoothed := lastReading.Ph
	turbiditySmoothed := lastReading.Turbidity
	tdsSmoothed := lastReading.TDS

	for i := 0; i < sp.forecastHorizon; i++ {
		// Apply exponential smoothing with trend
		flowSmoothed = flowSmoothed + patterns.FlowTrend
		phSmoothed = phSmoothed + patterns.PhTrend
		turbiditySmoothed = turbiditySmoothed + patterns.TurbidityTrend
		tdsSmoothed = tdsSmoothed + patterns.TDSTrend

		// Add mean reversion (values tend to revert to historical mean)
		meanReversionFactor := 0.1 // 10% pull towards mean
		flowSmoothed = flowSmoothed + (patterns.FlowMean-flowSmoothed)*meanReversionFactor
		phSmoothed = phSmoothed + (patterns.PhMean-phSmoothed)*meanReversionFactor
		turbiditySmoothed = turbiditySmoothed + (patterns.TurbidityMean-turbiditySmoothed)*meanReversionFactor
		tdsSmoothed = tdsSmoothed + (patterns.TDSMean-tdsSmoothed)*meanReversionFactor

		// Add cyclic component if detected
		if patterns.HasCyclicPattern {
			cyclicOffset := math.Sin(2 * math.Pi * float64(i) / float64(patterns.CyclePeriod))
			flowSmoothed += cyclicOffset * patterns.FlowStdDev * 0.3
			turbiditySmoothed += cyclicOffset * patterns.TurbidityStdDev * 0.3
		}

		// Calculate confidence (decreases with distance into future)
		confidenceScore := sp.calculateConfidence(i, patterns)

		// Ensure values are within valid ranges
		flowSmoothed = sp.clampValue(flowSmoothed, 0, 50)
		phSmoothed = sp.clampValue(phSmoothed, 0, 14)
		turbiditySmoothed = sp.clampValue(turbiditySmoothed, 0, 100)
		tdsSmoothed = sp.clampValue(tdsSmoothed, 0, 1000)

		predictions[i] = PredictionResult{
			Timestamp:          lastTimestamp.Add(time.Duration(i+1) * timeInterval),
			PredictedFlow:      sp.roundTo2Decimals(flowSmoothed),
			PredictedPh:        sp.roundTo2Decimals(phSmoothed),
			PredictedTurbidity: sp.roundTo2Decimals(turbiditySmoothed),
			PredictedTDS:       sp.roundTo2Decimals(tdsSmoothed),
			ConfidenceScore:    confidenceScore,
			Method:             "exponential_smoothing_with_trend",
		}
	}

	return predictions
}

// CalculateAccuracy compares predictions with actual values
func (sp *SensorPredictor) CalculateAccuracy(
	predictions []PredictionResult,
	actualReadings []models.SensorReading,
) map[string]float64 {

	accuracy := map[string]float64{
		"flow_mae":      0, // Mean Absolute Error
		"ph_mae":        0,
		"turbidity_mae": 0,
		"tds_mae":       0,
		"overall_accuracy": 0,
		"matches_found": 0,
	}

	if len(predictions) == 0 || len(actualReadings) == 0 {
		return accuracy
	}

	matches := 0
	totalFlowError := 0.0
	totalPhError := 0.0
	totalTurbidityError := 0.0
	totalTDSError := 0.0

	// Match predictions with actual readings (within 5 minutes)
	for _, pred := range predictions {
		for _, actual := range actualReadings {
			timeDiff := math.Abs(pred.Timestamp.Sub(actual.Timestamp).Minutes())

			if timeDiff <= 5.0 {
				// Found matching actual reading
				matches++
				totalFlowError += math.Abs(pred.PredictedFlow - actual.Flow)
				totalPhError += math.Abs(pred.PredictedPh - actual.Ph)
				totalTurbidityError += math.Abs(pred.PredictedTurbidity - actual.Turbidity)
				totalTDSError += math.Abs(pred.PredictedTDS - actual.TDS)
				break
			}
		}
	}

	if matches > 0 {
		accuracy["flow_mae"] = totalFlowError / float64(matches)
		accuracy["ph_mae"] = totalPhError / float64(matches)
		accuracy["turbidity_mae"] = totalTurbidityError / float64(matches)
		accuracy["tds_mae"] = totalTDSError / float64(matches)
		accuracy["matches_found"] = float64(matches)

		// Calculate overall accuracy as percentage
		// Lower MAE = higher accuracy
		avgError := (accuracy["flow_mae"] + accuracy["ph_mae"] +
			accuracy["turbidity_mae"] + accuracy["tds_mae"]) / 4

		// Convert error to accuracy percentage (simplified)
		accuracy["overall_accuracy"] = math.Max(0, 100-avgError*10)
	}

	return accuracy
}

// Helper functions

func (sp *SensorPredictor) calcMeanStdDev(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	varianceSum := 0.0
	for _, v := range values {
		diff := v - mean
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(len(values))
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

func (sp *SensorPredictor) calculateTrend(values []float64) float64 {
	n := len(values)
	if n < 2 {
		return 0
	}

	// Simple linear regression slope
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	nFloat := float64(n)
	slope := (nFloat*sumXY - sumX*sumY) / (nFloat*sumX2 - sumX*sumX)

	// Dampen extreme slopes
	if math.Abs(slope) > 0.1 {
		slope = slope * 0.5
	}

	return slope
}

func (sp *SensorPredictor) detectCyclicPattern(values []float64) bool {
	// Simplified autocorrelation check for daily patterns
	// In production, use proper FFT or autocorrelation function

	if len(values) < 48 {
		return false
	}

	// Check if values at 24-hour intervals are correlated
	correlation := 0.0
	count := 0
	period := 24

	for i := 0; i < len(values)-period; i++ {
		correlation += math.Abs(values[i] - values[i+period])
		count++
	}

	avgDiff := correlation / float64(count)
	mean, stdDev := sp.calcMeanStdDev(values)

	// If 24-hour differences are small relative to variance, likely cyclic
	if mean > 0 {
		return avgDiff < stdDev
	}

	return false
}

func (sp *SensorPredictor) estimateTimeInterval(readings []models.SensorReading) time.Duration {
	if len(readings) < 2 {
		return 1 * time.Hour // Default to 1 hour
	}

	// Calculate average time between readings
	totalDuration := time.Duration(0)
	count := 0

	for i := 1; i < len(readings) && i < 10; i++ {
		duration := readings[i].Timestamp.Sub(readings[i-1].Timestamp)
		if duration > 0 && duration < 24*time.Hour {
			totalDuration += duration
			count++
		}
	}

	if count > 0 {
		return totalDuration / time.Duration(count)
	}

	return 1 * time.Hour
}

func (sp *SensorPredictor) calculateConfidence(step int, patterns Pattern) float64 {
	// Confidence decreases exponentially with forecast distance
	baseConfidence := 0.95

	// Decay factor based on stability
	decayRate := 0.05
	if patterns.IsStable {
		decayRate = 0.03 // Slower decay for stable systems
	}

	confidence := baseConfidence * math.Exp(-decayRate*float64(step))

	// Adjust based on data variance
	if !patterns.IsStable {
		confidence *= 0.9
	}

	return math.Max(0.1, math.Min(1.0, confidence))
}

func (sp *SensorPredictor) clampValue(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (sp *SensorPredictor) roundTo2Decimals(value float64) float64 {
	return math.Round(value*100) / 100
}
