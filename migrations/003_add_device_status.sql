-- Add device_status table to store current operational state
-- Including the current filter mode setting

CREATE TABLE IF NOT EXISTS device_status (
    device_id VARCHAR(50) PRIMARY KEY,
    current_filter_mode VARCHAR(20) NOT NULL DEFAULT 'drinking_water' 
        CHECK (current_filter_mode IN ('drinking_water', 'household_water')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default status for the main STM32 device
INSERT INTO device_status (device_id, current_filter_mode) 
VALUES ('stm32_main', 'drinking_water')
ON CONFLICT (device_id) DO NOTHING;

-- Create index for quick lookups
CREATE INDEX IF NOT EXISTS idx_device_status_device_id ON device_status(device_id);
