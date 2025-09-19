# 🧪 Testing Your AquaSmart Water Purification IoT System

This guide shows you how to test that your system is properly receiving data from STM32/ESP8266 devices.

## 📋 Prerequisites

1. **MQTT Broker** - You need a running MQTT broker:
   ```bash
   # Install Mosquitto (Ubuntu/Debian)
   sudo apt update
   sudo apt install mosquitto mosquitto-clients

   # Start Mosquitto
   sudo systemctl start mosquitto
   sudo systemctl enable mosquitto

   # Or run with Docker
   docker run -it -p 1883:1883 eclipse-mosquitto:2.0
   ```

2. **Python MQTT Client** (for testing):
   ```bash
   pip install paho-mqtt
   ```

## 🚀 Step-by-Step Testing

### 1. Start Your AquaSmart Backend

```bash
# Build the application
go build -o bin/server cmd/server/main.go

# Run the server
./bin/server

# Or run directly
go run cmd/server/main.go
```

You should see output like:
```
Starting AquaSmart Water Purification IoT Backend...
Loaded configuration: Server port=8080, MQTT broker=tcp://localhost:1883
Initialized data store
Started WebSocket hub
Successfully connected to MQTT broker and subscribed to sensor topics
Starting HTTP server on port 8080
API endpoints available:
  GET /api/v1/sensors/latest - Latest readings from all devices
  ...
```

### 2. Test MQTT Data Reception

#### Option A: Use the Python Test Script
```bash
# Make the script executable
chmod +x test_mqtt_client.py

# Send a single test message
python3 test_mqtt_client.py --mode single --device device_001

# Send test scenarios with different water conditions
python3 test_mqtt_client.py --mode scenarios

# Simulate continuous data (sends every 10 seconds for 5 minutes)
python3 test_mqtt_client.py --mode continuous --interval 10 --duration 300
```

#### Option B: Use MQTT Command Line Tools
```bash
# Send a single sensor reading
mosquitto_pub -h localhost -t "aquasmart/sensors/device_001/data" \
  -m '{"device_id": "device_001", "ph": 7.2, "turbidity": 1.5, "tds": 250.0}'

# Send data with poor water quality
mosquitto_pub -h localhost -t "aquasmart/sensors/device_002/data" \
  -m '{"device_id": "device_002", "ph": 9.1, "turbidity": 6.2, "tds": 580.0}'
```

#### Option C: Simulate Your STM32/ESP8266
Your STM32/ESP8266 should publish to these topics:
- `aquasmart/sensors/{device_id}/data` (recommended)
- `aquasmart/sensors/data` (generic)

**JSON Format:**
```json
{
  "device_id": "device_001",
  "ph": 7.2,
  "turbidity": 1.5,
  "tds": 250.0
}
```

### 3. Verify Data Reception

#### Check Server Logs
When data is received, you'll see:
```
Received sensor data on topic aquasmart/sensors/device_001/data: {"device_id":"device_001","ph":7.2,"turbidity":1.5,"tds":250.0}
Parsed sensor reading: Device: device_001, Time: 2024-01-15 10:30:25, pH: 7.20, Turbidity: 1.50 NTU, TDS: 250.0 ppm
Stored sensor reading: device_001
Client connected. Total clients: 1
```

#### Test HTTP API Endpoints
```bash
# Check if server is running
curl http://localhost:8080/api/v1/health

# Get latest sensor readings
curl http://localhost:8080/api/v1/sensors/latest

# Get water quality status
curl http://localhost:8080/api/v1/sensors/quality

# Get recent readings
curl "http://localhost:8080/api/v1/sensors/recent?limit=10"

# Get active devices
curl http://localhost:8080/api/v1/devices
```

### 4. Test WebSocket Real-time Updates

#### Option A: Use the HTML Test Client
1. Open `test_websocket_client.html` in your browser
2. Click "Connect" to connect to WebSocket
3. Send MQTT data using the Python script or mosquitto_pub
4. Watch real-time updates in the web interface

#### Option B: Use wscat (WebSocket CLI tool)
```bash
# Install wscat
npm install -g wscat

# Connect to WebSocket
wscat -c ws://localhost:8080/ws

# You'll see messages when sensor data arrives
```

## 📊 Expected Test Results

### Good Water Quality Data:
```json
{
  "device_id": "device_001",
  "ph": 7.2,
  "turbidity": 1.5,
  "tds": 250.0
}
```
- pH: 7.2 (Normal - between 6.5-8.5)
- Turbidity: 1.5 NTU (Good - less than 4.0)
- TDS: 250.0 ppm (Low - less than 300)
- Overall Quality: "good"

### Poor Water Quality Data:
```json
{
  "device_id": "device_002",
  "ph": 9.1,
  "turbidity": 6.2,
  "tds": 580.0
}
```
- pH: 9.1 (Alkaline - above 8.5)
- Turbidity: 6.2 NTU (High - above 4.0)
- TDS: 580.0 ppm (High - above 500)
- Overall Quality: "poor"

## 🔧 Troubleshooting

### MQTT Connection Issues:
```bash
# Check if MQTT broker is running
sudo systemctl status mosquitto

# Test MQTT broker manually
mosquitto_sub -h localhost -t "aquasmart/sensors/+/data"
```

### Server Won't Start:
```bash
# Check if port is in use
netstat -tulpn | grep :8080

# Run with different port
PORT=8081 go run cmd/server/main.go
```

### No Data Received:
1. Verify MQTT broker is running
2. Check topic names match exactly
3. Ensure JSON format is correct
4. Check server logs for parsing errors

### WebSocket Connection Fails:
1. Verify server is running on correct port
2. Check browser console for errors
3. Test with different browsers
4. Verify CORS settings

## 🌊 ESP8266/STM32 Integration

For your ESP8266/STM32 devices, use this Arduino code structure:

```cpp
#include <WiFi.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>

const char* ssid = "your_wifi";
const char* password = "your_password";
const char* mqtt_server = "your_mqtt_broker_ip";

WiFiClient espClient;
PubSubClient client(espClient);

void sendSensorData() {
    // Read your sensors
    float ph = readPHSensor();
    float turbidity = readTurbiditySensor();
    float tds = readTDSSensor();

    // Create JSON payload
    StaticJsonDocument<200> doc;
    doc["device_id"] = "device_001";
    doc["ph"] = ph;
    doc["turbidity"] = turbidity;
    doc["tds"] = tds;

    char buffer[256];
    serializeJson(doc, buffer);

    // Publish to MQTT
    client.publish("aquasmart/sensors/device_001/data", buffer);
}
```

## 📈 Performance Monitoring

Monitor these metrics:
- Message processing rate
- WebSocket connection count
- Memory usage
- API response times

```bash
# Check API performance
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8080/api/v1/sensors/latest
```

Your system is working correctly if you see:
1. ✅ MQTT messages being received and parsed
2. ✅ Data stored and accessible via HTTP API
3. ✅ Real-time updates via WebSocket
4. ✅ Proper water quality assessments