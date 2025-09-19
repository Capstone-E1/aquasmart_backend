# 🌊 AquaSmart Backend

![Go Version](https://img.shields.io/badge/Go-1.24.6-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Status](https://img.shields.io/badge/Status-Active%20Development-orange.svg)

**AquaSmart Backend** is a Go-based IoT backend service for intelligent water purification system monitoring and control. The system integrates MQTT for device communication, WebSocket for real-time client updates, and HTTP REST APIs for client applications.

## 🚀 Features

### ✅ **Core Functionality**
- **4-Sensor Monitoring**: Flow rate, pH, turbidity, and TDS measurement
- **Dual Filtration Modes**: Drinking water and household water filtration
- **Real-time Data Processing**: Live sensor data validation and quality assessment
- **Filter Control**: Remote switching between filtration modes
- **Water Quality Analysis**: Automated good/moderate/poor quality classification

### ✅ **Communication Protocols**
- **MQTT Integration**: Bidirectional IoT device communication
- **WebSocket Support**: Real-time browser updates
- **REST API**: Complete CRUD operations for client applications
- **Cross-Origin Support**: CORS-enabled for web applications

### ✅ **Data Management**
- **In-Memory Storage**: Fast access with configurable capacity (1000 readings default)
- **Filter Context**: Separate data tracking per filtration mode
- **Historical Queries**: Time-range and recent readings retrieval
- **PostgreSQL Ready**: Database schema designed for persistence

## 🏗️ Architecture

```
┌─────────────────┐    MQTT     ┌─────────────────┐
│   STM32/ESP8266 │◄──────────►│  AquaSmart      │
│   (IoT Device)  │   Commands  │   Backend       │
└─────────────────┘   & Data    └─────────────────┘
                                         │
                    ┌────────────────────┼────────────────────┐
                    │                    │                    │
              WebSocket                HTTP                PostgreSQL
                    │                    │                    │
            ┌───────▼──────┐    ┌───────▼───────┐    ┌──────▼──────┐
            │ Web Client   │    │  Mobile App   │    │  Database   │
            │ (Dashboard)  │    │  (Control)    │    │ (Storage)   │
            └──────────────┘    └───────────────┘    └─────────────┘
```

## 📁 Project Structure

```
aquasmart_backend/
├── cmd/server/              # Application entry point
│   └── main.go
├── config/                  # Configuration management
│   └── config.go
├── internal/               # Private application code
│   ├── http/               # HTTP REST API
│   │   ├── handlers.go
│   │   └── routes.go
│   ├── models/             # Data models
│   │   └── sensor.go
│   ├── mqtt/               # MQTT client
│   │   └── client.go
│   ├── services/           # Business logic
│   │   └── parser.go
│   ├── store/              # Data storage
│   │   └── store.go
│   └── ws/                 # WebSocket hub
│       └── hub.go
├── migrations/             # Database migrations
├── scripts/                # Utility scripts
├── docker-compose.yml      # Development environment
└── README_TESTING.md       # Testing guide
```

## 🚀 Quick Start

### Prerequisites

- **Go 1.24.6+** - [Install Go](https://golang.org/doc/install)
- **Docker & Docker Compose** - [Install Docker](https://docs.docker.com/get-docker/)
- **MQTT Broker** (Mosquitto recommended)

### 1. Clone Repository

```bash
git clone https://github.com/your-username/aquasmart_backend.git
cd aquasmart_backend
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Start Database (Optional)

```bash
# Start PostgreSQL and pgAdmin
docker compose --profile admin up -d

# Or just PostgreSQL
docker compose up postgres -d
```

### 4. Configure Environment

```bash
# Create .env file (optional)
cat > .env << EOF
PORT=8080
MQTT_BROKER_URL=tcp://localhost:1883
MQTT_CLIENT_ID=aquasmart_backend
POSTGRES_URL=postgres://aquasmart_user:aquasmart_password@localhost:5433/aquasmart
EOF
```

### 5. Run Application

```bash
# Development (direct run)
go run cmd/server/main.go

# Or build and run
go build -o bin/server cmd/server/main.go
./bin/server
```

### 6. Verify Installation

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Expected response:
# {"success":true,"data":{"status":"healthy","timestamp":"2024-01-15T10:30:25Z","version":"1.0.0"}}
```



## 🧪 Testing

See **[README_TESTING.md](README_TESTING.md)** for comprehensive testing guide.


## 🚧 Development

### Build Commands

```bash
# Format code
go fmt ./...

# Lint code
go vet ./...

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Build binary
go build -o bin/server cmd/server/main.go

# Clean build artifacts
rm -rf bin/
```

### Hot Reload (Optional)

```bash
# Install Air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

## 🐳 Docker Deployment

```bash
# Build image
docker build -t aquasmart-backend .

# Run container
docker run -p 8080:8080 aquasmart-backend
```

## 📊 Monitoring

### Health Endpoints

```bash
# Application health
curl http://localhost:8080/api/v1/health

# System statistics
curl http://localhost:8080/api/v1/stats
```

### Logs

```bash
# Application logs (structured)
tail -f /var/log/aquasmart/app.log

# MQTT connection logs
grep "MQTT" /var/log/aquasmart/app.log
```

