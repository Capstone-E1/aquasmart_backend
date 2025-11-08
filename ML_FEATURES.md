# AquaSmart ML Features Documentation

## Overview

The AquaSmart backend now includes AI/ML capabilities for intelligent water quality monitoring:

1. **Anomaly Detection** - Real-time detection of unusual sensor readings
2. **Filter Lifespan Prediction** - Predictive maintenance for water filters

## Features

### 1. Anomaly Detection

Automatically detects abnormal sensor readings using statistical methods:

- **Z-Score Analysis** - Detects outliers beyond 3 standard deviations
- **Spike Detection** - Identifies sudden increases in sensor values
- **Drift Detection** - Detects gradual sensor calibration drift
- **Sensor Failure Detection** - Identifies impossible or out-of-range values

**Anomaly Types:**
- `spike` - Sudden increase in value
- `sudden_drop` - Sudden decrease in value
- `outlier` - Value outside normal range
- `drift` - Gradual sensor calibration drift
- `sensor_failure` - Sensor malfunction

**Severity Levels:**
- `low` - Minor deviation from baseline
- `medium` - Significant deviation requiring attention
- `high` - Major deviation requiring immediate attention
- `critical` - Severe deviation indicating potential system failure

### 2. Filter Lifespan Prediction

Predicts filter replacement needs based on performance analysis:

- **Health Score** - 0-100 score of filter condition
- **Efficiency Tracking** - Monitors filtration effectiveness over time
- **Degradation Detection** - Identifies declining filter performance
- **Replacement Prediction** - Estimates days until replacement needed

**Health Categories:**
- `excellent` - 90-100 score
- `good` - 75-89 score
- `fair` - 50-74 score
- `poor` - 25-49 score
- `critical` - 0-24 score

## API Endpoints

### ML Dashboard

```http
GET /api/v1/ml/dashboard
```

Returns comprehensive ML metrics including filter health, anomalies, and system status.

**Response:**
```json
{
  "filter_health": {
    "health_score": 87.5,
    "predicted_days_remaining": 45,
    "current_efficiency": 85.2,
    "efficiency_trend": "stable",
    "recommendations": [...]
  },
  "anomalies": {
    "unresolved_count": 2,
    "unresolved": [...],
    "recent": [...],
    "stats": {...}
  }
}
```

### Filter Health

#### Get Filter Health

```http
GET /api/v1/ml/filter/health?device_id=filter_system
```

Returns latest filter health assessment.

#### Analyze Filter Health

```http
POST /api/v1/ml/filter/analyze
```

Triggers a new filter health analysis.

**Response:**
```json
{
  "message": "Filter health analysis completed",
  "health": {
    "device_id": "filter_system",
    "health_score": 87.5,
    "predicted_days_remaining": 45,
    "estimated_replacement": "2025-12-22T10:30:00Z",
    "current_efficiency": 85.2,
    "average_efficiency": 86.1,
    "efficiency_trend": "stable",
    "turbidity_reduction": 92.5,
    "tds_reduction": 78.3,
    "ph_stabilization": 65.0,
    "maintenance_required": false,
    "replacement_urgent": false,
    "recommendations": [
      "Filter operating normally"
    ]
  }
}
```

### Anomaly Detection

#### Get Anomalies

```http
GET /api/v1/ml/anomalies?limit=50&device_id=stm32_pre&severity=high
```

Query parameters:
- `limit` - Number of anomalies to return (default: 50)
- `device_id` - Filter by device (optional)
- `severity` - Filter by severity level (optional)

#### Get Unresolved Anomalies

```http
GET /api/v1/ml/anomalies/unresolved
```

Returns all anomalies that haven't been resolved.

#### Get Anomaly Statistics

```http
GET /api/v1/ml/anomalies/stats
```

**Response:**
```json
{
  "total_anomalies": 45,
  "last_24_hours": 3,
  "last_7_days": 12,
  "by_severity": {
    "low": 20,
    "medium": 15,
    "high": 8,
    "critical": 2
  },
  "by_type": {
    "spike": 15,
    "outlier": 20,
    "drift": 5,
    "sensor_failure": 5
  },
  "false_positive_rate": 8.5,
  "most_affected_device": "stm32_pre"
}
```

#### Detect Anomalies Now

```http
POST /api/v1/ml/anomalies/detect
```

Runs anomaly detection on latest sensor readings immediately.

#### Resolve Anomaly

```http
POST /api/v1/ml/anomalies/{id}/resolve
```

Marks an anomaly as resolved.

#### Mark False Positive

```http
POST /api/v1/ml/anomalies/{id}/false-positive
```

Marks an anomaly as a false positive (helps improve accuracy).

### Sensor Baselines

#### Get Baselines

```http
GET /api/v1/ml/baselines
```

Returns all sensor baselines used for anomaly detection.

**Response:**
```json
{
  "count": 6,
  "baselines": [
    {
      "device_id": "stm32_pre",
      "filter_mode": "drinking_water",
      "flow_mean": 2.5,
      "flow_std_dev": 0.3,
      "ph_mean": 7.2,
      "ph_std_dev": 0.4,
      "turbidity_mean": 1.8,
      "turbidity_std_dev": 0.5,
      "tds_mean": 350.0,
      "tds_std_dev": 45.0,
      "sample_size": 150,
      "calculated_at": "2025-11-07T10:00:00Z"
    }
  ]
}
```

#### Calculate Baselines

