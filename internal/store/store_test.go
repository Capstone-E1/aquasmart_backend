package store

import (
	"math"
	"testing"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
)

func abs(f float64) float64 {
	return math.Abs(f)
}

func TestStore_FiltrationProcess_Basic(t *testing.T) {
	store := NewStore(100)

	// Initially no process should exist
	_, exists := store.GetFiltrationProcess()
	if exists {
		t.Error("Expected no filtration process initially")
	}

	// Should be able to change mode when no process
	canChange, reason := store.CanChangeFilterMode()
	if !canChange {
		t.Errorf("Expected to be able to change mode initially, got reason: %s", reason)
	}
}

func TestStore_StartFiltrationProcess(t *testing.T) {
	store := NewStore(100)
	mode := models.FilterModeDrinking
	targetVolume := 5.0

	// Start filtration process
	store.StartFiltrationProcess(mode, targetVolume)

	// Should now have a process
	process, exists := store.GetFiltrationProcess()
	if !exists {
		t.Fatal("Expected filtration process to exist after starting")
	}

	if process.State != models.FiltrationStateProcessing {
		t.Errorf("Expected state to be %v, got %v", models.FiltrationStateProcessing, process.State)
	}

	if process.CurrentMode != mode {
		t.Errorf("Expected mode to be %v, got %v", mode, process.CurrentMode)
	}

	if process.TargetVolume != targetVolume {
		t.Errorf("Expected target volume to be %v, got %v", targetVolume, process.TargetVolume)
	}

	// Current filter mode should be updated
	if store.GetCurrentFilterMode() != mode {
		t.Errorf("Expected current filter mode to be %v, got %v", mode, store.GetCurrentFilterMode())
	}
}

func TestStore_CanChangeFilterMode_WithActiveProcess(t *testing.T) {
	store := NewStore(100)

	// Start a process
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)

	// Initially should not be able to change (can't interrupt early)
	canChange, reason := store.CanChangeFilterMode()
	if canChange {
		t.Error("Expected to NOT be able to change mode during early filtration")
	}
	if reason != "filtration_in_progress" {
		t.Errorf("Expected reason 'filtration_in_progress', got '%s'", reason)
	}

	// Simulate progress to make it interruptible
	process, _ := store.GetFiltrationProcess()
	process.ProcessedVolume = 0.6 // 12% of 5L
	process.UpdateProgress(2.5)
	store.SetFiltrationProcess(process)

	// Now should be able to change (interruptible)
	canChange, reason = store.CanChangeFilterMode()
	if !canChange {
		t.Error("Expected to be able to change mode after 10% progress")
	}
	if reason != "filtration_interruptible" {
		t.Errorf("Expected reason 'filtration_interruptible', got '%s'", reason)
	}
}

func TestStore_UpdateFiltrationProgress(t *testing.T) {
	store := NewStore(100)

	// Start process
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)

	// Update with flow rate
	flowRate := 2.5
	store.UpdateFiltrationProgress(flowRate)

	// Get updated process
	process, exists := store.GetFiltrationProcess()
	if !exists {
		t.Fatal("Expected process to still exist after update")
	}

	if process.CurrentFlowRate != flowRate {
		t.Errorf("Expected flow rate to be %v, got %v", flowRate, process.CurrentFlowRate)
	}

	// Manually set processed volume to test progress calculation
	process.ProcessedVolume = 2.5
	process.UpdateProgress(flowRate)
	store.SetFiltrationProcess(process)

	// Check progress
	updatedProcess, _ := store.GetFiltrationProcess()
	expectedProgress := (2.5 / 5.0) * 100 // 50%
	if abs(updatedProcess.Progress-expectedProgress) > 0.01 {
		t.Errorf("Expected progress to be ~%v, got %v", expectedProgress, updatedProcess.Progress)
	}
}

func TestStore_CompleteFiltrationProcess(t *testing.T) {
	store := NewStore(100)

	// Start process
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)

	// Complete the process
	store.CompleteFiltrationProcess()

	// Check state
	process, exists := store.GetFiltrationProcess()
	if !exists {
		t.Fatal("Expected process to exist after completion")
	}

	if process.State != models.FiltrationStateCompleted {
		t.Errorf("Expected state to be %v, got %v", models.FiltrationStateCompleted, process.State)
	}

	if process.Progress != 100.0 {
		t.Errorf("Expected progress to be 100.0, got %v", process.Progress)
	}

	// Should be able to change mode after completion
	canChange, reason := store.CanChangeFilterMode()
	if !canChange {
		t.Errorf("Expected to be able to change mode after completion, got reason: %s", reason)
	}
}

