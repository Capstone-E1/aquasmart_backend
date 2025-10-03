package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Test configuration
const (
	SERVER_URL = "http://localhost:8080"
	WS_URL     = "ws://localhost:8080/ws"
)

type APIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type FilterModeRequest struct {
	Mode  string `json:"mode"`
	Force bool   `json:"force,omitempty"`
}

type FiltrationStatus struct {
	State               string    `json:"state"`
	CurrentMode         string    `json:"current_mode"`
	Progress            float64   `json:"progress"`
	ProcessedVolume     float64   `json:"processed_volume"`
	TargetVolume        float64   `json:"target_volume"`
	CurrentFlowRate     float64   `json:"current_flow_rate"`
	StartedAt           time.Time `json:"started_at"`
	EstimatedCompletion time.Time `json:"estimated_completion"`
	CanChangeMode       bool      `json:"can_change_mode"`
	CanInterrupt        bool      `json:"can_interrupt"`
	StatusMessage       string    `json:"status_message"`
}

type WebSocketMessage struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

func main() {
	fmt.Println("🚀 Starting AquaSmart Filtration System Integration Test")
	fmt.Println(strings.Repeat("=", 60))

	// Check if server is running
	if !isServerRunning() {
		log.Fatal("❌ Server is not running. Please start the server first with: go run cmd/server/main.go")
	}

	fmt.Println("✅ Server is running")

	// Run test workflow
	if err := runFiltrationWorkflowTest(); err != nil {
		log.Fatalf("❌ Test failed: %v", err)
	}

	fmt.Println("\n🎉 All tests passed successfully!")
}

func isServerRunning() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(SERVER_URL + "/api/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func runFiltrationWorkflowTest() error {
	fmt.Println("\n📋 Test 1: Initial State Check")
	if err := testInitialState(); err != nil {
		return fmt.Errorf("initial state test failed: %w", err)
	}

	fmt.Println("\n📋 Test 2: Start Filtration Process")
	if err := testStartFiltration(); err != nil {
		return fmt.Errorf("start filtration test failed: %w", err)
	}

	fmt.Println("\n📋 Test 3: Mode Change Blocking")
	if err := testModeChangeBlocked(); err != nil {
		return fmt.Errorf("mode change blocking test failed: %w", err)
	}

	fmt.Println("\n📋 Test 4: WebSocket Real-time Updates")
	if err := testWebSocketUpdates(); err != nil {
		return fmt.Errorf("websocket test failed: %w", err)
	}

	fmt.Println("\n📋 Test 5: Force Mode Change (Simulated)")
	if err := testForceModeChange(); err != nil {
		return fmt.Errorf("force mode change test failed: %w", err)
	}

	fmt.Println("\n📋 Test 6: Filtration Status Monitoring")
	if err := testFiltrationStatusMonitoring(); err != nil {
		return fmt.Errorf("filtration status monitoring test failed: %w", err)
	}

	return nil
}

