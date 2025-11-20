package models

import (
	"fmt"
	"strings"
	"time"
)

// FilterSchedule represents an automated filter mode schedule
type FilterSchedule struct {
	ID              int        `json:"id"`
	Name            string     `json:"name"`
	FilterMode      FilterMode `json:"filter_mode"`
	StartTime       string     `json:"start_time"`      // Format: "HH:MM:SS"
	DurationMinutes int        `json:"duration_minutes"`
	DaysOfWeek      []string   `json:"days_of_week"`    // e.g., ["monday", "tuesday"]
	IsActive        bool       `json:"is_active"`
	Timezone        string     `json:"timezone"`        // IANA Time Zone name
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ScheduleExecution represents a single execution of a schedule
type ScheduleExecution struct {
	ID             int       `json:"id"`
	ScheduleID     int       `json:"schedule_id"`
	ExecutedAt     time.Time `json:"executed_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	Status         string    `json:"status"` // 'running', 'completed', 'overridden', 'failed', 'cancelled'
	OverrideReason string    `json:"override_reason,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ScheduleWithExecution includes schedule info with its current/last execution
type ScheduleWithExecution struct {
	FilterSchedule
	CurrentExecution *ScheduleExecution `json:"current_execution,omitempty"`
	NextExecution    *time.Time         `json:"next_execution,omitempty"`
}

// CreateScheduleRequest represents the request body for creating a schedule
type CreateScheduleRequest struct {
	Name            string     `json:"name" validate:"required,min=3,max=100"`
	FilterMode      FilterMode `json:"filter_mode" validate:"required"`
	StartTime       string     `json:"start_time" validate:"required"`      // Format: "HH:MM" or "HH:MM:SS"
	DurationMinutes int        `json:"duration_minutes" validate:"required,min=1,max=1440"`
	DaysOfWeek      []string   `json:"days_of_week" validate:"required,min=1"`
	IsActive        bool       `json:"is_active"`
	Timezone        string     `json:"timezone" validate:"required"`        // IANA Time Zone name
}

// UpdateScheduleRequest represents the request body for updating a schedule
type UpdateScheduleRequest struct {
	Name            *string     `json:"name,omitempty"`
	FilterMode      *FilterMode `json:"filter_mode,omitempty"`
	StartTime       *string     `json:"start_time,omitempty"`
	DurationMinutes *int        `json:"duration_minutes,omitempty"`
	DaysOfWeek      []string    `json:"days_of_week,omitempty"`
	IsActive        *bool       `json:"is_active,omitempty"`
	Timezone        *string     `json:"timezone,omitempty"`      // IANA Time Zone name
}

// ValidDaysOfWeek contains all valid day names
var ValidDaysOfWeek = map[string]bool{
	"monday":    true,
	"tuesday":   true,
	"wednesday": true,
	"thursday":  true,
	"friday":    true,
	"saturday":  true,
	"sunday":    true,
}

// Validate validates the create schedule request
func (r *CreateScheduleRequest) Validate() error {
	// Validate name
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Name) < 3 || len(r.Name) > 100 {
		return fmt.Errorf("name must be between 3 and 100 characters")
	}

	// Validate filter mode
	if r.FilterMode != FilterModeDrinking && r.FilterMode != FilterModeHousehold {
		return fmt.Errorf("filter_mode must be 'drinking_water' or 'household_water'")
	}

	// Validate start time format
	if !isValidTimeFormat(r.StartTime) {
		return fmt.Errorf("start_time must be in HH:MM or HH:MM:SS format")
	}

	// Validate duration
	if r.DurationMinutes < 1 || r.DurationMinutes > 1440 {
		return fmt.Errorf("duration_minutes must be between 1 and 1440 (24 hours)")
	}

	// Validate days of week
	if len(r.DaysOfWeek) == 0 {
		return fmt.Errorf("days_of_week must contain at least one day")
	}

	for _, day := range r.DaysOfWeek {
		dayLower := strings.ToLower(strings.TrimSpace(day))
		if !ValidDaysOfWeek[dayLower] {
			return fmt.Errorf("invalid day of week: %s", day)
		}
	}

	// Validate timezone
	if !isValidTimezone(r.Timezone) {
		return fmt.Errorf("invalid timezone: %s", r.Timezone)
	}

	return nil
}

// isValidTimezone checks if a string is a valid IANA Time Zone database name
func isValidTimezone(tz string) bool {
	if tz == "" {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

// Validate validates the update schedule request
func (r *UpdateScheduleRequest) Validate() error {
	// Validate name if provided
	if r.Name != nil {
		if strings.TrimSpace(*r.Name) == "" {
			return fmt.Errorf("name cannot be empty")
		}
		if len(*r.Name) < 3 || len(*r.Name) > 100 {
			return fmt.Errorf("name must be between 3 and 100 characters")
		}
	}

	// Validate filter mode if provided
	if r.FilterMode != nil {
		if *r.FilterMode != FilterModeDrinking && *r.FilterMode != FilterModeHousehold {
			return fmt.Errorf("filter_mode must be 'drinking_water' or 'household_water'")
		}
	}

	// Validate start time if provided
	if r.StartTime != nil && !isValidTimeFormat(*r.StartTime) {
		return fmt.Errorf("start_time must be in HH:MM or HH:MM:SS format")
	}

	// Validate duration if provided
	if r.DurationMinutes != nil {
		if *r.DurationMinutes < 1 || *r.DurationMinutes > 1440 {
			return fmt.Errorf("duration_minutes must be between 1 and 1440 (24 hours)")
		}
	}

	// Validate days of week if provided
	if r.DaysOfWeek != nil {
		if len(r.DaysOfWeek) == 0 {
			return fmt.Errorf("days_of_week must contain at least one day")
		}
		for _, day := range r.DaysOfWeek {
			dayLower := strings.ToLower(strings.TrimSpace(day))
			if !ValidDaysOfWeek[dayLower] {
				return fmt.Errorf("invalid day of week: %s", day)
			}
		}
	}

	// Validate timezone if provided
	if r.Timezone != nil && !isValidTimezone(*r.Timezone) {
		return fmt.Errorf("invalid timezone: %s", *r.Timezone)
	}

	return nil
}

// NormalizeDaysOfWeek converts day names to lowercase
func NormalizeDaysOfWeek(days []string) []string {
	normalized := make([]string, len(days))
	for i, day := range days {
		normalized[i] = strings.ToLower(strings.TrimSpace(day))
	}
	return normalized
}

// isValidTimeFormat checks if time string is in HH:MM or HH:MM:SS format
func isValidTimeFormat(timeStr string) bool {
	// Try parsing as HH:MM:SS
	if _, err := time.Parse("15:04:05", timeStr); err == nil {
		return true
	}
	// Try parsing as HH:MM
	if _, err := time.Parse("15:04", timeStr); err == nil {
		return true
	}
	return false
}

// NormalizeTimeFormat converts HH:MM to HH:MM:SS format
func NormalizeTimeFormat(timeStr string) string {
	// If already in HH:MM:SS format, return as is
	if _, err := time.Parse("15:04:05", timeStr); err == nil {
		return timeStr
	}
	// If in HH:MM format, add :00
	if _, err := time.Parse("15:04", timeStr); err == nil {
		return timeStr + ":00"
	}
	return timeStr
}

// IsScheduledForToday checks if the schedule should run today
func (s *FilterSchedule) IsScheduledForToday() bool {
	today := strings.ToLower(time.Now().Weekday().String())
	for _, day := range s.DaysOfWeek {
		if strings.ToLower(day) == today {
			return true
		}
	}
	return false
}

// GetEndTime calculates the end time based on start time and duration
func (s *FilterSchedule) GetEndTime() (time.Time, error) {
	// Parse start time
	startTime, err := time.Parse("15:04:05", s.StartTime)
	if err != nil {
		return time.Time{}, err
	}

	// Add duration
	endTime := startTime.Add(time.Duration(s.DurationMinutes) * time.Minute)
	return endTime, nil
}

// ShouldExecuteNow checks if the schedule should be executed at the current time
func (s *FilterSchedule) ShouldExecuteNow() bool {
	if !s.IsActive {
		return false
	}

	if !s.IsScheduledForToday() {
		return false
	}

	now := time.Now()
	currentTime := now.Format("15:04:05")

	// Parse schedule start time
	scheduleTime, err := time.Parse("15:04:05", s.StartTime)
	if err != nil {
		return false
	}

	// Parse current time
	nowTime, err := time.Parse("15:04:05", currentTime)
	if err != nil {
		return false
	}

	// Calculate end time
	endTime := scheduleTime.Add(time.Duration(s.DurationMinutes) * time.Minute)

	// Check if current time is within schedule window
	return nowTime.After(scheduleTime) && nowTime.Before(endTime) || nowTime.Equal(scheduleTime)
}

// CalculateNextExecution calculates when this schedule will next execute in UTC
func (s *FilterSchedule) CalculateNextExecution() *time.Time {
	if !s.IsActive || len(s.DaysOfWeek) == 0 {
		return nil
	}

	// Load the schedule's timezone
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		// If timezone is invalid, cannot calculate next execution
		return nil
	}

	// Get the current time in the schedule's timezone
	nowInLoc := time.Now().In(loc)

	// Parse schedule start time (without date)
	startTime, err := time.Parse("15:04:05", s.StartTime)
	if err != nil {
		return nil
	}

	// Function to create a schedule time for a given day
	createScheduleTime := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(),
			startTime.Hour(), startTime.Minute(), startTime.Second(), 0, loc)
	}

	// Check for today
	todaySchedule := createScheduleTime(nowInLoc)
	if todaySchedule.After(nowInLoc) {
		isScheduledToday := false
		currentWeekday := strings.ToLower(nowInLoc.Weekday().String())
		for _, day := range s.DaysOfWeek {
			if strings.ToLower(day) == currentWeekday {
				isScheduledToday = true
				break
			}
		}
		if isScheduledToday {
			utcExecution := todaySchedule.UTC()
			return &utcExecution
		}
	}

	// Find the next scheduled day
	for i := 1; i <= 7; i++ {
		nextDay := nowInLoc.AddDate(0, 0, i)
		nextWeekday := strings.ToLower(nextDay.Weekday().String())

		for _, day := range s.DaysOfWeek {
			if strings.ToLower(day) == nextWeekday {
				nextSchedule := createScheduleTime(nextDay)
				utcExecution := nextSchedule.UTC()
				return &utcExecution
			}
		}
	}

	return nil
}

// GetStatusMessage returns a human-readable status message
func (e *ScheduleExecution) GetStatusMessage() string {
	switch e.Status {
	case "running":
		return "Schedule is currently running"
	case "completed":
		return "Schedule completed successfully"
	case "overridden":
		msg := "Schedule was overridden"
		if e.OverrideReason != "" {
			msg += ": " + e.OverrideReason
		}
		return msg
	case "failed":
		return "Schedule execution failed"
	case "cancelled":
		return "Schedule was cancelled"
	default:
		return "Unknown status"
	}
}
