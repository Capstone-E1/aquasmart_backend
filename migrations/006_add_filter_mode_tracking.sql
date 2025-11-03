-- Migration 006: Add Filter Mode Tracking
-- Track when filter mode started and total flow during current mode

-- Add columns to device_status for filter mode tracking
ALTER TABLE device_status 
ADD COLUMN IF NOT EXISTS filter_mode_started_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS total_flow_liters DECIMAL(10,2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS last_flow_update_at TIMESTAMP WITH TIME ZONE;

-- Add index for faster queries
CREATE INDEX IF NOT EXISTS idx_device_status_filter_tracking 
ON device_status(device_id, filter_mode_started_at);

-- Add comment for documentation
COMMENT ON COLUMN device_status.filter_mode_started_at IS 'Timestamp when current filter mode started';
COMMENT ON COLUMN device_status.total_flow_liters IS 'Total flow (liters) accumulated since current filter mode started';
COMMENT ON COLUMN device_status.last_flow_update_at IS 'Last time flow was calculated';
