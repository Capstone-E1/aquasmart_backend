#!/bin/bash
# Test Database Connection to Aiven

echo "🔍 Testing Aiven PostgreSQL Connection..."
echo "========================================="

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
    echo "✅ Loaded .env file"
else
    echo "❌ .env file not found"
    exit 1
fi

echo "📋 Connection Details:"
echo "   Host: $DB_HOST"
echo "   Port: $DB_PORT"
echo "   Database: $DB_NAME"
echo "   User: $DB_USER"
echo "   SSL Mode: $DB_SSLMODE"

# Test database connection using Go
echo ""
echo "🧪 Testing connection..."
go run cmd/migrate/main.go -check