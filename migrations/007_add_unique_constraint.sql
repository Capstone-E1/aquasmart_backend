-- Migration to add unique constraint for device_id and timestamp
-- This allows ON CONFLICT clause to work properly for upsert operations

-- Add unique constraint to prevent duplicate readings from same device at same time
ALTER TABLE sensor_readings 
ADD CONSTRAINT sensor_readings_device_timestamp_unique 
UNIQUE (device_id, timestamp);

-- Add comment for documentation
COMMENT ON CONSTRAINT sensor_readings_device_timestamp_unique ON sensor_readings 
IS 'Ensures no duplicate sensor readings from the same device at the same timestamp';
