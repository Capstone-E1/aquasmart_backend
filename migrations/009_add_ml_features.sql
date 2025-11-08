-- Migration: Add Machine Learning Features (Anomaly Detection & Filter Lifespan Prediction)
-- Created: 2025-11-07

-- 1. Filter Health Tracking Table
CREATE TABLE IF NOT EXISTS filter_health (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    filter_mode VARCHAR(50) NOT NULL CHECK (filter_mode IN ('drinking_water', 'household_water')),

    -- Health metrics
    health_score DECIMAL(5,2) NOT NULL CHECK (health_score >= 0 AND health_score <= 100),
    predicted_days_remaining INTEGER NOT NULL DEFAULT 0,
    estimated_replacement TIMESTAMP WITH TIME ZONE,

    -- Performance metrics
    current_efficiency DECIMAL(5,2) DEFAULT 0,
    average_efficiency DECIMAL(5,2) DEFAULT 0,
    efficiency_trend VARCHAR(20) DEFAULT 'stable' CHECK (efficiency_trend IN ('improving', 'stable', 'degrading')),

    -- Degradation indicators
    turbidity_reduction DECIMAL(5,2) DEFAULT 0,
    tds_reduction DECIMAL(5,2) DEFAULT 0,
    ph_stabilization DECIMAL(5,2) DEFAULT 0,

    -- Recommendations
    maintenance_required BOOLEAN DEFAULT false,
    replacement_urgent BOOLEAN DEFAULT false,
    recommendations JSONB,

    -- Timestamps
    last_calculated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Anomaly Detection Table
CREATE TABLE IF NOT EXISTS anomaly_detections (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Anomaly classification
    anomaly_type VARCHAR(50) NOT NULL CHECK (anomaly_type IN ('spike', 'drift', 'outlier', 'sensor_failure', 'sudden_drop', 'pattern_break')),
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),

    -- Affected metrics
    affected_metric VARCHAR(50) NOT NULL CHECK (affected_metric IN ('flow', 'ph', 'turbidity', 'tds')),
    expected_value DECIMAL(10,2),
    actual_value DECIMAL(10,2),
    deviation DECIMAL(10,2),

    -- Context
    filter_mode VARCHAR(50) NOT NULL CHECK (filter_mode IN ('drinking_water', 'household_water')),
    description TEXT,
    is_false_positive BOOLEAN DEFAULT false,
    resolved_at TIMESTAMP WITH TIME ZONE,

    -- Actions
    alert_sent BOOLEAN DEFAULT false,
    auto_resolved BOOLEAN DEFAULT false,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. ML Predictions Table (General purpose predictions)
CREATE TABLE IF NOT EXISTS ml_predictions (
    id SERIAL PRIMARY KEY,
    prediction_type VARCHAR(50) NOT NULL CHECK (prediction_type IN ('water_quality', 'filter_life', 'usage', 'maintenance')),
    device_id VARCHAR(100) NOT NULL,

    -- Prediction data
    predicted_value DECIMAL(10,2),
    confidence_score DECIMAL(3,2) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    predicted_for TIMESTAMP WITH TIME ZONE,

    -- Validation
    actual_value DECIMAL(10,2),
    accuracy DECIMAL(5,2),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 4. Sensor Baselines Table (For anomaly detection)
CREATE TABLE IF NOT EXISTS sensor_baselines (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    filter_mode VARCHAR(50) NOT NULL CHECK (filter_mode IN ('drinking_water', 'household_water')),

    -- Flow baselines
    flow_mean DECIMAL(10,2),
    flow_std_dev DECIMAL(10,2),
    flow_min DECIMAL(10,2),
    flow_max DECIMAL(10,2),

    -- pH baselines
    ph_mean DECIMAL(4,2),
    ph_std_dev DECIMAL(4,2),
    ph_min DECIMAL(4,2),
    ph_max DECIMAL(4,2),

    -- Turbidity baselines
    turbidity_mean DECIMAL(10,2),
    turbidity_std_dev DECIMAL(10,2),
    turbidity_min DECIMAL(10,2),
    turbidity_max DECIMAL(10,2),

    -- TDS baselines
    tds_mean DECIMAL(10,2),
    tds_std_dev DECIMAL(10,2),
    tds_min DECIMAL(10,2),
    tds_max DECIMAL(10,2),

    -- Metadata
    sample_size INTEGER DEFAULT 0,
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(device_id, filter_mode)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_filter_health_device_id ON filter_health(device_id);
CREATE INDEX IF NOT EXISTS idx_filter_health_updated_at ON filter_health(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_filter_health_score ON filter_health(health_score);

CREATE INDEX IF NOT EXISTS idx_anomaly_device_id ON anomaly_detections(device_id);
CREATE INDEX IF NOT EXISTS idx_anomaly_detected_at ON anomaly_detections(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_anomaly_severity ON anomaly_detections(severity);
CREATE INDEX IF NOT EXISTS idx_anomaly_type ON anomaly_detections(anomaly_type);
CREATE INDEX IF NOT EXISTS idx_anomaly_resolved ON anomaly_detections(resolved_at);

CREATE INDEX IF NOT EXISTS idx_predictions_device_id ON ml_predictions(device_id);
CREATE INDEX IF NOT EXISTS idx_predictions_type ON ml_predictions(prediction_type);
CREATE INDEX IF NOT EXISTS idx_predictions_created_at ON ml_predictions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_baselines_device_mode ON sensor_baselines(device_id, filter_mode);

-- Add comments for documentation
COMMENT ON TABLE filter_health IS 'Stores filter health metrics and lifespan predictions';
COMMENT ON TABLE anomaly_detections IS 'Records detected anomalies in sensor readings';
COMMENT ON TABLE ml_predictions IS 'Stores ML model predictions for validation and accuracy tracking';
COMMENT ON TABLE sensor_baselines IS 'Baseline statistics for each device and filter mode used in anomaly detection';
