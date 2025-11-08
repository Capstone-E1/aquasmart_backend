package models

import (
	"time"
)

// FilterHealth represents the health status and lifespan prediction of a filter
type FilterHealth struct {
	ID                    int       `json:"id"`
	DeviceID              string    `json:"device_id"`
	FilterMode            FilterMode `json:"filter_mode"`
	HealthScore           float64   `json:"health_score"`           // 0-100, 100 = excellent
	PredictedDaysRemaining int      `json:"predicted_days_remaining"`
	EstimatedReplacement  time.Time `json:"estimated_replacement"`

	// Performance metrics
	CurrentEfficiency     float64   `json:"current_efficiency"`     // 0-100%
	AverageEfficiency     float64   `json:"average_efficiency"`     // 0-100%
	EfficiencyTrend       string    `json:"efficiency_trend"`       // "improving", "stable", "degrading"

	// Degradation indicators
	TurbidityReduction    float64   `json:"turbidity_reduction"`    // % reduction pre to post
	TDSReduction          float64   `json:"tds_reduction"`          // % reduction pre to post
	PhStabilization       float64   `json:"ph_stabilization"`       // How well pH is maintained

	// Additional tracking metrics
	TotalFlowProcessed    float64   `json:"total_flow_processed"`   // Total water volume processed (liters)
	FilterAgeDays         int       `json:"filter_age_days"`        // Estimated filter age (days)

	// Recommendations
	MaintenanceRequired   bool      `json:"maintenance_required"`
	ReplacementUrgent     bool      `json:"replacement_urgent"`
	Recommendations       []string  `json:"recommendations"`

	LastCalculated        time.Time `json:"last_calculated"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// AnomalyDetection represents detected anomalies in sensor readings
type AnomalyDetection struct {
	ID               int       `json:"id"`
	DeviceID         string    `json:"device_id"`
	DetectedAt       time.Time `json:"detected_at"`
	AnomalyType      string    `json:"anomaly_type"`      // "spike", "drift", "outlier", "sensor_failure"
	Severity         string    `json:"severity"`          // "low", "medium", "high", "critical"

	// Affected metrics
	AffectedMetric   string    `json:"affected_metric"`   // "flow", "ph", "turbidity", "tds"
	ExpectedValue    float64   `json:"expected_value"`
	ActualValue      float64   `json:"actual_value"`
	Deviation        float64   `json:"deviation"`         // Percentage deviation

	// Context
	FilterMode       FilterMode `json:"filter_mode"`
	Description      string    `json:"description"`
	IsFalsePositive  bool      `json:"is_false_positive"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`

	// Actions taken
	AlertSent        bool      `json:"alert_sent"`
	AutoResolved     bool      `json:"auto_resolved"`

	CreatedAt        time.Time `json:"created_at"`
}

// MLPrediction represents general ML predictions
type MLPrediction struct {
	ID              int       `json:"id"`
	PredictionType  string    `json:"prediction_type"`  // "water_quality", "filter_life", "usage"
	DeviceID        string    `json:"device_id"`
	PredictedValue  float64   `json:"predicted_value"`
	ConfidenceScore float64   `json:"confidence_score"` // 0-1
	PredictedFor    time.Time `json:"predicted_for"`
	ActualValue     *float64  `json:"actual_value,omitempty"`
	Accuracy        *float64  `json:"accuracy,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// AnomalyStats represents statistics about anomalies
type AnomalyStats struct {
	TotalAnomalies      int                `json:"total_anomalies"`
	Last24Hours         int                `json:"last_24_hours"`
	Last7Days           int                `json:"last_7_days"`
	BySeverity          map[string]int     `json:"by_severity"`
	ByType              map[string]int     `json:"by_type"`
	ByMetric            map[string]int     `json:"by_metric"`
	FalsePositiveRate   float64            `json:"false_positive_rate"`
	MostAffectedDevice  string             `json:"most_affected_device"`
}

// FilterHealthSummary provides a summary of all filter health metrics
type FilterHealthSummary struct {
	DeviceID              string    `json:"device_id"`
	OverallHealth         string    `json:"overall_health"`    // "excellent", "good", "fair", "poor", "critical"
	HealthScore           float64   `json:"health_score"`
	DaysUntilReplacement  int       `json:"days_until_replacement"`
	ReplacementDate       time.Time `json:"replacement_date"`
	MaintenanceNeeded     bool      `json:"maintenance_needed"`
	RecommendedActions    []string  `json:"recommended_actions"`
	LastAssessment        time.Time `json:"last_assessment"`
}

// SensorBaseline represents normal baseline values for anomaly detection
type SensorBaseline struct {
	DeviceID         string     `json:"device_id"`
	FilterMode       FilterMode `json:"filter_mode"`

	// Flow baselines
	FlowMean         float64    `json:"flow_mean"`
	FlowStdDev       float64    `json:"flow_std_dev"`
	FlowMin          float64    `json:"flow_min"`
	FlowMax          float64    `json:"flow_max"`

	// pH baselines
	PhMean           float64    `json:"ph_mean"`
	PhStdDev         float64    `json:"ph_std_dev"`
	PhMin            float64    `json:"ph_min"`
	PhMax            float64    `json:"ph_max"`

	// Turbidity baselines
	TurbidityMean    float64    `json:"turbidity_mean"`
	TurbidityStdDev  float64    `json:"turbidity_std_dev"`
	TurbidityMin     float64    `json:"turbidity_min"`
	TurbidityMax     float64    `json:"turbidity_max"`

	// TDS baselines
	TDSMean          float64    `json:"tds_mean"`
	TDSStdDev        float64    `json:"tds_std_dev"`
	TDSMin           float64    `json:"tds_min"`
	TDSMax           float64    `json:"tds_max"`

	SampleSize       int        `json:"sample_size"`
	CalculatedAt     time.Time  `json:"calculated_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// GetHealthCategory returns the health category based on score
func (fh *FilterHealth) GetHealthCategory() string {
	switch {
	case fh.HealthScore >= 90:
		return "excellent"
	case fh.HealthScore >= 75:
		return "good"
	case fh.HealthScore >= 50:
		return "fair"
	case fh.HealthScore >= 25:
		return "poor"
	default:
		return "critical"
	}
}

// NeedsAttention returns true if filter needs immediate attention
func (fh *FilterHealth) NeedsAttention() bool {
	return fh.HealthScore < 50 || fh.PredictedDaysRemaining < 7 || fh.ReplacementUrgent
}

// GetSeverityLevel returns severity level for anomaly
func (ad *AnomalyDetection) GetSeverityLevel() int {
	switch ad.Severity {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	default:
		return 0
	}
}

// IsResolved returns true if anomaly is resolved
func (ad *AnomalyDetection) IsResolved() bool {
	return ad.ResolvedAt != nil
}

// IsStale returns true if anomaly detection is older than specified duration
func (ad *AnomalyDetection) IsStale(duration time.Duration) bool {
	return time.Since(ad.DetectedAt) > duration
}

// CalculateFilterEfficiency calculates filter efficiency from pre/post readings
func CalculateFilterEfficiency(preReading, postReading *SensorReading) float64 {
	if preReading == nil || postReading == nil {
		return 0.0
	}

	// Calculate improvement in each metric
	turbidityImprovement := 0.0
	if preReading.Turbidity > 0 {
		turbidityImprovement = ((preReading.Turbidity - postReading.Turbidity) / preReading.Turbidity) * 100
	}

	tdsImprovement := 0.0
	if preReading.TDS > 0 {
		tdsImprovement = ((preReading.TDS - postReading.TDS) / preReading.TDS) * 100
	}

	// pH stabilization (closer to 7.0 is better)
	phImprovement := 0.0
	prePhDeviation := abs(preReading.Ph - 7.0)
	postPhDeviation := abs(postReading.Ph - 7.0)
	if prePhDeviation > 0 {
		phImprovement = ((prePhDeviation - postPhDeviation) / prePhDeviation) * 100
	}

	// Weighted average (turbidity and TDS are more important)
	efficiency := (turbidityImprovement * 0.4) + (tdsImprovement * 0.4) + (phImprovement * 0.2)

	// Clamp between 0 and 100
	if efficiency < 0 {
		efficiency = 0
	}
	if efficiency > 100 {
		efficiency = 100
	}

	return efficiency
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// SensorPrediction represents a time-series prediction for sensor values
type SensorPrediction struct {
	ID              int        `json:"id"`
	DeviceID        string     `json:"device_id"`
	FilterMode      FilterMode `json:"filter_mode"`

	// Prediction metadata
	PredictedFor     time.Time `json:"predicted_for"`
	PredictionMethod string    `json:"prediction_method"`
	ConfidenceScore  float64   `json:"confidence_score"`

	// Predicted values
	PredictedFlow      float64 `json:"predicted_flow"`
	PredictedPh        float64 `json:"predicted_ph"`
	PredictedTurbidity float64 `json:"predicted_turbidity"`
	PredictedTDS       float64 `json:"predicted_tds"`

	// Actual values (when available)
	ActualFlow      *float64 `json:"actual_flow,omitempty"`
	ActualPh        *float64 `json:"actual_ph,omitempty"`
	ActualTurbidity *float64 `json:"actual_turbidity,omitempty"`
	ActualTDS       *float64 `json:"actual_tds,omitempty"`

	// Accuracy metrics
	FlowError       *float64 `json:"flow_error,omitempty"`
	PhError         *float64 `json:"ph_error,omitempty"`
	TurbidityError  *float64 `json:"turbidity_error,omitempty"`
	TDSError        *float64 `json:"tds_error,omitempty"`
	OverallAccuracy *float64 `json:"overall_accuracy,omitempty"`

	// Status
	IsValidated bool       `json:"is_validated"`
	ValidatedAt *time.Time `json:"validated_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PredictionAccuracySummary tracks model performance over time
type PredictionAccuracySummary struct {
	ID                     int        `json:"id"`
	DeviceID               string     `json:"device_id"`
	FilterMode             FilterMode `json:"filter_mode"`

	// Time period
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`

	// Metrics
	TotalPredictions     int     `json:"total_predictions"`
	ValidatedPredictions int     `json:"validated_predictions"`

	AvgFlowAccuracy      float64 `json:"avg_flow_accuracy"`
	AvgPhAccuracy        float64 `json:"avg_ph_accuracy"`
	AvgTurbidityAccuracy float64 `json:"avg_turbidity_accuracy"`
	AvgTDSAccuracy       float64 `json:"avg_tds_accuracy"`
	OverallAccuracy      float64 `json:"overall_accuracy"`

	ModelVersion string    `json:"model_version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PredictionUpdateLog tracks when predictions are regenerated
type PredictionUpdateLog struct {
	ID                   int       `json:"id"`
	DeviceID             string    `json:"device_id"`
	FilterMode           FilterMode `json:"filter_mode"`
	TriggerReason        string    `json:"trigger_reason"` // "new_data", "scheduled", "manual", "accuracy_drop"
	PredictionsGenerated int       `json:"predictions_generated"`
	HistoricalDataUsed   int       `json:"historical_data_used"`
	ExecutionTimeMs      int       `json:"execution_time_ms"`
	CreatedAt            time.Time `json:"created_at"`
}

// IsAccurate returns true if prediction was reasonably accurate
func (sp *SensorPrediction) IsAccurate(threshold float64) bool {
	if !sp.IsValidated || sp.OverallAccuracy == nil {
		return false
	}
	return *sp.OverallAccuracy >= threshold
}

// IsFuturePrediction returns true if prediction is for future time
func (sp *SensorPrediction) IsFuturePrediction() bool {
	return sp.PredictedFor.After(time.Now())
}

// TimeUntilPrediction returns duration until predicted time
func (sp *SensorPrediction) TimeUntilPrediction() time.Duration {
	return time.Until(sp.PredictedFor)
}
