-- AquaSmart Water Purification System Database Schema
-- This file creates the initial database structure for sensor data storage

-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Sensor readings table - stores all sensor data from IoT devices
CREATE TABLE sensor_readings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ph DECIMAL(4,2) NOT NULL CHECK (ph >= 0 AND ph <= 14),
    turbidity DECIMAL(8,2) NOT NULL CHECK (turbidity >= 0),
    tds DECIMAL(8,1) NOT NULL CHECK (tds >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Devices table - tracks active IoT devices
CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100),
    location VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_seen TIMESTAMPTZ,
    firmware_version VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Water quality assessments table - stores processed quality analysis
CREATE TABLE water_quality_assessments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(50) NOT NULL,
    sensor_reading_id UUID REFERENCES sensor_readings(id),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ph_status VARCHAR(20) NOT NULL CHECK (ph_status IN ('acidic', 'normal', 'alkaline')),
    turbidity_status VARCHAR(20) NOT NULL CHECK (turbidity_status IN ('low', 'moderate', 'high')),
    tds_status VARCHAR(20) NOT NULL CHECK (tds_status IN ('low', 'moderate', 'high')),
    overall_quality VARCHAR(20) NOT NULL CHECK (overall_quality IN ('good', 'moderate', 'poor')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_sensor_readings_device_id ON sensor_readings(device_id);
CREATE INDEX idx_sensor_readings_timestamp ON sensor_readings(timestamp);
CREATE INDEX idx_sensor_readings_device_timestamp ON sensor_readings(device_id, timestamp);

CREATE INDEX idx_devices_device_id ON devices(device_id);
CREATE INDEX idx_devices_active ON devices(is_active);

CREATE INDEX idx_quality_device_id ON water_quality_assessments(device_id);
CREATE INDEX idx_quality_timestamp ON water_quality_assessments(timestamp);
CREATE INDEX idx_quality_overall ON water_quality_assessments(overall_quality);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at
CREATE TRIGGER update_sensor_readings_updated_at
    BEFORE UPDATE ON sensor_readings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_devices_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert some sample devices (optional)
INSERT INTO devices (device_id, name, location) VALUES
    ('device_001', 'Main Tank Sensor', 'Primary Water Tank'),
    ('device_002', 'Filter Output Sensor', 'Filter Output Point'),
    ('device_003', 'Storage Tank Sensor', 'Clean Water Storage')
ON CONFLICT (device_id) DO NOTHING;

-- Create a view for latest readings per device
CREATE VIEW latest_sensor_readings AS
SELECT DISTINCT ON (device_id)
    device_id,
    id,
    timestamp,
    ph,
    turbidity,
    tds,
    created_at
FROM sensor_readings
ORDER BY device_id, timestamp DESC;

-- Create a view for current water quality status
CREATE VIEW current_water_quality AS
SELECT DISTINCT ON (wqa.device_id)
    wqa.device_id,
    wqa.timestamp,
    sr.ph,
    wqa.ph_status,
    sr.turbidity,
    wqa.turbidity_status,
    sr.tds,
    wqa.tds_status,
    wqa.overall_quality,
    d.name as device_name,
    d.location as device_location
FROM water_quality_assessments wqa
JOIN sensor_readings sr ON wqa.sensor_reading_id = sr.id
LEFT JOIN devices d ON wqa.device_id = d.device_id
ORDER BY wqa.device_id, wqa.timestamp DESC;