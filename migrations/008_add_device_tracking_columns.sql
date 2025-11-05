-- Migration to add missing columns to device_status table
-- These columns are needed for device tracking and flow accumulation

-- Add missing columns for device tracking
ALTER TABLE device_status 
ADD COLUMN IF NOT EXISTS last_seen TIMESTAMPTZ DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS total_readings INTEGER DEFAULT 0;

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_device_status_last_seen ON device_status(last_seen);

-- Add comments for documentation
COMMENT ON COLUMN device_status.last_seen IS 'Timestamp of last sensor data received from this device';
COMMENT ON COLUMN device_status.total_readings IS 'Total number of sensor readings received from this device';
