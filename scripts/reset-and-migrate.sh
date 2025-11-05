#!/bin/bash

# AquaSmart Database Reset and Migration Script
# This script drops all tables and re-runs all migrations from scratch

set -e  # Exit on error

# Load environment variables from .env file
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Database connection string
DB_URL="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"

echo "🔄 Resetting AquaSmart Database..."
echo "📍 Host: ${DB_HOST}"
echo "📊 Database: ${DB_NAME}"
echo ""

# Drop all existing tables and the schema_migrations table
echo "🗑️  Dropping all existing tables..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF
-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS schedule_executions CASCADE;
DROP TABLE IF EXISTS filter_schedules CASCADE;
DROP TABLE IF EXISTS filtration_process CASCADE;
DROP TABLE IF EXISTS sensor_readings CASCADE;
DROP TABLE IF EXISTS device_status CASCADE;
DROP TABLE IF EXISTS schema_migrations CASCADE;

-- Verify all tables are dropped
SELECT tablename FROM pg_tables WHERE schemaname = 'public';
EOF

echo ""
echo "✅ All tables dropped successfully!"
echo ""
echo "🚀 Now run the server to apply all migrations automatically:"
echo "   go build -o server ./cmd/server && ./server"