```http
POST /api/v1/ml/baselines/calculate
```

Recalculates baselines from historical data.

## How It Works

### Anomaly Detection Process

1. **Baseline Calculation**
   - Every hour, the system calculates statistical baselines (mean, std dev, min, max) for each sensor metric
   - Baselines are device and filter mode specific
   - Requires minimum 10 readings for baseline

2. **Real-Time Detection**
   - When new sensor data arrives, it's compared against the baseline
   - Z-score is calculated: `z = (value - mean) / std_dev`
   - Anomalies detected if `|z| > 3.0` (99.7% confidence interval)

3. **Severity Classification**
   - `z >= 6.0` → Critical
   - `z >= 4.5` → High
   - `z >= 3.5` → Medium
   - `z >= 3.0` → Low

### Filter Health Analysis

1. **Data Collection**
   - Collects pre-filtration readings (stm32_pre)
   - Collects post-filtration readings (stm32_post)
   - Matches readings by timestamp (within 1 minute)

2. **Efficiency Calculation**
   ```
   Turbidity Improvement = (pre - post) / pre * 100
   TDS Improvement = (pre - post) / pre * 100
   pH Stabilization = improvement towards pH 7.0

   Efficiency = (Turbidity * 0.4) + (TDS * 0.4) + (pH * 0.2)
   ```

3. **Trend Analysis**
   - Compares first half vs second half of data
   - Detects if efficiency is improving, stable, or degrading

4. **Lifespan Prediction**
   - Calculates degradation rate per day
   - Estimates days until efficiency drops below 30%
   - Adjusts based on trend (±20%)

## Background Tasks

The ML service runs automated background tasks:

### 1. Baseline Updates
- **Frequency:** Every 1 hour
- **Purpose:** Keep baselines current with latest data patterns
- **Process:** Recalculates statistics for all devices and modes

### 2. Filter Health Analysis
- **Frequency:** Every 30 minutes
- **Purpose:** Track filter performance over time
- **Process:** Analyzes pre/post readings and updates health metrics

### 3. Real-Time Anomaly Detection
- **Frequency:** On every new sensor reading
- **Purpose:** Immediate detection of issues
- **Process:** Compares reading to baseline, saves anomalies

## Requirements

### For Anomaly Detection
- Minimum 10 readings per device/mode for baseline calculation
- More data = better accuracy (recommended: 100+ readings)

### For Filter Health Analysis
- Minimum 20 pre-filtration readings
- Minimum 20 post-filtration readings
- Readings should be within 1 minute of each other

## Usage Examples

### 1. Initial Setup (First Time)

```bash
# 1. Ensure database has enough historical data
# 2. Calculate initial baselines
curl -X POST http://localhost:8080/api/v1/ml/baselines/calculate

# 3. Run initial filter health analysis
curl -X POST http://localhost:8080/api/v1/ml/filter/analyze

# 4. Check ML dashboard
curl http://localhost:8080/api/v1/ml/dashboard
```

### 2. Monitor Anomalies

```bash
# Check for unresolved anomalies
curl http://localhost:8080/api/v1/ml/anomalies/unresolved

# Get anomaly statistics
curl http://localhost:8080/api/v1/ml/anomalies/stats

# Get high severity anomalies
curl "http://localhost:8080/api/v1/ml/anomalies?severity=high&limit=10"
```

### 3. Check Filter Health

```bash
# Get latest filter health
curl http://localhost:8080/api/v1/ml/filter/health

# Trigger new analysis
curl -X POST http://localhost:8080/api/v1/ml/filter/analyze
```

### 4. Resolve Anomalies

```bash
# Mark anomaly as resolved
curl -X POST http://localhost:8080/api/v1/ml/anomalies/1/resolve

# Mark as false positive
curl -X POST http://localhost:8080/api/v1/ml/anomalies/2/false-positive
```

## Database Schema

### Tables Created

1. **filter_health** - Stores filter health assessments
2. **anomaly_detections** - Records detected anomalies
3. **ml_predictions** - General ML predictions
4. **sensor_baselines** - Statistical baselines for anomaly detection

Migration: `migrations/009_add_ml_features.sql`

## Configuration

The ML service can be configured in `internal/ml/ml_service.go`:

```go
baselineUpdateInterval: 1 * time.Hour    // How often to update baselines
healthAnalysisInterval: 30 * time.Minute // How often to analyze filter health
enableRealTimeAnomaly: true              // Enable/disable real-time detection
```

## Monitoring

The ML service logs important events:

- ✅ Baseline updates
- 🔬 Filter health analyses
- ⚠️  Anomaly detections
- 🚨 Urgent maintenance alerts

## Future Enhancements

Potential improvements:
- Machine learning models (LSTM for time-series prediction)
- Adaptive baselines that learn over time
- Predictive water quality forecasting
- Auto-adjustment of filter modes based on predictions
- Integration with notification system for alerts

## Troubleshooting

### "Insufficient data for analysis"
- Wait for more sensor readings to accumulate
- Minimum 20 pre/post readings needed for filter health
- Minimum 10 readings per device/mode for baselines

### "No baseline available"
- Run `POST /api/v1/ml/baselines/calculate`
- Ensure devices are sending data
- Check that at least 10 readings exist per device/mode

### High false positive rate
- Mark false positives using the API
- System will learn over time
- Consider adjusting z-score threshold if needed

## Support

For issues or questions about ML features, check the logs for detailed error messages and system status.
