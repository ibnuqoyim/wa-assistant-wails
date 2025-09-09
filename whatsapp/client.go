package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Manager struct {
	client    *whatsmeow.Client
	container *sqlstore.Container
	log       waLog.Logger
	qrChan    chan string
	eventChan chan ConnectionEvent
	autoReply *AutoReplyManager
	scheduler *Scheduler
	messageDB *MessageDB
}

type ConnectionEvent struct {
	Type    string `json:"type"` // "connected", "disconnected", "qr", "code", "error"
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

type ConnectionStatus struct {
	IsConnected bool   `json:"isConnected"`
	DeviceID    string `json:"deviceId,omitempty"`
	PushName    string `json:"pushName,omitempty"`
}

type Chat struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Last    string `json:"last"`
	Time    string `json:"time"`
	Unread  int    `json:"unread"`
	IsGroup bool   `json:"isGroup"`
	Avatar  string `json:"avatar,omitempty"`
}

type Message struct {
	ID     string `json:"id"`
	ChatID string `json:"chatId"`
	Author string `json:"author"`
	Text   string `json:"text"`
	Time   string `json:"time"`
	IsMine bool   `json:"mine"`
	Type   string `json:"type"` // text, image, audio, etc
}

func NewManager(dbPath string) (*Manager, error) {
	// Initialize database
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:"+dbPath+"?_foreign_keys=on", waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	// Get the first device
	device, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %v", err)
	}

	// Create client
	client := whatsmeow.NewClient(device, waLog.Noop)

	// Initialize message database
	messageDB, err := NewMessageDB(dbPath + "_messages.db")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize message database: %v", err)
	}

	manager := &Manager{
		client:    client,
		container: container,
		log:       waLog.Noop,
		qrChan:    make(chan string, 1),
		eventChan: make(chan ConnectionEvent, 10),
		messageDB: messageDB,
	}

	// Load configuration from database
	config, err := messageDB.LoadConfig()
	if err != nil {
		// Log error but don't fail initialization
		fmt.Printf("Failed to load config from database: %v, using defaults\n", err)
		config = GetDefaultAutoReplyConfig()
	}

	// Initialize auto-reply manager with loaded config
	manager.autoReply = NewAutoReplyManager(config)

	// Initialize scheduler
	manager.scheduler = NewScheduler(manager, log.New(os.Stdout, "[Scheduler] ", log.LstdFlags))

	// Start scheduler
	if err := manager.scheduler.Start(); err != nil {
		return nil, fmt.Errorf("failed to start scheduler: %v", err)
	}

	// Setup event handlers
	client.AddEventHandler(manager.handleEvent)

	return manager, nil
}

// handleEvent handles WhatsApp events including incoming messages
func (m *Manager) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		// Store incoming message in database
		if err := m.messageDB.StoreMessageFromEvent(v); err != nil {
			m.log.Errorf("Failed to store message: %v", err)
		}

		// Handle auto-reply if enabled
		if m.autoReply != nil {
			if err := m.autoReply.ProcessIncomingMessage(v, m); err != nil {
				m.log.Errorf("Failed to process auto-reply: %v", err)
			}
		}

	case *events.Connected:
		m.eventChan <- ConnectionEvent{
			Type:    "connected",
			Message: "WhatsApp connected successfully",
		}

	case *events.Disconnected:
		m.eventChan <- ConnectionEvent{
			Type:    "disconnected",
			Message: "WhatsApp disconnected",
		}

	case *events.QR:
		// Generate QR code and send to channel
		qrString := v.Codes[0]
		png, err := qrcode.Encode(qrString, qrcode.Medium, 256)
		if err != nil {
			m.log.Errorf("Failed to generate QR code: %v", err)
			return
		}

		qrBase64 := base64.StdEncoding.EncodeToString(png)
		m.qrChan <- qrBase64

		m.eventChan <- ConnectionEvent{
			Type:    "qr",
			Message: "QR code generated",
			Data:    qrBase64,
		}
	}
}

func (m *Manager) CheckExistingDevice() (*ConnectionStatus, error) {
	// Get all devices from store
	devices, err := m.container.GetAllDevices(context.Background())
	if err != nil {
		return &ConnectionStatus{IsConnected: false}, err
	}

	// Check if we have any existing device
	if len(devices) == 0 {
		return &ConnectionStatus{IsConnected: false}, nil
	}

	// Try to use the first device (most recent)
	device := devices[0]

	// Create client with existing device
	m.client = whatsmeow.NewClient(device, m.log)

	// Check if device is already logged in
	if device.ID == nil {
		return &ConnectionStatus{IsConnected: false}, nil
	}

	// Try to connect
	if m.client.Store.ID == nil {
		return &ConnectionStatus{IsConnected: false}, nil
	}

	return &ConnectionStatus{
		IsConnected: true,
		DeviceID:    device.ID.String(),
		PushName:    m.client.Store.PushName,
	}, nil
}

func (m *Manager) ConnectExistingDevice() error {
	if m.client == nil {
		return fmt.Errorf("no client available")
	}

	// Add event handlers
	m.addEventHandlers()

	// Connect to WhatsApp
	err := m.client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	// Wait for connection to be established
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return fmt.Errorf("connection timeout")
	case event := <-m.eventChan:
		if event.Type == "connected" {
			return nil
		} else if event.Type == "error" {
			return fmt.Errorf("connection error: %s", event.Message)
		}
	}

	return nil
}

