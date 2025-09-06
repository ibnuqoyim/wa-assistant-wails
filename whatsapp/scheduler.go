package whatsapp

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// ScheduledTaskType represents the type of scheduled task
type ScheduledTaskType string

const (
	TaskTypeMessage ScheduledTaskType = "message"
	TaskTypeStatus  ScheduledTaskType = "status"
	TaskTypeStory   ScheduledTaskType = "story"
)

// ScheduledTaskStatus represents the status of a scheduled task
type ScheduledTaskStatus string

const (
	TaskStatusPending   ScheduledTaskStatus = "pending"
	TaskStatusRunning   ScheduledTaskStatus = "running"
	TaskStatusCompleted ScheduledTaskStatus = "completed"
	TaskStatusFailed    ScheduledTaskStatus = "failed"
	TaskStatusCancelled ScheduledTaskStatus = "cancelled"
)

// ScheduledTask represents a scheduled task
type ScheduledTask struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Type       ScheduledTaskType   `json:"type"`
	Status     ScheduledTaskStatus `json:"status"`
	CronExpr   string              `json:"cronExpr"`
	Recipients []string            `json:"recipients"`
	Content    TaskContent         `json:"content"`
	CreatedAt  time.Time           `json:"createdAt"`
	UpdatedAt  time.Time           `json:"updatedAt"`
	NextRun    *time.Time          `json:"nextRun,omitempty"`
	LastRun    *time.Time          `json:"lastRun,omitempty"`
	RunCount   int                 `json:"runCount"`
	MaxRuns    int                 `json:"maxRuns,omitempty"` // 0 means unlimited
	IsActive   bool                `json:"isActive"`
	ErrorMsg   string              `json:"errorMsg,omitempty"`
}

// TaskContent represents the content of a scheduled task
type TaskContent struct {
	Text        string            `json:"text,omitempty"`
	MediaPath   string            `json:"mediaPath,omitempty"`
	MediaType   string            `json:"mediaType,omitempty"` // image, video, audio, document
	Caption     string            `json:"caption,omitempty"`
	Variables   map[string]string `json:"variables,omitempty"`  // For template variables
	StatusType  string            `json:"statusType,omitempty"` // text, image, video for status
	StoryConfig *StoryConfig      `json:"storyConfig,omitempty"`
}

// StoryConfig represents configuration for story/status posts
type StoryConfig struct {
	BackgroundColor string `json:"backgroundColor,omitempty"`
	Font            string `json:"font,omitempty"`
	FontSize        int    `json:"fontSize,omitempty"`
	TextColor       string `json:"textColor,omitempty"`
	Duration        int    `json:"duration,omitempty"` // in seconds
}

// TaskExecution represents a single execution of a scheduled task
type TaskExecution struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"taskId"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Results   []string  `json:"results,omitempty"` // Message IDs or status IDs
}

// Scheduler manages scheduled tasks
type Scheduler struct {
	cron      *cron.Cron
	tasks     map[string]*ScheduledTask
	tasksMux  sync.RWMutex
	manager   *Manager
	logger    *log.Logger
	isRunning bool
}

// NewScheduler creates a new scheduler instance
func NewScheduler(manager *Manager, logger *log.Logger) *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()),
		tasks:   make(map[string]*ScheduledTask),
		manager: manager,
		logger:  logger,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	s.cron.Start()
	s.isRunning = true
	s.logger.Println("Scheduler started")
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if !s.isRunning {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.isRunning = false
	s.logger.Println("Scheduler stopped")
}

// AddTask adds a new scheduled task
func (s *Scheduler) AddTask(task *ScheduledTask) error {
	s.tasksMux.Lock()
	defer s.tasksMux.Unlock()

	// Generate ID if not provided
	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}

	// Set timestamps
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now
	task.Status = TaskStatusPending
	task.IsActive = true

	// Validate cron expression
	schedule, err := cron.ParseStandard(task.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %v", err)
	}

	// Calculate next run time
	nextRun := schedule.Next(now)
	task.NextRun = &nextRun

	// Add to cron scheduler
	cronID, err := s.cron.AddFunc(task.CronExpr, func() {
		s.executeTask(task.ID)
	})
	if err != nil {
		return fmt.Errorf("failed to add task to cron: %v", err)
	}

	// Store task
	s.tasks[task.ID] = task
	s.logger.Printf("Added scheduled task: %s (ID: %s, Cron ID: %d)", task.Name, task.ID, cronID)

	return nil
}

// RemoveTask removes a scheduled task
func (s *Scheduler) RemoveTask(taskID string) error {
	s.tasksMux.Lock()
	defer s.tasksMux.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Mark as cancelled
	task.Status = TaskStatusCancelled
	task.IsActive = false
	task.UpdatedAt = time.Now()

	s.logger.Printf("Removed scheduled task: %s (ID: %s)", task.Name, taskID)
	return nil
}

