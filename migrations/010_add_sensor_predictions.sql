-- Migration: Add Sensor Value Predictions Table
-- Created: 2025-11-07
-- Purpose: Store time-series predictions for sensor values with autonomous updates

-- Sensor Predictions Table
CREATE TABLE IF NOT EXISTS sensor_predictions (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    filter_mode VARCHAR(50) NOT NULL CHECK (filter_mode IN ('drinking_water', 'household_water')),

    -- Prediction metadata
    predicted_for TIMESTAMP WITH TIME ZONE NOT NULL, -- When this prediction is for
    prediction_method VARCHAR(50) DEFAULT 'exponential_smoothing_with_trend',
    confidence_score DECIMAL(3,2) CHECK (confidence_score >= 0 AND confidence_score <= 1),

    -- Predicted sensor values
    predicted_flow DECIMAL(10,2),
    predicted_ph DECIMAL(4,2),
    predicted_turbidity DECIMAL(10,2),
    predicted_tds DECIMAL(10,2),

    -- Actual values (filled when real data arrives)
    actual_flow DECIMAL(10,2),
    actual_ph DECIMAL(4,2),
    actual_turbidity DECIMAL(10,2),
    actual_tds DECIMAL(10,2),

    -- Accuracy metrics
    flow_error DECIMAL(10,2),
    ph_error DECIMAL(4,2),
    turbidity_error DECIMAL(10,2),
    tds_error DECIMAL(10,2),
    overall_accuracy DECIMAL(5,2), -- Percentage accuracy

    -- Status
    is_validated BOOLEAN DEFAULT false, -- Set to true when actual values are recorded

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    validated_at TIMESTAMP WITH TIME ZONE,

    -- Ensure one prediction per device/time/mode combo
    UNIQUE(device_id, filter_mode, predicted_for)
);

-- Prediction Accuracy Summary Table (for tracking model performance)
CREATE TABLE IF NOT EXISTS prediction_accuracy_summary (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    filter_mode VARCHAR(50) NOT NULL CHECK (filter_mode IN ('drinking_water', 'household_water')),

    -- Time period
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Accuracy metrics
    total_predictions INTEGER DEFAULT 0,
    validated_predictions INTEGER DEFAULT 0,

    avg_flow_accuracy DECIMAL(5,2),
    avg_ph_accuracy DECIMAL(5,2),
    avg_turbidity_accuracy DECIMAL(5,2),
    avg_tds_accuracy DECIMAL(5,2),
    overall_accuracy DECIMAL(5,2),

    -- Model info
    model_version VARCHAR(50) DEFAULT 'v1.0',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(device_id, filter_mode, period_start)
);

-- Prediction Update Log Table (tracks when predictions were regenerated)
CREATE TABLE IF NOT EXISTS prediction_update_log (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    filter_mode VARCHAR(50) NOT NULL CHECK (filter_mode IN ('drinking_water', 'household_water')),

    -- Update info
    trigger_reason VARCHAR(100), -- 'new_data', 'scheduled', 'manual', 'accuracy_drop'
    predictions_generated INTEGER DEFAULT 0,
    historical_data_used INTEGER DEFAULT 0, -- How many readings were used

    -- Performance
    execution_time_ms INTEGER, -- How long the prediction took

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_sensor_predictions_device ON sensor_predictions(device_id);
CREATE INDEX IF NOT EXISTS idx_sensor_predictions_predicted_for ON sensor_predictions(predicted_for DESC);
CREATE INDEX IF NOT EXISTS idx_sensor_predictions_device_mode ON sensor_predictions(device_id, filter_mode);
CREATE INDEX IF NOT EXISTS idx_sensor_predictions_validated ON sensor_predictions(is_validated);
CREATE INDEX IF NOT EXISTS idx_sensor_predictions_created_at ON sensor_predictions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_accuracy_summary_device ON prediction_accuracy_summary(device_id);
CREATE INDEX IF NOT EXISTS idx_accuracy_summary_period ON prediction_accuracy_summary(period_start DESC);

CREATE INDEX IF NOT EXISTS idx_update_log_device ON prediction_update_log(device_id);
CREATE INDEX IF NOT EXISTS idx_update_log_created_at ON prediction_update_log(created_at DESC);

-- Add comments for documentation
COMMENT ON TABLE sensor_predictions IS 'Stores time-series predictions for sensor values with validation against actual data';
COMMENT ON TABLE prediction_accuracy_summary IS 'Aggregated accuracy metrics for prediction model performance tracking';
COMMENT ON TABLE prediction_update_log IS 'Audit log of when and why predictions were regenerated';

COMMENT ON COLUMN sensor_predictions.predicted_for IS 'The timestamp this prediction is forecasting for';
COMMENT ON COLUMN sensor_predictions.is_validated IS 'True when actual sensor data has been recorded and compared';
COMMENT ON COLUMN prediction_update_log.trigger_reason IS 'Why the prediction was updated: new_data, scheduled, manual, accuracy_drop';
