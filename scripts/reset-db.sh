#!/bin/bash
# AquaSmart Database Reset Script - Useful for testing

set -e  # Exit on any error

echo "🔄 AquaSmart Database Reset"
echo "=========================="

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}⚠️  This will DELETE ALL database data!${NC}"
read -p "Are you sure you want to continue? (y/N): " -n 1 -r
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Operation cancelled."
    exit 0
fi

echo -e "${YELLOW}🛑 Stopping containers...${NC}"
docker-compose down

echo -e "${YELLOW}🗑️  Removing database volume...${NC}"
docker volume rm aquasmart_backend_postgres_data 2>/dev/null || echo "Volume already removed or doesn't exist"

echo -e "${YELLOW}📦 Starting fresh database...${NC}"
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
    exit 1
fi

echo -e "${GREEN}🔄 Database reset complete!${NC}"
echo "   Fresh database with initial schema is ready"