// GetTask retrieves a scheduled task by ID
func (s *Scheduler) GetTask(taskID string) (*ScheduledTask, error) {
	s.tasksMux.RLock()
	defer s.tasksMux.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// GetAllTasks retrieves all scheduled tasks
func (s *Scheduler) GetAllTasks() []*ScheduledTask {
	s.tasksMux.RLock()
	defer s.tasksMux.RUnlock()

	tasks := make([]*ScheduledTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// UpdateTask updates an existing scheduled task
func (s *Scheduler) UpdateTask(taskID string, updatedTask *ScheduledTask) error {
	s.tasksMux.Lock()
	defer s.tasksMux.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Update fields
	task.Name = updatedTask.Name
	task.Type = updatedTask.Type
	task.CronExpr = updatedTask.CronExpr
	task.Recipients = updatedTask.Recipients
	task.Content = updatedTask.Content
	task.MaxRuns = updatedTask.MaxRuns
	task.IsActive = updatedTask.IsActive
	task.UpdatedAt = time.Now()

	// Validate new cron expression
	schedule, err := cron.ParseStandard(task.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %v", err)
	}

	// Update next run time
	nextRun := schedule.Next(time.Now())
	task.NextRun = &nextRun

	s.logger.Printf("Updated scheduled task: %s (ID: %s)", task.Name, taskID)
	return nil
}

// executeTask executes a scheduled task
func (s *Scheduler) executeTask(taskID string) {
	s.tasksMux.Lock()
	task, exists := s.tasks[taskID]
	if !exists || !task.IsActive {
		s.tasksMux.Unlock()
		return
	}

	// Check if max runs reached
	if task.MaxRuns > 0 && task.RunCount >= task.MaxRuns {
		task.Status = TaskStatusCompleted
		task.IsActive = false
		task.UpdatedAt = time.Now()
		s.tasksMux.Unlock()
		s.logger.Printf("Task %s reached max runs (%d)", taskID, task.MaxRuns)
		return
	}

	// Update task status
	task.Status = TaskStatusRunning
	now := time.Now()
	task.LastRun = &now
	task.RunCount++
	task.UpdatedAt = now
	s.tasksMux.Unlock()

	s.logger.Printf("Executing task: %s (ID: %s, Run: %d)", task.Name, taskID, task.RunCount)

	// Execute the task
	err := s.performTask(task)

	// Update task status after execution
	s.tasksMux.Lock()
	if err != nil {
		task.Status = TaskStatusFailed
		task.ErrorMsg = err.Error()
		s.logger.Printf("Task execution failed: %s - %v", taskID, err)
	} else {
		task.Status = TaskStatusPending // Ready for next run
		task.ErrorMsg = ""
		s.logger.Printf("Task executed successfully: %s", taskID)
	}

	// Calculate next run time
	schedule, _ := cron.ParseStandard(task.CronExpr)
	nextRun := schedule.Next(time.Now())
	task.NextRun = &nextRun
	task.UpdatedAt = time.Now()
	s.tasksMux.Unlock()
}

// performTask performs the actual task execution
func (s *Scheduler) performTask(task *ScheduledTask) error {
	if s.manager == nil || s.manager.client == nil || !s.manager.client.IsConnected() {
		return fmt.Errorf("WhatsApp client not connected")
	}

	switch task.Type {
	case TaskTypeMessage:
		return s.sendScheduledMessage(task)
	case TaskTypeStatus:
		return s.sendScheduledStatus(task)
	case TaskTypeStory:
		return s.sendScheduledStory(task)
	default:
		return fmt.Errorf("unknown task type: %s", task.Type)
	}
}

// sendScheduledMessage sends a scheduled message
func (s *Scheduler) sendScheduledMessage(task *ScheduledTask) error {
	content := s.processContent(task.Content)

	for _, recipient := range task.Recipients {
		err := s.manager.SendMessage(recipient, content.Text)
		if err != nil {
			s.logger.Printf("Failed to send message to %s: %v", recipient, err)
			return err
		}
		s.logger.Printf("Sent scheduled message to %s", recipient)
	}

	return nil
}

// sendScheduledStatus sends a scheduled status/story
func (s *Scheduler) sendScheduledStatus(task *ScheduledTask) error {
	// Note: WhatsApp status/story sending requires specific implementation
	// This is a placeholder for the actual implementation
	s.logger.Printf("Sending scheduled status: %s", task.Content.Text)

	// TODO: Implement actual status sending logic
	// This would involve creating a status message and broadcasting it

	return nil
}

// sendScheduledStory sends a scheduled story
func (s *Scheduler) sendScheduledStory(task *ScheduledTask) error {
	// Note: Stories and status are essentially the same in WhatsApp
	return s.sendScheduledStatus(task)
}

// processContent processes task content, replacing variables
func (s *Scheduler) processContent(content TaskContent) TaskContent {
	processedContent := content

	// Replace variables in text
	if content.Variables != nil {
		text := content.Text
		for key, value := range content.Variables {
			placeholder := fmt.Sprintf("{{%s}}", key)
			text = fmt.Sprintf("%s %s", placeholder, value)
		}
		processedContent.Text = text
	}

	// Add timestamp variables
	now := time.Now()
	processedContent.Text = fmt.Sprintf(processedContent.Text,
		"{{date}}", now.Format("2006-01-02"),
		"{{time}}", now.Format("15:04:05"),
		"{{datetime}}", now.Format("2006-01-02 15:04:05"),
	)

	return processedContent
}

// GetTaskStats returns statistics about scheduled tasks
func (s *Scheduler) GetTaskStats() map[string]interface{} {
	s.tasksMux.RLock()
	defer s.tasksMux.RUnlock()

	stats := map[string]interface{}{
		"total":     len(s.tasks),
		"active":    0,
		"pending":   0,
		"running":   0,
		"completed": 0,
		"failed":    0,
		"cancelled": 0,
	}

	for _, task := range s.tasks {
		if task.IsActive {
			stats["active"] = stats["active"].(int) + 1
		}

		switch task.Status {
		case TaskStatusPending:
			stats["pending"] = stats["pending"].(int) + 1
		case TaskStatusRunning:
			stats["running"] = stats["running"].(int) + 1
		case TaskStatusCompleted:
			stats["completed"] = stats["completed"].(int) + 1
		case TaskStatusFailed:
			stats["failed"] = stats["failed"].(int) + 1
		case TaskStatusCancelled:
			stats["cancelled"] = stats["cancelled"].(int) + 1
		}
	}

	return stats
}