func (m *Manager) StartNewConnection() error {
	// Get device store
	deviceStore := m.container.NewDevice()
	m.client = whatsmeow.NewClient(deviceStore, m.log)

	// Add event handlers
	m.addEventHandlers()

	// Connect to WhatsApp
	err := m.client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	return nil
}

func (m *Manager) addEventHandlers() {
	m.client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			// Store incoming message in database
			if err := m.messageDB.StoreMessageFromEvent(v); err != nil {
				m.log.Errorf("Failed to store message: %v", err)
			}
			// Handle auto-reply if enabled
			if m.autoReply != nil {
				if err := m.autoReply.ProcessIncomingMessage(v, m); err != nil {
					m.log.Errorf("Failed to process auto-reply: %v", err)
				}
			}
		case *events.Connected:
			m.eventChan <- ConnectionEvent{
				Type:    "connected",
				Message: "Successfully connected to WhatsApp",
			}
		case *events.Disconnected:
			m.eventChan <- ConnectionEvent{
				Type:    "disconnected",
				Message: "Disconnected from WhatsApp",
			}
		case *events.QR:
			qrString := v.Codes[0]
			m.log.Infof("QR code: %s", qrString)

			// Generate QR code image
			qrCode, err := qrcode.New(qrString, qrcode.Medium)
			if err != nil {
				m.log.Errorf("Failed to generate QR code: %v", err)
				return
			}

			// Convert to base64
			qrPNG, err := qrCode.PNG(256)
			if err != nil {
				m.log.Errorf("Failed to convert QR code to PNG: %v", err)
				return
			}

			qrBase64 := base64.StdEncoding.EncodeToString(qrPNG)
			qrDataURL := "data:image/png;base64," + qrBase64

			m.eventChan <- ConnectionEvent{
				Type:    "qr",
				Message: "QR Code generated",
				Data:    qrDataURL,
			}
		case *events.PairSuccess:
			m.eventChan <- ConnectionEvent{
				Type:    "connected",
				Message: "Device paired successfully",
			}
		}
	})
}

func (m *Manager) generateQRCode(code string) (string, error) {
	// Generate QR code
	qr, err := qrcode.New(code, qrcode.Medium)
	if err != nil {
		return "", err
	}

	// Convert to PNG bytes
	pngBytes, err := qr.PNG(256)
	if err != nil {
		return "", err
	}

	// Convert to base64 data URL
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	return dataURL, nil
}

func (m *Manager) RequestPairingCode(phoneNumber string) (string, error) {
	if m.client == nil {
		return "", fmt.Errorf("client not initialized")
	}

	// Request pairing code
	code, err := m.client.PairPhone(context.Background(), phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return "", fmt.Errorf("failed to request pairing code: %v", err)
	}

	return code, nil
}

func (m *Manager) GetConnectionStatus() *ConnectionStatus {
	if m.client == nil || !m.client.IsConnected() {
		return &ConnectionStatus{IsConnected: false}
	}

	return &ConnectionStatus{
		IsConnected: true,
		DeviceID:    m.client.Store.ID.String(),
		PushName:    m.client.Store.PushName,
	}
}

func (m *Manager) Disconnect() error {
	if m.client != nil {
		m.client.Disconnect()
	}
	return nil
}

func (m *Manager) GetEventChannel() <-chan ConnectionEvent {
	return m.eventChan
}

func (m *Manager) getContactName(jid string) string {
	// Try to get contact info
	if m.client != nil && m.client.IsConnected() {
		contactJID, err := types.ParseJID(jid)
		if err == nil {
			contactInfo, err := m.client.Store.Contacts.GetContact(context.Background(), contactJID)
			if err == nil && contactInfo.Found {
				if contactInfo.PushName != "" {
					return contactInfo.PushName
				}
				if contactInfo.BusinessName != "" {
					return contactInfo.BusinessName
				}
			}
		}
	}

	// Fallback to JID or phone number
	if len(jid) > 0 && jid != "me" {
		// Extract phone number from JID
		if idx := strings.Index(jid, "@"); idx > 0 {
			return jid[:idx]
		}
		return jid
	}

	return "Unknown"
}

// Auto-reply methods

// GetAutoReplyConfig returns the current auto-reply configuration from database
func (m *Manager) GetAutoReplyConfig() *AutoReplyConfig {
	if m.messageDB == nil {
		return GetDefaultAutoReplyConfig()
	}

	config, err := m.messageDB.LoadConfig()
	if err != nil {
		m.log.Errorf("Failed to load config from database: %v", err)
		return GetDefaultAutoReplyConfig()
	}

	// Update the autoReply manager with the loaded config
	if m.autoReply == nil {
		m.autoReply = NewAutoReplyManager(config)
	} else {
		m.autoReply.UpdateConfig(config)
	}

	return config
}

// UpdateAutoReplyConfig updates the auto-reply configuration and saves to database
func (m *Manager) UpdateAutoReplyConfig(config *AutoReplyConfig) error {
	if m.messageDB == nil {
		return fmt.Errorf("database not initialized")
	}

	// Update the autoReply manager
	if m.autoReply == nil {
		m.autoReply = NewAutoReplyManager(config)
	} else {
		m.autoReply.UpdateConfig(config)
	}

	// Save to database
	return m.messageDB.SaveConfig(config)
}

// Contact management methods

func (m *Manager) Close() error {
	if m.client != nil {
		m.client.Disconnect()
	}

	// Stop scheduler
	if m.scheduler != nil {
		m.scheduler.Stop()
	}

	// Close message database
	if m.messageDB != nil {
		m.messageDB.Close()
	}

	return nil
}
