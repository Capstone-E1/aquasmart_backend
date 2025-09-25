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
	BrokerURL    string
	ClientID     string
	Username     string
	Password     string
	KeepAlive    time.Duration
	PingTimeout  time.Duration
	ConnectRetry bool
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
			BrokerURL:    getEnv("MQTT_BROKER_URL", "tcp://localhost:1883"),
			ClientID:     getEnv("MQTT_CLIENT_ID", "aquasmart_backend"),
			Username:     getEnv("MQTT_USERNAME", ""),
			Password:     getEnv("MQTT_PASSWORD", ""),
			KeepAlive:    getDurationEnv("MQTT_KEEP_ALIVE", 30*time.Second),
			PingTimeout:  getDurationEnv("MQTT_PING_TIMEOUT", 10*time.Second),
			ConnectRetry: getBoolEnv("MQTT_CONNECT_RETRY", true),
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