func TestStore_ClearCompletedProcess(t *testing.T) {
	store := NewStore(100)

	// Start and complete process
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)
	store.CompleteFiltrationProcess()

	// Clear completed process
	store.ClearCompletedProcess()

	// Should no longer exist
	_, exists := store.GetFiltrationProcess()
	if exists {
		t.Error("Expected no process after clearing completed process")
	}
}

func TestStore_ClearCompletedProcess_OnlyWhenCompleted(t *testing.T) {
	store := NewStore(100)

	// Start process but don't complete
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)

	// Try to clear (should not work)
	store.ClearCompletedProcess()

	// Should still exist
	process, exists := store.GetFiltrationProcess()
	if !exists {
		t.Error("Expected process to still exist when not completed")
	}

	if process.State != models.FiltrationStateProcessing {
		t.Errorf("Expected state to be %v, got %v", models.FiltrationStateProcessing, process.State)
	}
}

func TestStore_SetFiltrationProcess_WithNil(t *testing.T) {
	store := NewStore(100)

	// Start a process
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)

	// Set to nil
	store.SetFiltrationProcess(nil)

	// Should no longer exist
	_, exists := store.GetFiltrationProcess()
	if exists {
		t.Error("Expected no process after setting to nil")
	}
}

func TestStore_GetFiltrationProcess_ReturnsCopy(t *testing.T) {
	store := NewStore(100)

	// Start process
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)

	// Get process and modify it
	process1, _ := store.GetFiltrationProcess()
	process1.ProcessedVolume = 999.0

	// Get process again - should not be modified
	process2, _ := store.GetFiltrationProcess()
	if process2.ProcessedVolume == 999.0 {
		t.Error("Expected GetFiltrationProcess to return a copy, not reference")
	}

	if process2.ProcessedVolume != 0.0 {
		t.Errorf("Expected original processed volume to be 0.0, got %v", process2.ProcessedVolume)
	}
}

func TestStore_FiltrationProcess_ConcurrentAccess(t *testing.T) {
	store := NewStore(100)

	// Start process
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)

	// Test concurrent reads and writes
	done := make(chan bool, 2)

	// Goroutine 1: Update progress
	go func() {
		for i := 0; i < 10; i++ {
			store.UpdateFiltrationProgress(2.5)
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 2: Read process state
	go func() {
		for i := 0; i < 10; i++ {
			_, exists := store.GetFiltrationProcess()
			if !exists {
				t.Error("Expected process to exist during concurrent access")
			}
			store.CanChangeFilterMode()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final state is still valid
	process, exists := store.GetFiltrationProcess()
	if !exists {
		t.Error("Expected process to exist after concurrent access")
	}

	if process.State != models.FiltrationStateProcessing {
		t.Errorf("Expected state to still be processing after concurrent access, got %v", process.State)
	}
}

func TestStore_FiltrationProcess_StateTransitions(t *testing.T) {
	store := NewStore(100)

	// Test complete workflow
	// 1. Start process
	store.StartFiltrationProcess(models.FilterModeDrinking, 5.0)

	process, _ := store.GetFiltrationProcess()
	if process.State != models.FiltrationStateProcessing {
		t.Errorf("Expected initial state to be processing, got %v", process.State)
	}

	// 2. Update progress to make interruptible
	process.ProcessedVolume = 0.6
	process.UpdateProgress(2.5)
	store.SetFiltrationProcess(process)

	canChange, reason := store.CanChangeFilterMode()
	if !canChange || reason != "filtration_interruptible" {
		t.Error("Expected process to be interruptible after 10% progress")
	}

	// 3. Switch to switching state (simulating force mode change)
	process.State = models.FiltrationStateSwitching
	store.SetFiltrationProcess(process)

	canChange, reason = store.CanChangeFilterMode()
	if canChange {
		t.Error("Expected NOT to be able to change mode during switching")
	}
	if reason != "mode_change_in_progress" {
		t.Errorf("Expected reason 'mode_change_in_progress', got '%s'", reason)
	}

	// 4. Complete process
	store.CompleteFiltrationProcess()

	process, _ = store.GetFiltrationProcess()
	if process.State != models.FiltrationStateCompleted {
		t.Errorf("Expected state to be completed, got %v", process.State)
	}

	canChange, _ = store.CanChangeFilterMode()
	if !canChange {
		t.Error("Expected to be able to change mode after completion")
	}

	// 5. Clear completed process
	store.ClearCompletedProcess()

	_, exists := store.GetFiltrationProcess()
	if exists {
		t.Error("Expected no process after clearing completed")
	}
}