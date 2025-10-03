package http

import (
	"testing"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
)

// TestFiltrationBlocking tests the core filtration blocking logic
func TestFiltrationBlocking_Basic(t *testing.T) {
	store := store.NewStore(100)

	// Initially should be able to change mode
	canChange, reason := store.CanChangeFilterMode()
	if !canChange {
		t.Errorf("Expected to be able to change mode initially, got reason: %s", reason)
	}

	// Start filtration process
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)

	// Should not be able to change mode now
	canChange, reason = store.CanChangeFilterMode()
	if canChange {
		t.Error("Expected NOT to be able to change mode during filtration")
	}
	if reason != "filtration_in_progress" {
		t.Errorf("Expected reason 'filtration_in_progress', got '%s'", reason)
	}
}

// TestFiltrationProgress tests progress calculation
func TestFiltrationProgress_Calculation(t *testing.T) {
	store := store.NewStore(100)

	// Start filtration
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)

	// Update progress
	store.UpdateFiltrationProgress(2.5)

	// Get process
	process, exists := store.GetFiltrationProcess()
	if !exists {
		t.Fatal("Expected process to exist")
	}

	if process.CurrentFlowRate != 2.5 {
		t.Errorf("Expected flow rate 2.5, got %v", process.CurrentFlowRate)
	}

	if process.State != models.FiltrationStateProcessing {
		t.Errorf("Expected processing state, got %v", process.State)
	}
}

// TestFiltrationForceMode tests force mode functionality
func TestFiltrationForceMode_Logic(t *testing.T) {
	store := store.NewStore(100)

	// Start filtration
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)

	// Initially can't interrupt
	process, _ := store.GetFiltrationProcess()
	canChange, reason := process.CanChangeMode()
	if canChange {
		t.Error("Expected NOT to be able to change mode initially")
	}
	if reason != "filtration_in_progress" {
		t.Errorf("Expected 'filtration_in_progress', got '%s'", reason)
	}

	// Simulate progress to make interruptible
	process.ProcessedVolume = 0.6 // 12% of 5L
	process.UpdateProgress(2.5)
	store.SetFiltrationProcess(process)

	// Now should be interruptible
	process, _ = store.GetFiltrationProcess()
	canChange, reason = process.CanChangeMode()
	if !canChange {
		t.Error("Expected to be able to change mode after 10% progress")
	}
	if reason != "filtration_interruptible" {
		t.Errorf("Expected 'filtration_interruptible', got '%s'", reason)
	}
}

// TestFiltrationCompletion tests process completion
func TestFiltrationCompletion_Logic(t *testing.T) {
	store := store.NewStore(100)

	// Start and complete process
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)
	store.CompleteFiltrationProcess()

	// Should be able to change mode after completion
	canChange, reason := store.CanChangeFilterMode()
	if !canChange {
		t.Errorf("Expected to be able to change mode after completion, got reason: %s", reason)
	}

	// Process should be marked completed
	process, exists := store.GetFiltrationProcess()
	if !exists {
		t.Fatal("Expected process to exist")
	}

	if process.State != models.FiltrationStateCompleted {
		t.Errorf("Expected completed state, got %v", process.State)
	}

	if process.Progress != 100.0 {
		t.Errorf("Expected 100%% progress, got %v", process.Progress)
	}
}

// TestAPIResponse tests API response structure
func TestAPIResponse_Structure(t *testing.T) {
	response := APIResponse{
		Success: true,
		Message: "Test message",
		Data:    map[string]string{"test": "data"},
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	if response.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got '%s'", response.Message)
	}

	if response.Data == nil {
		t.Error("Expected data to be set")
	}
}