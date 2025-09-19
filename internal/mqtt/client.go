package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/Capstone-E1/aquasmart_backend/internal/models"
	"github.com/Capstone-E1/aquasmart_backend/internal/services"
)

// Client wraps the MQTT client with water purification specific functionality
type Client struct {
	client        mqtt.Client
	parser        *services.SensorParser
	dataHandler   func(*models.SensorReading)
	errorHandler  func(error)
	isConnected   bool
	filterModeFunc func() models.FilterMode // Function to get current filter mode
}

// Config holds MQTT connection configuration
type Config struct {
	BrokerURL    string
	ClientID     string
	Username     string
	Password     string
	KeepAlive    time.Duration
	PingTimeout  time.Duration
	ConnectRetry bool
}

// DefaultConfig returns default MQTT configuration
func DefaultConfig() *Config {
	return &Config{
		BrokerURL:    "tcp://localhost:1883",
		ClientID:     "aquasmart_backend",
		Username:     "",
		Password:     "",
		KeepAlive:    30 * time.Second,
		PingTimeout:  10 * time.Second,
		ConnectRetry: true,
	}
}

// NewClient creates a new MQTT client for water purification IoT
func NewClient(config *Config) *Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.BrokerURL)
	opts.SetClientID(config.ClientID)
	opts.SetKeepAlive(config.KeepAlive)
	opts.SetPingTimeout(config.PingTimeout)
	opts.SetCleanSession(true)

	if config.Username != "" {
		opts.SetUsername(config.Username)
	}
	if config.Password != "" {
		opts.SetPassword(config.Password)
	}

	client := &Client{
		parser:      services.NewSensorParser(),
		isConnected: false,
	}

	// Set connection handlers
	opts.SetDefaultPublishHandler(client.defaultMessageHandler)
	opts.SetOnConnectHandler(client.onConnect)
	opts.SetConnectionLostHandler(client.onConnectionLost)

	client.client = mqtt.NewClient(opts)

	return client
}

// Connect establishes connection to MQTT broker
func (c *Client) Connect() error {
	log.Println("Connecting to MQTT broker...")

	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	log.Println("Successfully connected to MQTT broker")
	c.isConnected = true
	return nil
}

// Disconnect closes the MQTT connection
func (c *Client) Disconnect() {
	if c.isConnected {
		c.client.Disconnect(250)
		c.isConnected = false
		log.Println("Disconnected from MQTT broker")
	}
}

// IsConnected returns the connection status
func (c *Client) IsConnected() bool {
	return c.isConnected && c.client.IsConnected()
}

// SubscribeToSensorData subscribes to sensor data topics
func (c *Client) SubscribeToSensorData() error {
	topics := map[string]byte{
		"aquasmart/sensors/+/data": 1, // + is wildcard for device ID
		"aquasmart/sensors/data":   1, // General sensor data topic
	}

	for topic, qos := range topics {
		if token := c.client.Subscribe(topic, qos, c.sensorDataHandler); token.Wait() && token.Error() != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %w", topic, token.Error())
		}
		log.Printf("Subscribed to topic: %s", topic)
	}

	return nil
}

// SetDataHandler sets the callback function for processed sensor data
func (c *Client) SetDataHandler(handler func(*models.SensorReading)) {
	c.dataHandler = handler
}

// SetErrorHandler sets the callback function for errors
func (c *Client) SetErrorHandler(handler func(error)) {
	c.errorHandler = handler
}

// SetFilterModeFunc sets the function to get the current filter mode
func (c *Client) SetFilterModeFunc(fn func() models.FilterMode) {
	c.filterModeFunc = fn
}

// sensorDataHandler processes incoming sensor data messages
func (c *Client) sensorDataHandler(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received sensor data on topic %s: %s", msg.Topic(), string(msg.Payload()))

	// Get current filter mode
	currentFilterMode := models.FilterModeDrinking // Default
	if c.filterModeFunc != nil {
		currentFilterMode = c.filterModeFunc()
	}

	// Try parsing as JSON first (preferred format)
	reading, err := c.parser.ParseSensorJSON(msg.Payload(), currentFilterMode)
	if err != nil {
		// Fallback to comma-separated format
		reading, err = c.parser.ParseSensorString(string(msg.Payload()), currentFilterMode)
		if err != nil {
			log.Printf("Failed to parse sensor data: %v", err)
			if c.errorHandler != nil {
				c.errorHandler(fmt.Errorf("sensor data parsing failed: %w", err))
			}
			return
		}
	}

	// Log the successful parsing
	log.Printf("Parsed sensor reading: %s", c.parser.FormatSensorReading(reading))

	// Call the data handler if set
	if c.dataHandler != nil {
		c.dataHandler(reading)
	}
}

// defaultMessageHandler handles messages on unsubscribed topics
func (c *Client) defaultMessageHandler(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message on unhandled topic %s: %s", msg.Topic(), string(msg.Payload()))
}

// onConnect callback when connection is established
func (c *Client) onConnect(client mqtt.Client) {
	log.Println("MQTT client connected")
	c.isConnected = true
}

// onConnectionLost callback when connection is lost
func (c *Client) onConnectionLost(client mqtt.Client, err error) {
	log.Printf("MQTT connection lost: %v", err)
	c.isConnected = false

	if c.errorHandler != nil {
		c.errorHandler(fmt.Errorf("MQTT connection lost: %w", err))
	}
}

// PublishCommand publishes a command to control the purification system (single device)
func (c *Client) PublishCommand(command string) error {
	topic := "aquasmart/commands"

	if token := c.client.Publish(topic, 1, false, command); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish command: %w", token.Error())
	}

	log.Printf("Published command to %s: %s", topic, command)
	return nil
}

// PublishFilterCommand publishes a filter mode command to the STM32 device
func (c *Client) PublishFilterCommand(filterCommand *models.FilterCommand) error {
	payload, err := json.Marshal(filterCommand)
	if err != nil {
		return fmt.Errorf("failed to marshal filter command: %w", err)
	}

	return c.PublishCommand(string(payload))
}

// SubscribeToCommandResponses subscribes to command response topics from STM32
func (c *Client) SubscribeToCommandResponses() error {
	topics := map[string]byte{
		"aquasmart/responses": 1, // STM32 sends responses here
	}

	for topic, qos := range topics {
		if token := c.client.Subscribe(topic, qos, c.commandResponseHandler); token.Wait() && token.Error() != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %w", topic, token.Error())
		}
		log.Printf("Subscribed to command response topic: %s", topic)
	}

	return nil
}

// commandResponseHandler processes incoming command responses from STM32
func (c *Client) commandResponseHandler(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received command response on topic %s: %s", msg.Topic(), string(msg.Payload()))

	var response models.CommandResponse
	if err := json.Unmarshal(msg.Payload(), &response); err != nil {
		log.Printf("Failed to parse command response: %v", err)
		if c.errorHandler != nil {
			c.errorHandler(fmt.Errorf("command response parsing failed: %w", err))
		}
		return
	}

	log.Printf("Command response: %s - %s (%s)", response.Command, response.Status, response.Message)

	// You can add a response handler here if needed
	// For now, we just log the response
}