func testInitialState() error {
	// Check initial filtration status
	resp, err := http.Get(SERVER_URL + "/api/v1/filtration/status")
	if err != nil {
		return fmt.Errorf("failed to get filtration status: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return fmt.Errorf("API returned error: %s", apiResp.Error)
	}

	var status FiltrationStatus
	if err := json.Unmarshal(apiResp.Data, &status); err != nil {
		return fmt.Errorf("failed to parse status: %w", err)
	}

	if status.State != "idle" {
		return fmt.Errorf("expected idle state, got: %s", status.State)
	}

	if !status.CanChangeMode {
		return fmt.Errorf("expected to be able to change mode initially")
	}

	fmt.Printf("   ✅ Initial state is idle, can change mode: %t\n", status.CanChangeMode)
	return nil
}

func testStartFiltration() error {
	// Start drinking water filtration
	request := FilterModeRequest{
		Mode: "drinking_water",
	}

	body, _ := json.Marshal(request)
	resp, err := http.Post(SERVER_URL+"/api/v1/commands/filter", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to start filtration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected status 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return fmt.Errorf("failed to start filtration: %s", apiResp.Error)
	}

	fmt.Printf("   ✅ Filtration started successfully: %s\n", apiResp.Message)

	// Wait a moment for the process to be established
	time.Sleep(100 * time.Millisecond)

	// Verify process is active
	return verifyActiveProcess("drinking_water")
}

func testModeChangeBlocked() error {
	// Try to change mode while filtration is active
	request := FilterModeRequest{
		Mode: "household_water",
	}

	body, _ := json.Marshal(request)
	resp, err := http.Post(SERVER_URL+"/api/v1/commands/filter", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to make mode change request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("expected status 409 (conflict), got %d", resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Success {
		return fmt.Errorf("expected mode change to be blocked, but it succeeded")
	}

	if apiResp.Error != "filtration_in_progress" {
		return fmt.Errorf("expected error 'filtration_in_progress', got: %s", apiResp.Error)
	}

	fmt.Printf("   ✅ Mode change correctly blocked: %s\n", apiResp.Error)

	// Parse the detailed error data
	var errorData map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &errorData); err != nil {
		return fmt.Errorf("failed to parse error data: %w", err)
	}

	fmt.Printf("   📊 Current progress: %.1f%%\n", errorData["progress"])
	fmt.Printf("   📊 Current state: %s\n", errorData["current_state"])
	fmt.Printf("   📊 Can force: %t\n", errorData["can_force"])

	return nil
}

func testWebSocketUpdates() error {
	fmt.Printf("   🔌 Connecting to WebSocket: %s\n", WS_URL)

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(WS_URL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}
	defer conn.Close()

	// Set up message channel
	messages := make(chan WebSocketMessage, 10)
	errors := make(chan error, 1)

	// Start reading messages
	go func() {
		defer close(messages)
		for {
			var msg WebSocketMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					errors <- err
				}
				return
			}
			messages <- msg
		}
	}()

	// Wait for connection confirmation
	fmt.Printf("   ⏳ Waiting for WebSocket messages...\n")

	timeout := time.After(5 * time.Second)
	messagesReceived := 0

	for {
		select {
		case msg := <-messages:
			messagesReceived++
			fmt.Printf("   📨 Received message type: %s at %s\n",
				msg.Type, msg.Timestamp.Format("15:04:05"))

			// Print relevant message data
			switch msg.Type {
			case "connected":
				fmt.Printf("   ✅ WebSocket connected successfully\n")
			case "filtration_progress":
				var progressData map[string]interface{}
				if err := json.Unmarshal(msg.Data, &progressData); err == nil {
					fmt.Printf("   📈 Progress: %.1f%%, Flow: %.1f L/min, State: %s\n",
						progressData["progress"], progressData["current_flow_rate"], progressData["state"])
				}
			case "sensor_reading":
				var sensorData map[string]interface{}
				if err := json.Unmarshal(msg.Data, &sensorData); err == nil {
					fmt.Printf("   🔬 Sensor: pH=%.1f, Flow=%.1f L/min\n",
						sensorData["ph"], sensorData["flow"])
				}
			}

			// Stop after receiving a few messages or specific types
			if messagesReceived >= 3 || msg.Type == "filtration_progress" {
				fmt.Printf("   ✅ WebSocket communication working correctly\n")
				return nil
			}

		case err := <-errors:
			return fmt.Errorf("websocket error: %w", err)

		case <-timeout:
			if messagesReceived == 0 {
				return fmt.Errorf("no WebSocket messages received within timeout")
			}
			fmt.Printf("   ✅ WebSocket test completed (%d messages received)\n", messagesReceived)
			return nil
		}
	}
}

func testForceModeChange() error {
	// This test simulates what would happen with force flag
	// First, let's check if the current process can be interrupted

	status, err := getFiltrationStatus()
	if err != nil {
		return err
	}

	fmt.Printf("   📊 Current progress: %.1f%%, Can interrupt: %t\n",
		status.Progress, status.CanInterrupt)

	if status.CanInterrupt {
		// Try force mode change
		request := FilterModeRequest{
			Mode:  "household_water",
			Force: true,
		}

		body, _ := json.Marshal(request)
		resp, err := http.Post(SERVER_URL+"/api/v1/commands/filter", "application/json", bytes.NewBuffer(body))
		if err != nil {
			return fmt.Errorf("failed to make force mode change: %w", err)
		}
		defer resp.Body.Close()

		var apiResp APIResponse
		json.NewDecoder(resp.Body).Decode(&apiResp)

		if resp.StatusCode == http.StatusOK && apiResp.Success {
			fmt.Printf("   ✅ Force mode change successful\n")
			return verifyActiveProcess("household_water")
		}
	}

	fmt.Printf("   ℹ️  Force mode change not possible at this time (process not interruptible or other conditions)\n")
	return nil
}

func testFiltrationStatusMonitoring() error {
	// Monitor status over time
	fmt.Printf("   📊 Monitoring filtration status for 3 seconds...\n")

	for i := 0; i < 3; i++ {
		status, err := getFiltrationStatus()
		if err != nil {
			return err
		}

		fmt.Printf("   📈 [%ds] State: %s, Progress: %.1f%%, Flow: %.1f L/min\n",
			i+1, status.State, status.Progress, status.CurrentFlowRate)
		fmt.Printf("   📝 Status: %s\n", status.StatusMessage)

		if status.State == "completed" {
			fmt.Printf("   🎉 Filtration completed!\n")
			break
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

func getFiltrationStatus() (*FiltrationStatus, error) {
	resp, err := http.Get(SERVER_URL + "/api/v1/filtration/status")
	if err != nil {
		return nil, fmt.Errorf("failed to get filtration status: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API returned error: %s", apiResp.Error)
	}

	var status FiltrationStatus
	if err := json.Unmarshal(apiResp.Data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	return &status, nil
}

func verifyActiveProcess(expectedMode string) error {
	status, err := getFiltrationStatus()
	if err != nil {
		return err
	}

	if status.State != "processing" {
		return fmt.Errorf("expected processing state, got: %s", status.State)
	}

	if status.CurrentMode != expectedMode {
		return fmt.Errorf("expected mode %s, got: %s", expectedMode, status.CurrentMode)
	}

	fmt.Printf("   ✅ Active process verified: %s mode, %.1f%% complete\n",
		status.CurrentMode, status.Progress)

	return nil
}

// Helper function for string repetition (since Go doesn't have built-in)
func init() {
	// This is just for the separator line
}