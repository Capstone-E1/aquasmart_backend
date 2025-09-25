package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

// Client represents a WebSocket client connection
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	deviceID string // Optional: filter for specific device
}

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

// Message represents a WebSocket message structure
type Message struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// In production, implement proper origin checking
		return true
	},
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the WebSocket hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("Client connected. Total clients: %d", len(h.clients))

			// Send welcome message
			welcome := Message{
				Type:      "connected",
				Timestamp: time.Now(),
				Data:      map[string]string{"status": "connected"},
			}
			if data, err := json.Marshal(welcome); err == nil {
				select {
				case client.send <- data:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client disconnected. Total clients: %d", len(h.clients))
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// BroadcastSensorReading broadcasts a new sensor reading to all connected clients
func (h *Hub) BroadcastSensorReading(reading *models.SensorReading) {
	message := Message{
		Type:      "sensor_reading",
		Timestamp: time.Now(),
		Data:      reading,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling sensor reading: %v", err)
		return
	}

	select {
	case h.broadcast <- data:
	default:
		log.Println("Broadcast channel is full, dropping message")
	}
}

// BroadcastWaterQualityStatus broadcasts water quality status to all clients
func (h *Hub) BroadcastWaterQualityStatus(status *models.WaterQualityStatus) {
	message := Message{
		Type:      "water_quality_status",
		Timestamp: time.Now(),
		Data:      status,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling water quality status: %v", err)
		return
	}

	select {
	case h.broadcast <- data:
	default:
		log.Println("Broadcast channel is full, dropping message")
	}
}

// BroadcastError broadcasts error messages to all clients
func (h *Hub) BroadcastError(errorMsg string) {
	message := Message{
		Type:      "error",
		Timestamp: time.Now(),
		Data:      map[string]string{"error": errorMsg},
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling error message: %v", err)
		return
	}

	select {
	case h.broadcast <- data:
	default:
		log.Println("Broadcast channel is full, dropping message")
	}
}

// BroadcastFiltrationProgress broadcasts filtration process progress to all connected clients
func (h *Hub) BroadcastFiltrationProgress(process *models.FiltrationProcess) {
	// Create detailed progress data for frontend
	progressData := map[string]interface{}{
		"state":                process.State,
		"current_mode":         process.CurrentMode,
		"progress":             process.Progress,
		"processed_volume":     process.ProcessedVolume,
		"target_volume":        process.TargetVolume,
		"current_flow_rate":    process.CurrentFlowRate,
		"started_at":           process.StartedAt,
		"estimated_completion": process.EstimatedCompletion,
		"can_interrupt":        process.CanInterrupt,
		"status_message":       process.GetStatusMessage(),
	}

	message := Message{
		Type:      "filtration_progress",
		Timestamp: time.Now(),
		Data:      progressData,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling filtration progress: %v", err)
		return
	}

	select {
	case h.broadcast <- data:
	default:
		log.Println("Broadcast channel is full, dropping filtration progress message")
	}
}

// BroadcastModeChangeBlocked broadcasts when a mode change is blocked due to active filtration
func (h *Hub) BroadcastModeChangeBlocked(reason string, process *models.FiltrationProcess) {
	blockData := map[string]interface{}{
		"reason":               reason,
		"retry_after":          process.EstimatedCompletion,
		"current_progress":     process.Progress,
		"status_message":       process.GetStatusMessage(),
	}

	message := Message{
		Type:      "mode_change_blocked",
		Timestamp: time.Now(),
		Data:      blockData,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling mode change blocked message: %v", err)
		return
	}

	select {
	case h.broadcast <- data:
	default:
		log.Println("Broadcast channel is full, dropping mode change blocked message")
	}
}

// GetConnectedClientsCount returns the number of connected clients
func (h *Hub) GetConnectedClientsCount() int {
	return len(h.clients)
}

// HandleWebSocket handles WebSocket connection requests
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Get device ID from query parameter if provided
	deviceID := r.URL.Query().Get("device_id")

	client := &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 256),
		deviceID: deviceID,
	}

	client.hub.register <- client

	// Start goroutines for handling the client
	go client.writePump()
	go client.readPump()
}

// readPump handles reading messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// Set read deadline and pong handler
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages from clients (e.g., device filter requests)
		log.Printf("Received message from client: %s", message)
	}
}

// writePump handles writing messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to current message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}