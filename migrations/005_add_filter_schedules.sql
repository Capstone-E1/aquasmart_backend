-- Migration 005: Add Filter Schedules and Execution History
-- This migration adds automated filter mode scheduling functionality

-- Create filter_schedules table
CREATE TABLE IF NOT EXISTS filter_schedules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    filter_mode VARCHAR(20) NOT NULL CHECK (filter_mode IN ('drinking_water', 'household_water')),
    start_time TIME NOT NULL,
    duration_minutes INTEGER NOT NULL CHECK (duration_minutes > 0),
    days_of_week TEXT[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on filter_schedules for active schedules
CREATE INDEX IF NOT EXISTS idx_filter_schedules_active ON filter_schedules(is_active) WHERE is_active = true;

-- Create index on filter_schedules start_time for efficient lookup
CREATE INDEX IF NOT EXISTS idx_filter_schedules_start_time ON filter_schedules(start_time);

-- Create schedule_executions table to track execution history
CREATE TABLE IF NOT EXISTS schedule_executions (
    id SERIAL PRIMARY KEY,
    schedule_id INTEGER NOT NULL REFERENCES filter_schedules(id) ON DELETE CASCADE,
    executed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('running', 'completed', 'overridden', 'failed', 'cancelled')),
    override_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on schedule_executions for schedule lookup
CREATE INDEX IF NOT EXISTS idx_schedule_executions_schedule_id ON schedule_executions(schedule_id);

-- Create index on schedule_executions for status lookup
CREATE INDEX IF NOT EXISTS idx_schedule_executions_status ON schedule_executions(status);

-- Create index on schedule_executions for date range queries
CREATE INDEX IF NOT EXISTS idx_schedule_executions_executed_at ON schedule_executions(executed_at DESC);

-- Add trigger to update updated_at on filter_schedules
CREATE OR REPLACE FUNCTION update_filter_schedules_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_filter_schedules_updated_at
    BEFORE UPDATE ON filter_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_filter_schedules_updated_at();

-- Insert sample schedules (optional, for testing)
-- Morning drinking water schedule (weekdays 6 AM - 8 AM)
INSERT INTO filter_schedules (name, filter_mode, start_time, duration_minutes, days_of_week, is_active)
VALUES 
    ('Morning Drinking Water', 'drinking_water', '06:00:00', 120, ARRAY['monday', 'tuesday', 'wednesday', 'thursday', 'friday'], true),
    ('Evening Household Water', 'household_water', '18:00:00', 180, ARRAY['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'], true)
ON CONFLICT DO NOTHING;

COMMENT ON TABLE filter_schedules IS 'Stores automated filter mode schedules';
COMMENT ON TABLE schedule_executions IS 'Tracks execution history of filter schedules';
