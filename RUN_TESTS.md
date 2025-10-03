# 🧪 AquaSmart Filtration Progress System - Test Guide

This guide explains how to run all the test cases for the filtration progress tracking system.

## ✅ System Status
- ✅ Backend compiles and runs successfully
- ✅ All unit tests pass
- ✅ Integration tests ready
- ✅ MQTT simulation scripts ready

## 🏃‍♂️ Running Tests

### 1. Unit Tests

#### **Model Tests** (Filtration Logic)
```bash
# Run with verbose output
go test ./internal/models -v

# Run with coverage
go test ./internal/models -cover
```

#### **Store Tests** (State Management)
```bash
# Run with verbose output
go test ./internal/store -v

# Run with coverage
go test ./internal/store -cover
```

#### **Handler Tests** (API Validation)
```bash
# Run with verbose output
go test ./internal/http -v

# Run with coverage
go test ./internal/http -cover
```

#### **All Unit Tests**
```bash
# Run all tests
go test ./...

# Run all tests with coverage
go test -cover ./...
```

### 2. Integration Tests

#### **Full System Workflow Test**
```bash
# Terminal 1: Start the server
go run cmd/server/main.go

# Terminal 2: Run integration test
go run test_filtration_workflow.go
# OR
./test_integration
```

#### **Expected Integration Test Output:**
```
🚀 Starting AquaSmart Filtration System Integration Test
============================================================
✅ Server is running

📋 Test 1: Initial State Check
   ✅ Initial state is idle, can change mode: true

📋 Test 2: Start Filtration Process
   ✅ Filtration started successfully

📋 Test 3: Mode Change Blocking
   ✅ Mode change correctly blocked: filtration_in_progress
   📊 Current progress: 0.0%
   📊 Current state: processing

📋 Test 4: WebSocket Real-time Updates
   📨 Received message type: connected
   ✅ WebSocket communication working correctly

🎉 All tests passed successfully!
```

### 3. MQTT Simulation Tests

#### **Basic Simulation**
```bash
# Terminal 1: Start the server
go run cmd/server/main.go

# Terminal 2: Install Python MQTT library (if not installed)
pip install paho-mqtt

# Terminal 3: Run basic simulation
python test_mqtt_filtration.py --duration 5 --interval 2
```

#### **Scenario Testing**
```bash
# Run predefined test scenarios
python test_mqtt_filtration.py --scenario
```

#### **Expected MQTT Simulation Output:**
```
🌊 AquaSmart Filtration MQTT Simulator
========================================
📍 Broker: localhost:1883
🔧 Device ID: test_device_001
✅ Connected to MQTT broker at localhost:1883
📡 Subscribed to command topic: aquasmart/commands/filter
🚰 Starting filtration process: drinking_water mode, target: 50.0L

📊 [18:35:30] Mode: drinking_water, Progress: 15.2%,
              Volume: 7.6L/50.0L, Flow: 2.3L/min
📊 [18:35:32] Mode: drinking_water, Progress: 18.7%,
              Volume: 9.4L/50.0L, Flow: 2.1L/min
```

## 🎯 Testing Scenarios Covered

### **Unit Tests**
- ✅ Filtration process creation and initialization
- ✅ Flow-based progress calculation
- ✅ Time-based completion estimation
- ✅ Mode change validation rules
- ✅ State transition logic
- ✅ Concurrent access safety
- ✅ API request validation
- ✅ Error handling and responses

### **Integration Tests**
- ✅ System initialization and health check
- ✅ Filtration process start via API
- ✅ Mode change blocking with detailed errors
- ✅ WebSocket real-time communication
- ✅ Force override scenarios
- ✅ Progress monitoring over time

### **MQTT Simulation Tests**
- ✅ Realistic sensor data generation
- ✅ Progressive flow rate changes
- ✅ Water quality improvements during filtration
- ✅ Filtration completion detection
- ✅ Command response handling
- ✅ Multiple filtration scenarios

## 🔍 Testing the Frontend Integration

### **API Endpoints to Test**

#### 1. **Check Filtration Status**
```bash
curl -X GET http://localhost:8080/api/v1/filtration/status
```

#### 2. **Start Filtration (Normal)**
```bash
curl -X POST http://localhost:8080/api/v1/commands/filter \
  -H "Content-Type: application/json" \
  -d '{"mode": "drinking_water"}'
```

#### 3. **Try Mode Change (Should Be Blocked)**
```bash
curl -X POST http://localhost:8080/api/v1/commands/filter \
  -H "Content-Type: application/json" \
  -d '{"mode": "household_water"}'
```

#### 4. **Force Mode Change**
```bash
curl -X POST http://localhost:8080/api/v1/commands/filter \
  -H "Content-Type: application/json" \
  -d '{"mode": "household_water", "force": true}'
```

### **WebSocket Testing**
Connect to `ws://localhost:8080/ws` and listen for:
- `filtration_progress` messages
- `mode_change_blocked` messages
- `sensor_reading` messages

## 🏆 Test Success Criteria

### **Unit Tests:** All tests should pass with >80% code coverage
### **Integration Tests:** Complete workflow should execute without errors
### **MQTT Tests:** Should simulate realistic filtration scenarios

## 🐛 Troubleshooting

### **Server Won't Start**
- Check if port 8080 is available
- Ensure Go 1.24+ is installed
- Database connection failure is OK (falls back to in-memory)

### **MQTT Tests Fail**
- Check if MQTT broker is running (mosquitto)
- Or modify broker URL in test scripts
- Default expects broker at localhost:1883

### **WebSocket Connection Fails**
- Check server logs for WebSocket errors
- Verify CORS settings allow your origin
- Test with simple WebSocket client first

## 🎉 Expected Results

When all tests pass, your system will:
1. ✅ Block mode changes during active filtration
2. ✅ Provide detailed progress information
3. ✅ Support force override when safe
4. ✅ Broadcast real-time updates via WebSocket
5. ✅ Handle edge cases gracefully

Your frontend can now safely integrate with these APIs knowing the backend properly handles filtration state management!