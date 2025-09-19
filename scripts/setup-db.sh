#!/bin/bash
# AquaSmart Database Setup Script

set -e  # Exit on any error

echo "🌊 AquaSmart Database Setup"
echo "=========================="

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if Docker is running
if ! docker info &> /dev/null; then
    echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

echo -e "${YELLOW}📦 Starting PostgreSQL container...${NC}"

# Start PostgreSQL container
docker-compose up -d postgres

echo -e "${YELLOW}⏳ Waiting for PostgreSQL to be ready...${NC}"

# Wait for PostgreSQL to be healthy
timeout=60
counter=0
while [ $counter -lt $timeout ]; do
    if docker-compose exec -T postgres pg_isready -U aquasmart_user -d aquasmart &> /dev/null; then
        echo -e "${GREEN}✅ PostgreSQL is ready!${NC}"
        break
    fi

    echo "   Waiting... ($counter/$timeout seconds)"
    sleep 2
    counter=$((counter + 2))
done

if [ $counter -ge $timeout ]; then
    echo -e "${RED}❌ PostgreSQL failed to start within $timeout seconds${NC}"
    echo "Check logs with: docker-compose logs postgres"
    exit 1
fi

echo -e "${GREEN}🗄️  Database setup complete!${NC}"
echo ""
echo "📋 Database Connection Info:"
echo "   Host: localhost"
echo "   Port: 5432"
echo "   Database: aquasmart"
echo "   Username: aquasmart_user"
echo "   Password: aquasmart_password"
echo ""
echo "🔧 Useful commands:"
echo "   View logs: docker-compose logs postgres"
echo "   Stop database: docker-compose down"
echo "   Connect to DB: docker-compose exec postgres psql -U aquasmart_user -d aquasmart"
echo ""
echo "🌐 Optional: Start pgAdmin (web interface):"
echo "   docker-compose --profile admin up -d"
echo "   Access at: http://localhost:8081"
echo "   Email: admin@aquasmart.com"
echo "   Password: admin123"