package services

import (
	"log"
	"sync"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
	"github.com/Capstone-E1/aquasmart_backend/internal/mqtt"
	"github.com/Capstone-E1/aquasmart_backend/internal/store"
)

// Scheduler manages automated filter mode scheduling
type Scheduler struct {
	store            store.DataStore
	ticker           *time.Ticker
	stopChan         chan bool
	mu               sync.RWMutex
	isRunning        bool
	currentExecution *models.ScheduleExecution
	mqttClient       *mqtt.Client
}

// NewScheduler creates a new scheduler instance
func NewScheduler(dataStore store.DataStore, mqttClient *mqtt.Client) *Scheduler {
	return &Scheduler{
		store:      dataStore,
		stopChan:   make(chan bool),
		mqttClient: mqttClient,
	}
}

// Start begins the scheduler background process
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		log.Println("⚠️  Scheduler: Already running")
		return
	}

	// Check every minute for schedules to execute
	s.ticker = time.NewTicker(1 * time.Minute)
	s.isRunning = true

	log.Println("🕐 Scheduler: Started - checking schedules every minute")

	go s.run()
}

// Stop halts the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	s.ticker.Stop()
	s.stopChan <- true
	s.isRunning = false

	log.Println("🛑 Scheduler: Stopped")
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	// Check immediately on start
	s.checkAndExecuteSchedules()

	for {
		select {
		case <-s.ticker.C:
			s.checkAndExecuteSchedules()
		case <-s.stopChan:
			return
		}
	}
}

// checkAndExecuteSchedules checks for active schedules and executes them
func (s *Scheduler) checkAndExecuteSchedules() {
	// Get all active schedules
	schedules, err := s.store.GetAllSchedules(true) // only active schedules
	if err != nil {
		log.Printf("❌ Scheduler: Failed to get schedules: %v", err)
		return
	}

	if len(schedules) == 0 {
		return
	}

	now := time.Now()
	executed := 0

	for _, schedule := range schedules {
		// Check if schedule should execute now
		if schedule.ShouldExecuteNow() {
			// Check if already executing
			if s.isScheduleCurrentlyExecuting(schedule.ID) {
				continue
			}

			// Check if was already executed recently (within last minute)
			if s.wasRecentlyExecuted(schedule.ID) {
				continue
			}

			// Execute the schedule
			if err := s.executeSchedule(&schedule); err != nil {
				log.Printf("❌ Scheduler: Failed to execute schedule '%s': %v", schedule.Name, err)
			} else {
				executed++
			}
		}
	}

	if executed > 0 {
		log.Printf("✅ Scheduler: Executed %d schedule(s) at %s", executed, now.Format("15:04:05"))
	}
}

// executeSchedule executes a single schedule
func (s *Scheduler) executeSchedule(schedule *models.FilterSchedule) error {
	log.Printf("🔄 Scheduler: Executing schedule '%s' - Mode: %s", schedule.Name, schedule.FilterMode)

	// Create execution record
	execution := &models.ScheduleExecution{
		ScheduleID: schedule.ID,
		ExecutedAt: time.Now(),
		Status:     "running",
	}

	// Save execution record
	if err := s.store.CreateScheduleExecution(execution); err != nil {
		return err
	}

	// Store current execution
	s.mu.Lock()
	s.currentExecution = execution
	s.mu.Unlock()

	// Change filter mode in the database
	s.store.SetCurrentFilterMode(schedule.FilterMode)

	// Publish filter command via MQTT
	if s.mqttClient != nil {
		if err := s.mqttClient.PublishFilterCommand(schedule.FilterMode); err != nil {
			log.Printf("❌ Scheduler: Failed to publish MQTT filter command: %v", err)
			// Do not return error, just log it, as the main action (DB update) succeeded
		}
	}

	log.Printf("✅ Scheduler: Successfully set filter mode to '%s' for schedule '%s'", schedule.FilterMode, schedule.Name)

	// Schedule completion after duration
	go s.completeScheduleAfterDuration(execution, schedule)

	return nil
}

// completeScheduleAfterDuration marks the execution as completed after the scheduled duration
func (s *Scheduler) completeScheduleAfterDuration(execution *models.ScheduleExecution, schedule *models.FilterSchedule) {
	duration := time.Duration(schedule.DurationMinutes) * time.Minute
	time.Sleep(duration)

	// Mark as completed
	execution.Status = "completed"
	execution.CompletedAt = timePtr(time.Now())

	if err := s.store.UpdateScheduleExecution(execution); err != nil {
		log.Printf("❌ Scheduler: Failed to update execution status: %v", err)
	}

	s.mu.Lock()
	if s.currentExecution != nil && s.currentExecution.ID == execution.ID {
		s.currentExecution = nil
	}
	s.mu.Unlock()

	log.Printf("✅ Scheduler: Schedule '%s' completed after %d minutes", schedule.Name, schedule.DurationMinutes)
}

// isScheduleCurrentlyExecuting checks if a schedule is currently being executed
func (s *Scheduler) isScheduleCurrentlyExecuting(scheduleID int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.currentExecution == nil {
		return false
	}

	return s.currentExecution.ScheduleID == scheduleID && s.currentExecution.Status == "running"
}

// wasRecentlyExecuted checks if schedule was executed within the last minute
func (s *Scheduler) wasRecentlyExecuted(scheduleID int) bool {
	// Get recent executions for this schedule
	executions, err := s.store.GetScheduleExecutions(scheduleID, 1)
	if err != nil || len(executions) == 0 {
		return false
	}

	lastExecution := executions[0]
	timeSince := time.Since(lastExecution.ExecutedAt)

	// Consider recently executed if within last 2 minutes
	return timeSince < 2*time.Minute
}

// HandleManualOverride is called when user manually changes filter mode
func (s *Scheduler) HandleManualOverride(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentExecution == nil {
		return
	}

	// Mark current execution as overridden
	s.currentExecution.Status = "overridden"
	s.currentExecution.OverrideReason = reason
	s.currentExecution.CompletedAt = timePtr(time.Now())

	if err := s.store.UpdateScheduleExecution(s.currentExecution); err != nil {
		log.Printf("❌ Scheduler: Failed to mark execution as overridden: %v", err)
	} else {
		log.Printf("⚠️  Scheduler: Current schedule execution overridden - Reason: %s", reason)
	}

	s.currentExecution = nil
}

// GetCurrentExecution returns the currently running schedule execution
func (s *Scheduler) GetCurrentExecution() *models.ScheduleExecution {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentExecution
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// timePtr returns a pointer to a time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}
