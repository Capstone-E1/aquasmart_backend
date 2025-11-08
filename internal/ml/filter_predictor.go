package ml

import (
	"fmt"
	"math"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

// FilterPredictor provides filter lifespan prediction and health assessment
type FilterPredictor struct {
	minDataPoints        int     // Minimum readings needed for prediction
	degradationThreshold float64 // Efficiency drop threshold for concern
	maxFilterLifeDays    int     // Maximum filter lifespan in days
	maxFilterVolumeLiters float64 // Maximum volume before replacement (liters)
}

// NewFilterPredictor creates a new filter predictor
func NewFilterPredictor() *FilterPredictor {
	return &FilterPredictor{
		minDataPoints:         20,      // Need at least 20 pre/post reading pairs
		degradationThreshold:  10.0,    // 10% efficiency drop is concerning
		maxFilterLifeDays:     180,     // 6 months maximum filter life
		maxFilterVolumeLiters: 100000.0, // 100,000 liters capacity
	}
}

// AnalyzeFilterHealth performs comprehensive filter health analysis
func (fp *FilterPredictor) AnalyzeFilterHealth(
	preReadings []models.SensorReading,
	postReadings []models.SensorReading,
	filterMode models.FilterMode,
) (*models.FilterHealth, error) {

	if len(preReadings) < fp.minDataPoints || len(postReadings) < fp.minDataPoints {
		return nil, fmt.Errorf("insufficient data: need at least %d readings", fp.minDataPoints)
	}

	// Match pre and post readings by timestamp (within 1 minute)
	matchedPairs := fp.matchReadings(preReadings, postReadings)
	if len(matchedPairs) < fp.minDataPoints/2 {
		return nil, fmt.Errorf("insufficient matched pre/post reading pairs")
	}

	// Calculate efficiency metrics
	efficiencies := fp.calculateEfficiencies(matchedPairs)
	currentEfficiency := fp.getRecentAverage(efficiencies, 5)
	averageEfficiency := fp.calculateMean(efficiencies)

	// Detect efficiency trend
	trend := fp.detectTrend(efficiencies)

	// Calculate degradation metrics
	turbidityReduction := fp.calculateAverageReduction(matchedPairs, "turbidity")
	tdsReduction := fp.calculateAverageReduction(matchedPairs, "tds")
	phStabilization := fp.calculatePhStabilization(matchedPairs)

	// Calculate health score (0-100)
	healthScore := fp.calculateHealthScore(
		currentEfficiency,
		averageEfficiency,
		turbidityReduction,
		tdsReduction,
		trend,
	)

	// Calculate additional metrics for enhanced prediction
	totalFlowProcessed := fp.calculateTotalFlowProcessed(preReadings)
	filterAgeDays := fp.calculateFilterAgeDays(preReadings)

	// Predict remaining lifespan using enhanced multi-factor prediction
	daysRemaining := fp.predictRemainingDaysEnhanced(
		efficiencies,
		currentEfficiency,
		trend,
		preReadings,
	)

	// Generate recommendations
	recommendations := fp.generateRecommendations(
		healthScore,
		currentEfficiency,
		daysRemaining,
		trend,
	)

	health := &models.FilterHealth{
		DeviceID:              "filter_system", // Can be made dynamic
		FilterMode:            filterMode,
		HealthScore:           healthScore,
		PredictedDaysRemaining: daysRemaining,
		EstimatedReplacement:  time.Now().AddDate(0, 0, daysRemaining),
		CurrentEfficiency:     currentEfficiency,
		AverageEfficiency:     averageEfficiency,
		EfficiencyTrend:       trend,
		TurbidityReduction:    turbidityReduction,
		TDSReduction:          tdsReduction,
		PhStabilization:       phStabilization,
		TotalFlowProcessed:    totalFlowProcessed,
		FilterAgeDays:         filterAgeDays,
		MaintenanceRequired:   healthScore < 75,
		ReplacementUrgent:     healthScore < 30 || daysRemaining < 7,
		Recommendations:       recommendations,
		LastCalculated:        time.Now(),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	return health, nil
}

// matchReadings matches pre and post filtration readings by timestamp
func (fp *FilterPredictor) matchReadings(preReadings, postReadings []models.SensorReading) []struct {
	pre  models.SensorReading
	post models.SensorReading
} {
	var pairs []struct {
		pre  models.SensorReading
		post models.SensorReading
	}

	// Match readings within 1 minute of each other
	for _, pre := range preReadings {
		for _, post := range postReadings {
			timeDiff := math.Abs(pre.Timestamp.Sub(post.Timestamp).Minutes())
			if timeDiff <= 1.0 {
				pairs = append(pairs, struct {
					pre  models.SensorReading
					post models.SensorReading
				}{pre, post})
				break
			}
		}
	}

	return pairs
}

// calculateEfficiencies calculates filter efficiency for each matched pair
func (fp *FilterPredictor) calculateEfficiencies(pairs []struct {
	pre  models.SensorReading
	post models.SensorReading
}) []float64 {
	efficiencies := make([]float64, len(pairs))

	for i, pair := range pairs {
		efficiencies[i] = models.CalculateFilterEfficiency(&pair.pre, &pair.post)
	}

	return efficiencies
}

// calculateAverageReduction calculates average reduction percentage for a metric
func (fp *FilterPredictor) calculateAverageReduction(pairs []struct {
	pre  models.SensorReading
	post models.SensorReading
}, metric string) float64 {
	if len(pairs) == 0 {
		return 0.0
	}

	totalReduction := 0.0
	count := 0

	for _, pair := range pairs {
		var preValue, postValue float64

		switch metric {
		case "turbidity":
			preValue = pair.pre.Turbidity
			postValue = pair.post.Turbidity
		case "tds":
			preValue = pair.pre.TDS
			postValue = pair.post.TDS
		}

		if preValue > 0 {
			reduction := ((preValue - postValue) / preValue) * 100
			if reduction > 0 { // Only count positive reductions
				totalReduction += reduction
				count++
			}
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalReduction / float64(count)
}

// calculatePhStabilization measures how well pH is stabilized to neutral
func (fp *FilterPredictor) calculatePhStabilization(pairs []struct {
	pre  models.SensorReading
	post models.SensorReading
}) float64 {
	if len(pairs) == 0 {
		return 0.0
	}

	totalImprovement := 0.0
	targetPh := 7.0

	for _, pair := range pairs {
		preDeviation := math.Abs(pair.pre.Ph - targetPh)
		postDeviation := math.Abs(pair.post.Ph - targetPh)

		if preDeviation > 0 {
			improvement := ((preDeviation - postDeviation) / preDeviation) * 100
			totalImprovement += improvement
		}
	}

	return totalImprovement / float64(len(pairs))
}

// detectTrend analyzes efficiency trend over time
func (fp *FilterPredictor) detectTrend(efficiencies []float64) string {
	if len(efficiencies) < 10 {
		return "stable"
	}

	// Split into first half and second half
	mid := len(efficiencies) / 2
	firstHalf := efficiencies[:mid]
	secondHalf := efficiencies[mid:]

	firstAvg := fp.calculateMean(firstHalf)
	secondAvg := fp.calculateMean(secondHalf)

	change := ((secondAvg - firstAvg) / firstAvg) * 100

	switch {
	case change > 5.0:
		return "improving"
	case change < -5.0:
		return "degrading"
	default:
		return "stable"
	}
}

// calculateHealthScore computes overall filter health score (0-100)
func (fp *FilterPredictor) calculateHealthScore(
	currentEfficiency float64,
	averageEfficiency float64,
	turbidityReduction float64,
	tdsReduction float64,
	trend string,
) float64 {

	// Base score from current efficiency
	score := currentEfficiency

	// Adjust based on average performance
	if averageEfficiency > 70 {
		score += 5
	} else if averageEfficiency < 50 {
		score -= 10
	}

	// Adjust based on turbidity reduction
	if turbidityReduction > 80 {
		score += 5
	} else if turbidityReduction < 40 {
		score -= 10
	}

	// Adjust based on TDS reduction
	if tdsReduction > 70 {
		score += 5
	} else if tdsReduction < 30 {
		score -= 10
	}

	// Adjust based on trend
	switch trend {
	case "improving":
		score += 10
	case "degrading":
		score -= 15
	}

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// predictRemainingDays estimates days until filter replacement needed
func (fp *FilterPredictor) predictRemainingDays(
	efficiencies []float64,
	currentEfficiency float64,
	trend string,
) int {

	// Minimum acceptable efficiency before replacement
	minEfficiency := 30.0

	if currentEfficiency < minEfficiency {
		return 0 // Immediate replacement
	}

	// Calculate degradation rate
	degradationRate := fp.calculateDegradationRate(efficiencies)

	if degradationRate <= 0 {
		// No degradation detected, assume 90 days
		return 90
	}

	// Calculate days until efficiency drops below threshold
	efficiencyGap := currentEfficiency - minEfficiency
	daysRemaining := int(efficiencyGap / degradationRate)

	// Adjust based on trend
	switch trend {
	case "degrading":
		daysRemaining = int(float64(daysRemaining) * 0.8) // Reduce by 20%
	case "improving":
		daysRemaining = int(float64(daysRemaining) * 1.2) // Increase by 20%
	}

	// Clamp between 0 and 180 days
	if daysRemaining < 0 {
		daysRemaining = 0
	}
	if daysRemaining > 180 {
		daysRemaining = 180
	}

	return daysRemaining
}

// calculateDegradationRate calculates efficiency loss per day
func (fp *FilterPredictor) calculateDegradationRate(efficiencies []float64) float64 {
	if len(efficiencies) < 5 {
		return 0.0
	}

	// Use linear regression to estimate degradation
	n := len(efficiencies)
	firstQuarter := efficiencies[:n/4]
	lastQuarter := efficiencies[3*n/4:]

	firstAvg := fp.calculateMean(firstQuarter)
	lastAvg := fp.calculateMean(lastQuarter)

	efficiencyDrop := firstAvg - lastAvg

	// Assume readings span proportional time
	// This is a simplification - in production, use actual timestamps
	estimatedDays := float64(n) / 24.0 // Assuming hourly readings

	if estimatedDays == 0 {
		return 0.0
	}

	degradationRate := efficiencyDrop / estimatedDays

	if degradationRate < 0 {
		return 0.0 // Filter improving, not degrading
	}

	return degradationRate
}

// generateRecommendations creates actionable recommendations
func (fp *FilterPredictor) generateRecommendations(
	healthScore float64,
	currentEfficiency float64,
	daysRemaining int,
	trend string,
) []string {
	recommendations := []string{}

	// Health-based recommendations
	switch {
	case healthScore < 30:
		recommendations = append(recommendations, "URGENT: Replace filter immediately")
		recommendations = append(recommendations, "Filter efficiency critically low")
	case healthScore < 50:
		recommendations = append(recommendations, "Schedule filter replacement soon")
		recommendations = append(recommendations, "Monitor water quality closely")
	case healthScore < 75:
		recommendations = append(recommendations, "Perform filter maintenance check")
		recommendations = append(recommendations, "Consider cleaning pre-filters")
	default:
		recommendations = append(recommendations, "Filter operating normally")
	}

	// Efficiency-based recommendations
	if currentEfficiency < 40 {
		recommendations = append(recommendations, "Current efficiency below optimal levels")
	}

	// Time-based recommendations
	if daysRemaining <= 7 {
		recommendations = append(recommendations, fmt.Sprintf("Only %d days until replacement recommended", daysRemaining))
	} else if daysRemaining <= 30 {
		recommendations = append(recommendations, fmt.Sprintf("Plan filter replacement within %d days", daysRemaining))
	}

	// Trend-based recommendations
	if trend == "degrading" {
		recommendations = append(recommendations, "Filter performance is declining - monitor regularly")
	} else if trend == "improving" {
		recommendations = append(recommendations, "Filter performance is stable or improving")
	}

	return recommendations
}

// Enhanced prediction methods combining multiple factors

// predictRemainingDaysEnhanced uses multiple factors for better prediction
func (fp *FilterPredictor) predictRemainingDaysEnhanced(
	efficiencies []float64,
	currentEfficiency float64,
	trend string,
	readings []models.SensorReading,
) int {
	// Get base prediction from efficiency degradation
	efficiencyBasedDays := fp.predictRemainingDays(efficiencies, currentEfficiency, trend)

	// Calculate flow-based prediction
	flowBasedDays := fp.predictByFlowVolume(readings)

	// Calculate age-based prediction
	ageBasedDays := fp.predictByAge(readings)

	// Combine predictions with weighted average
	// Efficiency: 50%, Flow: 30%, Age: 20%
	weightedDays := (float64(efficiencyBasedDays) * 0.5) +
	                (float64(flowBasedDays) * 0.3) +
	                (float64(ageBasedDays) * 0.2)

	finalDays := int(weightedDays)

	// Apply trend adjustment
	switch trend {
	case "degrading":
		finalDays = int(float64(finalDays) * 0.85) // Reduce by 15% if degrading
	case "improving":
		finalDays = int(float64(finalDays) * 1.1) // Increase by 10% if improving
	}

	// Clamp between 0 and max filter life
	if finalDays < 0 {
		finalDays = 0
	}
	if finalDays > fp.maxFilterLifeDays {
		finalDays = fp.maxFilterLifeDays
	}

	return finalDays
}

// predictByFlowVolume predicts remaining days based on water volume processed
func (fp *FilterPredictor) predictByFlowVolume(readings []models.SensorReading) int {
	if len(readings) < 10 {
		return fp.maxFilterLifeDays // Not enough data, return maximum
	}

	// Calculate total flow processed (in liters)
	totalFlow := 0.0
	for _, reading := range readings {
		// Assuming flow is in L/min and readings are taken periodically
		// Estimate flow per reading period (simplified)
		totalFlow += reading.Flow * 15.0 // Assume 15 minutes per reading
	}

	if totalFlow <= 0 {
		return fp.maxFilterLifeDays
	}

	// Calculate remaining capacity
	remainingCapacity := fp.maxFilterVolumeLiters - totalFlow

	if remainingCapacity <= 0 {
		return 0 // Filter capacity exhausted
	}

	// Calculate average daily flow
	if len(readings) < 2 {
		return fp.maxFilterLifeDays
	}

	firstReading := readings[0]
	lastReading := readings[len(readings)-1]
	daysCovered := lastReading.Timestamp.Sub(firstReading.Timestamp).Hours() / 24.0

	if daysCovered <= 0 {
		return fp.maxFilterLifeDays
	}

	averageDailyFlow := totalFlow / daysCovered

	if averageDailyFlow <= 0 {
		return fp.maxFilterLifeDays
	}

	// Calculate days remaining based on flow rate
	daysRemaining := int(remainingCapacity / averageDailyFlow)

	// Clamp to reasonable range
	if daysRemaining < 0 {
		daysRemaining = 0
	}
	if daysRemaining > fp.maxFilterLifeDays {
		daysRemaining = fp.maxFilterLifeDays
	}

	return daysRemaining
}

// predictByAge predicts remaining days based on filter age
func (fp *FilterPredictor) predictByAge(readings []models.SensorReading) int {
	if len(readings) < 2 {
		return fp.maxFilterLifeDays
	}

	// Estimate filter age from first reading
	// In production, this should be tracked explicitly
	oldestReading := readings[0]
	filterAgeDays := int(time.Since(oldestReading.Timestamp).Hours() / 24.0)

	// Calculate remaining days based on maximum filter life
	remainingDays := fp.maxFilterLifeDays - filterAgeDays

	if remainingDays < 0 {
		return 0
	}

	return remainingDays
}

// calculateTotalFlowProcessed calculates total water volume processed
func (fp *FilterPredictor) calculateTotalFlowProcessed(readings []models.SensorReading) float64 {
	if len(readings) == 0 {
		return 0.0
	}

	totalFlow := 0.0
	for _, reading := range readings {
		// Flow is in L/min, estimate volume per reading period
		// Assuming 15-minute intervals between readings
		totalFlow += reading.Flow * 15.0
	}

	return totalFlow
}

// calculateFilterAgeDays calculates filter age in days
func (fp *FilterPredictor) calculateFilterAgeDays(readings []models.SensorReading) int {
	if len(readings) == 0 {
		return 0
	}

	// Find oldest reading as proxy for filter installation date
	oldestReading := readings[0]
	ageDays := int(time.Since(oldestReading.Timestamp).Hours() / 24.0)

	if ageDays < 0 {
		return 0
	}

	return ageDays
}

// Helper functions

func (fp *FilterPredictor) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (fp *FilterPredictor) getRecentAverage(values []float64, n int) float64 {
	if len(values) == 0 {
		return 0.0
	}

	start := len(values) - n
	if start < 0 {
		start = 0
	}

	recent := values[start:]
	return fp.calculateMean(recent)
}
