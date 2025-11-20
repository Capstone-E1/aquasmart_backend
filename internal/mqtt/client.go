package mqtt

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// Client wraps MQTT client with data store
type Client struct {
	client             MQTT.Client
	store              store.DataStore
	topicSensorData    string
	topicFilterCommand string
}

// NewClient creates and connects a new MQTT client
func NewClient(brokerURL, clientID, username, password string, dataStore store.DataStore, topics map[string]string) (*Client, error) {
	opts := MQTT.NewClientOptions()

	// Add broker URL - support both tcp:// and tls:// schemes
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)

	// Set credentials
	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}

	// Configure TLS for secure connections (HiveMQ Cloud uses TLS on port 8883)
	// This will automatically use TLS if the broker URL uses tls:// or ssl:// scheme
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false, // Set to true only for testing with self-signed certs
		MinVersion:         tls.VersionTLS12,
	}
	opts.SetTLSConfig(tlsConfig)

	// Set callbacks
	opts.SetDefaultPublishHandler(messageHandler)
	opts.SetOnConnectHandler(onConnect)
	opts.SetConnectionLostHandler(onConnectionLost)

	// Connection settings
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetConnectTimeout(30 * time.Second) // Add connection timeout
	opts.SetWriteTimeout(10 * time.Second)

	log.Printf("🔌 Connecting to MQTT broker: %s", brokerURL)

	client := MQTT.NewClient(opts)
	token := client.Connect()
	token.Wait()

	if token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	mqttClient := &Client{
		client:             client,
		store:              dataStore,
		topicSensorData:    topics["sensor_data"],
		topicFilterCommand: topics["filter_command"],
	}

	// Subscribe to sensor data topic
	mqttClient.SubscribeToSensorData()

	log.Printf("✅ MQTT client connected to broker: %s", brokerURL)
	return mqttClient, nil
}

// SubscribeToSensorData subscribes to sensor data topic
func (c *Client) SubscribeToSensorData() {
	token := c.client.Subscribe(c.topicSensorData, 1, c.handleSensorData)
	token.Wait()

	if token.Error() != nil {
		log.Printf("❌ Failed to subscribe to %s: %v", c.topicSensorData, token.Error())
		return
	}

	log.Printf("📡 Subscribed to topic: %s", c.topicSensorData)
}

// handleSensorData handles incoming sensor data from MQTT
func (c *Client) handleSensorData(client MQTT.Client, msg MQTT.Message) {
	log.Printf("📥 Received sensor data from MQTT topic: %s", msg.Topic())

	// Try parsing as dummy data format first (with actual values)
	var dummyPayload struct {
		DeviceID   string  `json:"device_id"`
		FilterMode string  `json:"filter_mode"`
		Timestamp  string  `json:"timestamp"`
		Flow       float64 `json:"flow"`
		Ph         float64 `json:"ph"`
		Turbidity  float64 `json:"turbidity"`
		TDS        float64 `json:"tds"`
	}

	// Parse JSON payload
	var payload struct {
		DeviceID         string  `json:"device_id"`
		Flow             float64 `json:"flow"`
		PhVoltage        float64 `json:"ph_v"`
		TurbidityVoltage float64 `json:"turbidity_v"`
		TDSVoltage       float64 `json:"tds_v"`
	}

	var ph, turbidity, tds float64
	var filterMode models.FilterMode
	var deviceID string
	var flow float64
	isDummyData := false

	// Try parsing as dummy data format (with actual values, not voltages)
	if err := json.Unmarshal(msg.Payload(), &dummyPayload); err == nil && dummyPayload.Ph > 0 {
		// This is dummy data with actual values
		ph = dummyPayload.Ph
		turbidity = dummyPayload.Turbidity
		tds = dummyPayload.TDS
		flow = dummyPayload.Flow
		deviceID = dummyPayload.DeviceID
		filterMode = models.FilterMode(dummyPayload.FilterMode)
		isDummyData = true
		log.Printf("🤖 Detected DUMMY data from %s", deviceID)
	} else {
		// Parse as real sensor data (with voltages)
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			log.Printf("❌ Error parsing sensor data: %v", err)
			log.Printf("   Raw payload: %s", string(msg.Payload()))
			return
		}

		// Convert voltages to actual values
		ph = convertPhVoltage(payload.PhVoltage)
		turbidity = convertTurbidityVoltage(payload.TurbidityVoltage)
		tds = convertTDSVoltage(payload.TDSVoltage)
		flow = payload.Flow
		deviceID = payload.DeviceID
		filterMode = c.store.GetCurrentFilterMode()
		log.Printf("📡 Detected REAL sensor data from %s", deviceID)
	}

	// Create sensor reading
	sensorData := models.SensorReading{
		DeviceID:   deviceID,
		Timestamp:  time.Now(),
		FilterMode: filterMode,
		Flow:       flow,
		Ph:         ph,
		Turbidity:  turbidity,
		TDS:        tds,
	}

	// Store in database
	c.store.AddSensorReading(sensorData)

	logPrefix := "✅"
	if isDummyData {
		logPrefix = "🤖✅"
	}
	log.Printf("%s Stored sensor data from %s via MQTT: Flow=%.2f, pH=%.2f, Turbidity=%.2f, TDS=%.2f",
		logPrefix, deviceID, flow, ph, turbidity, tds)
}

// PublishFilterCommand publishes filter mode change command to ESP32
func (c *Client) PublishFilterCommand(filterMode models.FilterMode) error {
	payload := map[string]interface{}{
		"filter_mode": string(filterMode),
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal filter command: %w", err)
	}

	token := c.client.Publish(c.topicFilterCommand, 1, false, data)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("failed to publish filter command: %w", token.Error())
	}

	log.Printf("📤 Published filter command via MQTT: %s", filterMode)
	return nil
}

// Disconnect gracefully disconnects from MQTT broker
func (c *Client) Disconnect() {
	c.client.Disconnect(250)
	log.Println("🔌 MQTT client disconnected")
}

// Conversion functions (same as HTTP handlers)
func convertPhVoltage(voltage float64) float64 {
	// pH sensor: Voltage 0-3.3V maps to pH 0-14
	// Calibrated formula
	ph := ((voltage * (14.0 / 3.3)) - 1.0)
	if ph < 0 {
		ph = 0
	}
	if ph > 14 {
		ph = 14
	}
	return ph
}

func convertTurbidityVoltage(voltage float64) float64 {
	// Turbidity sensor: Lower voltage = clearer water
	// 0V = 0 NTU (clear), 3.3V = 1000 NTU (very turbid)
	turbidity := (-5 * 1000 / 3.3 * voltage) + 1005.0
	if turbidity < 0 {
		turbidity = 0
	}
	if turbidity > 1000 {
		turbidity = 1000
	}
	return turbidity
}

func convertTDSVoltage(voltage float64) float64 {
	// TDS sensor: 0V = 0 PPM, 3.3V = ~1000 PPM
	// Linear approximation
	tds := voltage * (1000.0 / 3.0)
	if tds < 0 {
		tds = 0
	}
	if tds > 1000 {
		tds = 1000
	}
	return tds
}

// MQTT event handlers
func messageHandler(client MQTT.Client, msg MQTT.Message) {
	log.Printf("📨 Received message on topic: %s", msg.Topic())
}

func onConnect(client MQTT.Client) {
	log.Println("✅ MQTT client connected to broker")
}

func onConnectionLost(client MQTT.Client, err error) {
	log.Printf("⚠️  MQTT connection lost: %v", err)
	log.Println("🔄 Auto-reconnecting...")
}
