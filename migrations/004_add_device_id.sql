-- Migration to add device_id column for supporting multiple STM32 devices
-- This allows differentiating between pre-filtration and post-filtration devices

-- Add device_id column to sensor_readings table
-- Default to 'stm32_main' for backward compatibility with existing data
ALTER TABLE sensor_readings
ADD COLUMN IF NOT EXISTS device_id VARCHAR(50) NOT NULL DEFAULT 'stm32_main';

-- Create index for efficient device_id queries
CREATE INDEX IF NOT EXISTS idx_sensor_readings_device_id ON sensor_readings(device_id);

-- Create composite index for device_id and timestamp (common query pattern)
CREATE INDEX IF NOT EXISTS idx_sensor_readings_device_timestamp ON sensor_readings(device_id, timestamp DESC);

-- Update device_status table to include both pre and post filtration devices
INSERT INTO device_status (device_id, current_filter_mode) 
VALUES 
    ('stm32_pre', 'drinking_water'),
    ('stm32_post', 'drinking_water')
ON CONFLICT (device_id) DO NOTHING;

-- Create view for latest reading per device
CREATE OR REPLACE VIEW latest_readings_by_device AS
SELECT DISTINCT ON (device_id)
    device_id,
    id,
    timestamp,
    filter_mode,
    flow,
    ph,
    turbidity,
    tds,
    created_at
FROM sensor_readings
ORDER BY device_id, timestamp DESC;

-- Update existing view to include device_id
DROP VIEW IF EXISTS latest_readings_by_mode;
CREATE VIEW latest_readings_by_mode AS
SELECT DISTINCT ON (filter_mode, device_id)
    device_id,
    filter_mode,
    id,
    timestamp,
    flow,
    ph,
    turbidity,
    tds,
    created_at
FROM sensor_readings
ORDER BY filter_mode, device_id, timestamp DESC;

-- Add comments for documentation
COMMENT ON COLUMN sensor_readings.device_id IS 'Identifier for the device: stm32_pre (pre-filtration) or stm32_post (post-filtration)';
COMMENT ON VIEW latest_readings_by_device IS 'Latest sensor reading for each device';
