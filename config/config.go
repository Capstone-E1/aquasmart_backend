package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the water purification IoT backend
type Config struct {
	Server   ServerConfig
	MQTT     MQTTConfig
	Database DatabaseConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// MQTTConfig holds MQTT broker configuration
type MQTTConfig struct {
	BrokerURL          string
	ClientID           string
	Username           string
	Password           string
	KeepAlive          time.Duration
	PingTimeout        time.Duration
	ConnectRetry       bool
	TopicSensorData    string
	TopicFilterCommand string
}

// DatabaseConfig holds PostgreSQL database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
		},
		MQTT: MQTTConfig{ 
			BrokerURL:          getMQTTBrokerURL(),
			ClientID:           getEnv("MQTT_CLIENT_ID", "aquasmart_backend"),
			Username:           getEnv("MQTT_USERNAME", ""),
			Password:           getEnv("MQTT_PASSWORD", ""),
			KeepAlive:          getDurationEnv("MQTT_KEEP_ALIVE", 30*time.Second),
			PingTimeout:        getDurationEnv("MQTT_PING_TIMEOUT", 10*time.Second),
			ConnectRetry:       getBoolEnv("MQTT_CONNECT_RETRY", true),
			TopicSensorData:    getEnv("MQTT_TOPIC_SENSOR_DATA", "aquasmart/sensors/data"),
			TopicFilterCommand: getEnv("MQTT_TOPIC_FILTER_COMMAND", "aquasmart/filter/command"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "aquasmart"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
	}
}

// getEnv returns environment variable value or default if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDurationEnv returns duration environment variable value or default if not set
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getBoolEnv returns boolean environment variable value or default if not set
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getMQTTBrokerURL returns MQTT broker URL with appropriate scheme prefix
// Supports tcp://, tls://, ssl:// schemes and auto-detects based on MQTT_USE_TLS
func getMQTTBrokerURL() string {
	broker := getEnv("MQTT_BROKER", getEnv("MQTT_BROKER_URL", "tcp://localhost:1883"))
	
	// If broker already has a scheme, return as-is
	if len(broker) >= 6 {
		if broker[:6] == "tcp://" || broker[:6] == "tls://" || broker[:6] == "ssl://" {
			return broker
		}
	}
	if len(broker) >= 7 && broker[:7] == "tcps://" {
		return broker
	}
	
	// Auto-detect scheme based on MQTT_USE_TLS setting
	useTLS := getBoolEnv("MQTT_USE_TLS", false)
	
	if useTLS {
		// Use TLS/SSL for encrypted connections (HiveMQ Cloud, etc.)
		return "tls://" + broker
	}
	
	// Default to TCP for unencrypted local connections
	return "tcp://" + broker
}
