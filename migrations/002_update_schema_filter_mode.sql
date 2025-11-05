-- AquaSmart Water Purification System - Schema Update
-- This migration updates the schema to support filter mode context and removes device_id dependency

-- Drop existing views that depend on device_id
DROP VIEW IF EXISTS current_water_quality;
DROP VIEW IF EXISTS latest_sensor_readings;

-- Drop water_quality_assessments table (calculate on-the-fly instead)
DROP TABLE IF EXISTS water_quality_assessments;

-- Drop devices table (not needed for single sensor setup)
DROP TABLE IF EXISTS devices;

-- Drop existing indexes on sensor_readings
DROP INDEX IF EXISTS idx_sensor_readings_device_id;
DROP INDEX IF EXISTS idx_sensor_readings_device_timestamp;

-- Update sensor_readings table structure
ALTER TABLE sensor_readings
DROP COLUMN IF EXISTS device_id,
ADD COLUMN IF NOT EXISTS filter_mode VARCHAR(20) NOT NULL DEFAULT 'drinking_water'
    CHECK (filter_mode IN ('drinking_water', 'household_water')),
ADD COLUMN IF NOT EXISTS flow DECIMAL(8,2) NOT NULL DEFAULT 0 CHECK (flow >= 0);

-- Create new optimized indexes
CREATE INDEX IF NOT EXISTS idx_sensor_readings_timestamp ON sensor_readings(timestamp);
CREATE INDEX IF NOT EXISTS idx_sensor_readings_filter_mode ON sensor_readings(filter_mode);
CREATE INDEX IF NOT EXISTS idx_sensor_readings_mode_timestamp ON sensor_readings(filter_mode, timestamp);

-- Create view for latest reading per filter mode
CREATE VIEW latest_readings_by_mode AS
SELECT DISTINCT ON (filter_mode)
    filter_mode,
    id,
    timestamp,
    flow,
    ph,
    turbidity,
    tds,
    created_at
FROM sensor_readings
ORDER BY filter_mode, timestamp DESC;

-- Create view for current water quality (calculated on-the-fly)
CREATE VIEW current_water_quality AS
SELECT
    filter_mode,
    timestamp,
    flow,
    ph,
    CASE
        WHEN ph < 6.5 THEN 'acidic'
        WHEN ph > 8.5 THEN 'alkaline'
        ELSE 'normal'
    END as ph_status,
    turbidity,
    CASE
        WHEN turbidity > 4.0 THEN 'high'
        WHEN turbidity > 1.0 THEN 'moderate'
        ELSE 'low'
    END as turbidity_status,
    tds,
    CASE
        WHEN tds > 500 THEN 'high'
        WHEN tds > 300 THEN 'moderate'
        ELSE 'low'
    END as tds_status,
    CASE
        WHEN (ph < 6.5 OR ph > 8.5 OR turbidity > 4.0 OR tds > 500) THEN 'poor'
        WHEN (turbidity > 1.0 OR tds > 300) THEN 'moderate'
        ELSE 'good'
    END as overall_quality
FROM latest_readings_by_mode;

-- Add comment to explain the new structure
COMMENT ON TABLE sensor_readings IS 'Stores sensor data from single sensor set monitoring both filtration modes';
COMMENT ON COLUMN sensor_readings.filter_mode IS 'Current filtration mode: drinking_water or household_water';
COMMENT ON COLUMN sensor_readings.flow IS 'Water flow rate in liters per minute';