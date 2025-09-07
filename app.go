package main

import (
	"context"
	"fmt"
	"log"

	"wa-bot-wails/whatsapp"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx       context.Context
	waManager *whatsapp.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	waManager, err := whatsapp.NewManager("./whatsapp/storage/whatsapp.db")
	if err != nil {
		log.Printf("Failed to create WhatsApp manager: %v", err)
		return &App{}
	}

	return &App{
		waManager: waManager,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Test event emission to verify frontend communication
	fmt.Println("App startup - testing event emission...")
	runtime.EventsEmit(ctx, "app:startup", "Application started successfully")

	// Start listening for WhatsApp events
	if a.waManager != nil {
		fmt.Println("Starting WhatsApp event listener...")
		go a.listenForWhatsAppEvents()
	} else {
		fmt.Println("WhatsApp manager is nil!")
	}
}

// listenForWhatsAppEvents listens for events from WhatsApp manager and emits them to frontend
func (a *App) listenForWhatsAppEvents() {
	eventChan := a.waManager.GetEventChannel()
	if eventChan == nil {
		return
	}

	for event := range eventChan {
		switch event.Type {
		case "qr":
			fmt.Println("Received QR code event:", event.Data)
			fmt.Println("Emitting whatsapp:qr event to frontend...")
			runtime.EventsEmit(a.ctx, "whatsapp:qr", event.Data)
			fmt.Println("Event emitted successfully")
		case "connected":
			runtime.EventsEmit(a.ctx, "whatsapp:connected", event.Message)
		case "disconnected":
			runtime.EventsEmit(a.ctx, "whatsapp:disconnected", event.Message)
		case "error":
			runtime.EventsEmit(a.ctx, "whatsapp:error", event.Message)
		}
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// CheckWhatsAppConnection checks if there's an existing WhatsApp device connection
func (a *App) CheckWhatsAppConnection() (*whatsapp.ConnectionStatus, error) {
	if a.waManager == nil {
		return &whatsapp.ConnectionStatus{IsConnected: false}, fmt.Errorf("WhatsApp manager not initialized")
	}

	return a.waManager.CheckExistingDevice()
}

// ConnectExistingDevice attempts to reconnect using existing device credentials
func (a *App) ConnectExistingDevice() error {
	if a.waManager == nil {
		return fmt.Errorf("WhatsApp manager not initialized")
	}

	return a.waManager.ConnectExistingDevice()
}

// StartNewConnection starts a new WhatsApp connection (will generate QR code)
func (a *App) StartNewConnection() error {
	if a.waManager == nil {
		return fmt.Errorf("WhatsApp manager not initialized")
	}

	fmt.Println("App: Starting new WhatsApp connection...")
	err := a.waManager.StartNewConnection()
	if err != nil {
		fmt.Printf("App: Error starting connection: %v\n", err)
	} else {
		fmt.Println("App: StartNewConnection completed successfully")
	}
	return err
}

// RequestPairingCode requests a pairing code for phone number authentication
func (a *App) RequestPairingCode(phoneNumber string) (string, error) {
	if a.waManager == nil {
		return "", fmt.Errorf("WhatsApp manager not initialized")
	}

	return a.waManager.RequestPairingCode(phoneNumber)
}

// GetConnectionStatus returns current connection status
func (a *App) GetConnectionStatus() *whatsapp.ConnectionStatus {
	if a.waManager == nil {
		return &whatsapp.ConnectionStatus{IsConnected: false}
	}

	return a.waManager.GetConnectionStatus()
}

// DisconnectWhatsApp disconnects from WhatsApp
func (a *App) DisconnectWhatsApp() error {
	if a.waManager == nil {
		return fmt.Errorf("WhatsApp manager not initialized")
	}

	return a.waManager.Disconnect()
}

// GetConnectionEvents returns a channel for connection events
func (a *App) GetConnectionEvents() <-chan whatsapp.ConnectionEvent {
	if a.waManager == nil {
		return nil
	}

	return a.waManager.GetEventChannel()
}

// GetChats returns all WhatsApp chats
func (a *App) GetChats() ([]whatsapp.Chat, error) {
	if a.waManager == nil {
		return nil, fmt.Errorf("WhatsApp manager not initialized")
	}

	return a.waManager.GetChats()
}

// GetMessages retrieves messages for a specific chat
func (a *App) GetMessages(chatID string) ([]whatsapp.Message, error) {
	return a.waManager.GetMessages(chatID)
}

// Auto-reply methods

// GetAutoReplyConfig gets the current auto-reply configuration
func (a *App) GetAutoReplyConfig() (*whatsapp.AutoReplyConfig, error) {
	if a.waManager == nil {
		return nil, fmt.Errorf("WhatsApp manager not initialized")
	}
	return a.waManager.GetAutoReplyConfig(), nil
}

// UpdateAutoReplyConfig updates the auto-reply configuration
func (a *App) UpdateAutoReplyConfig(config *whatsapp.AutoReplyConfig) error {
	if a.waManager == nil {
		return fmt.Errorf("WhatsApp manager not initialized")
	}
	return a.waManager.UpdateAutoReplyConfig(config)
}

// TestAIConnection tests the AI connection with current config
func (a *App) TestAIConnection(provider string) (string, error) {
	return a.waManager.TestAIConnection(provider)
}

// GetContacts retrieves all WhatsApp contacts
func (a *App) GetContacts() ([]whatsapp.Contact, error) {
	return a.waManager.GetContacts()
}

// GetContactInfo retrieves information about a specific contact
func (a *App) GetContactInfo(jid string) (*whatsapp.ContactInfo, error) {
	return a.waManager.GetContactInfo(jid)
}

// Scheduler methods

// AddScheduledTask adds a new scheduled task
func (a *App) AddScheduledTask(task *whatsapp.ScheduledTask) error {
	return a.waManager.AddScheduledTask(task)
}

// RemoveScheduledTask removes a scheduled task
func (a *App) RemoveScheduledTask(taskID string) error {
	return a.waManager.RemoveScheduledTask(taskID)
}

// GetScheduledTask retrieves a scheduled task by ID
func (a *App) GetScheduledTask(taskID string) (*whatsapp.ScheduledTask, error) {
	return a.waManager.GetScheduledTask(taskID)
}

// GetAllScheduledTasks retrieves all scheduled tasks
func (a *App) GetAllScheduledTasks() []*whatsapp.ScheduledTask {
	return a.waManager.GetAllScheduledTasks()
}

// UpdateScheduledTask updates an existing scheduled task
func (a *App) UpdateScheduledTask(taskID string, task *whatsapp.ScheduledTask) error {
	return a.waManager.UpdateScheduledTask(taskID, task)
}

// GetSchedulerStats returns scheduler statistics
func (a *App) GetSchedulerStats() map[string]interface{} {
	return a.waManager.GetSchedulerStats()
}

func (a *App) SendMessage(chatID, text string) error {
	return a.waManager.SendMessage(chatID, text)
}
