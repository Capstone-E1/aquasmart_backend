#!/bin/bash

# Test script for AquaSmart ML Features
# This script demonstrates the ML API endpoints

BASE_URL="http://localhost:8080/api/v1"

echo "🤖 AquaSmart ML Features Test Script"
echo "======================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. Check ML Dashboard
echo -e "${BLUE}1. Checking ML Dashboard...${NC}"
curl -s "${BASE_URL}/ml/dashboard" | jq '.'
echo ""
echo "---"
echo ""

# 2. Calculate Baselines
echo -e "${BLUE}2. Calculating Sensor Baselines...${NC}"
curl -s -X POST "${BASE_URL}/ml/baselines/calculate" | jq '.'
echo ""
echo "---"
echo ""

# 3. Get Baselines
echo -e "${BLUE}3. Getting Calculated Baselines...${NC}"
curl -s "${BASE_URL}/ml/baselines" | jq '.'
echo ""
echo "---"
echo ""

# 4. Analyze Filter Health
echo -e "${BLUE}4. Analyzing Filter Health...${NC}"
curl -s -X POST "${BASE_URL}/ml/filter/analyze" | jq '.'
echo ""
echo "---"
echo ""

# 5. Get Filter Health
echo -e "${BLUE}5. Getting Filter Health Status...${NC}"
curl -s "${BASE_URL}/ml/filter/health" | jq '.'
echo ""
echo "---"
echo ""

# 6. Detect Anomalies
echo -e "${BLUE}6. Running Anomaly Detection...${NC}"
curl -s -X POST "${BASE_URL}/ml/anomalies/detect" | jq '.'
echo ""
echo "---"
echo ""

# 7. Get Anomalies
echo -e "${BLUE}7. Getting Recent Anomalies...${NC}"
curl -s "${BASE_URL}/ml/anomalies?limit=10" | jq '.'
echo ""
echo "---"
echo ""

# 8. Get Anomaly Stats
echo -e "${BLUE}8. Getting Anomaly Statistics...${NC}"
curl -s "${BASE_URL}/ml/anomalies/stats" | jq '.'
echo ""
echo "---"
echo ""

# 9. Get Unresolved Anomalies
echo -e "${BLUE}9. Getting Unresolved Anomalies...${NC}"
curl -s "${BASE_URL}/ml/anomalies/unresolved" | jq '.'
echo ""
echo "---"
echo ""

echo -e "${GREEN}✅ ML Features Test Complete!${NC}"
echo ""
echo "💡 Tips:"
echo "  - Baselines require at least 10 readings per device/mode"
echo "  - Filter health analysis requires 20+ pre and post readings"
echo "  - The ML service runs background tasks automatically"
echo "  - Check the server logs for ML activity"
echo ""
