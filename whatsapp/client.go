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
	waProto "go.mau.fi/whatsmeow/binary/proto"
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

type Contact struct {
	JID           string `json:"jid"`
	Name          string `json:"name"`
	PhoneNumber   string `json:"phoneNumber"`
	PushName      string `json:"pushName"`
	BusinessName  string `json:"businessName"`
	ProfilePicURL string `json:"profilePicUrl"`
	IsGroup       bool   `json:"isGroup"`
	IsBusiness    bool   `json:"isBusiness"`
	LastSeen      string `json:"lastSeen"`
}

type ContactInfo struct {
	JID           string     `json:"jid"`
	Name          string     `json:"name"`
	PhoneNumber   string     `json:"phoneNumber"`
	PushName      string     `json:"pushName"`
	BusinessName  string     `json:"businessName"`
	ProfilePicURL string     `json:"profilePicUrl"`
	Status        string     `json:"status"`
	IsGroup       bool       `json:"isGroup"`
	IsBusiness    bool       `json:"isBusiness"`
	IsBlocked     bool       `json:"isBlocked"`
	LastSeen      string     `json:"lastSeen"`
	GroupInfo     *GroupInfo `json:"groupInfo,omitempty"`
}

type GroupInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Owner       string    `json:"owner"`
	CreatedAt   time.Time `json:"createdAt"`
	MemberCount int       `json:"memberCount"`
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

// GetChats retrieves chats from SQLite database only (no server calls)
func (m *Manager) GetChats() ([]Chat, error) {
	if m.container == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Get chats from message database
	return m.getChatsFromMessageDB()
}

func (m *Manager) getChatsFromMessageDB() ([]Chat, error) {
	if m.messageDB == nil {
		// Fallback to demo data if message DB not available
		return m.getDemoChats()
	}

	// Get stored chats from message database
	storedChats, err := m.messageDB.GetAllChats()
	if err != nil {
		// Fallback to demo data on error
		return m.getDemoChats()
	}

	// Convert StoredChat to Chat format
	var chats []Chat
	for _, stored := range storedChats {
		// Get last message for display
		lastMsg, _ := m.messageDB.GetLastMessage(stored.JID)

		lastText := "No messages"
		timeStr := ""
		if lastMsg != nil {
			lastText = lastMsg.Content
			timeStr = m.formatMessageTime(lastMsg.Timestamp)
		}

		chat := Chat{
			ID:      stored.JID,
			Name:    stored.Name,
			Last:    lastText,
			Time:    timeStr,
			Unread:  stored.UnreadCount,
			Avatar:  "",
			IsGroup: stored.IsGroup,
		}
		chats = append(chats, chat)
	}

	// If no chats found, return demo data
	if len(chats) == 0 {
		return m.getDemoChats()
	}

	return chats, nil
}

func (m *Manager) getDemoChats() ([]Chat, error) {
	demoChats := []Chat{
		{
			ID:      "demo1@s.whatsapp.net",
			Name:    "John Doe",
			Last:    "Hello there!",
			Time:    "10:30",
			Unread:  2,
			Avatar:  "",
			IsGroup: false,
		},
		{
			ID:      "demo2@s.whatsapp.net",
			Name:    "Jane Smith",
			Last:    "How are you?",
			Time:    "09:15",
			Unread:  0,
			Avatar:  "",
			IsGroup: false,
		},
		{
			ID:      "demo3@g.us",
			Name:    "Family Group",
			Last:    "See you tomorrow",
			Time:    "Yesterday",
			Unread:  5,
			Avatar:  "",
			IsGroup: true,
		},
	}

	return demoChats, nil
}

func (m *Manager) formatMessageTime(timestamp int64) string {
	msgTime := time.Unix(timestamp, 0)
	now := time.Now()

	// If today, show time
	if msgTime.Format("2006-01-02") == now.Format("2006-01-02") {
		return msgTime.Format("15:04")
	}

	// If yesterday
	yesterday := now.AddDate(0, 0, -1)
	if msgTime.Format("2006-01-02") == yesterday.Format("2006-01-02") {
		return "Yesterday"
	}

	// If this week, show day name
	if msgTime.After(now.AddDate(0, 0, -7)) {
		return msgTime.Format("Monday")
	}

	// Otherwise show date
	return msgTime.Format("02/01")
}

// Fallback method to get chats from contacts when database query fails
func (m *Manager) getChatsFromContacts(contacts map[types.JID]types.ContactInfo) ([]Chat, error) {
	var chats []Chat
	count := 0

	for jid, contact := range contacts {
		if count >= 20 {
			break
		}

		if jid.IsEmpty() || jid.Server == "broadcast" {
			continue
		}

		// Determine chat name
		chatName := jid.User
		if contact.PushName != "" {
			chatName = contact.PushName
		} else if contact.BusinessName != "" {
			chatName = contact.BusinessName
		}

		// For group chats, try to get group info
		if jid.Server == "g.us" {
			groupInfo, err := m.client.GetGroupInfo(jid)
			if err == nil && groupInfo.Name != "" {
				chatName = groupInfo.Name
			}
		}

		chat := Chat{
			ID:      jid.String(),
			Name:    chatName,
			Last:    "Tap to load messages",
			Time:    "",
			Unread:  0,
			IsGroup: jid.Server == "g.us",
		}

		chats = append(chats, chat)
		count++
	}

	return chats, nil
}

func (m *Manager) GetMessages(chatID string) ([]Message, error) {
	if m.messageDB == nil {
		// Fallback to demo messages if message DB not available
		return m.getDemoMessages()
	}

	// Get messages from database
	storedMessages, err := m.messageDB.GetChatMessages(chatID, 50, 0)
	if err != nil {
		// Fallback to demo messages on error
		return m.getDemoMessages()
	}

	// Convert StoredMessage to Message format
	var messages []Message
	for _, stored := range storedMessages {
		senderName := "Unknown"
		if stored.IsFromMe {
			senderName = "Me"
		} else {
			// Try to get contact name from sender JID
			senderName = m.getContactName(stored.SenderJID)
		}

		message := Message{
			ID:     stored.ID,
			ChatID: chatID,
			Author: senderName,
			Text:   stored.Content,
			Time:   m.formatMessageTime(stored.Timestamp),
			IsMine: stored.IsFromMe,
			Type:   stored.MessageType,
		}
		messages = append(messages, message)
	}

	// Reverse to show newest at bottom
	for i := len(messages)/2 - 1; i >= 0; i-- {
		opp := len(messages) - 1 - i
		messages[i], messages[opp] = messages[opp], messages[i]
	}

	// If no messages found, return demo messages
	if len(messages) == 0 {
		return m.getDemoMessages()
	}

	return messages, nil
}

func (m *Manager) getDemoMessages() ([]Message, error) {
	demoMessages := []Message{
		{
			ID:     "msg1",
			ChatID: "demo",
			Author: "John Doe",
			Text:   "Hello! How are you?",
			Time:   "10:30",
			IsMine: false,
			Type:   "text",
		},
		{
			ID:     "msg2",
			ChatID: "demo",
			Author: "Me",
			Text:   "I'm doing great, thanks for asking!",
			Time:   "10:32",
			IsMine: true,
			Type:   "text",
		},
		{
			ID:     "msg3",
			ChatID: "demo",
			Author: "John Doe",
			Text:   "That's wonderful to hear!",
			Time:   "10:35",
			IsMine: false,
			Type:   "text",
		},
	}

	return demoMessages, nil
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

// SendMessage sends a text message to a specific chat
func (m *Manager) SendMessage(chatID, text string) error {
	if m.client == nil || !m.client.IsConnected() {
		return fmt.Errorf("WhatsApp client not connected")
	}

	// Parse JID from chatID
	jid, err := types.ParseJID(chatID)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %v", err)
	}

	// Create message
	msg := &waProto.Message{
		Conversation: &text,
	}

	// Send message
	response, err := m.client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	// Store the sent message in the database
	if m.messageDB != nil {
		now := time.Now()
		storedMsg := &StoredMessage{
			ID:          response.ID,
			ChatJID:     jid.String(),
			SenderJID:   "me",
			MessageType: "text",
			Content:     text,
			Timestamp:   now.Unix(),
			IsFromMe:    true,
			IsGroup:     jid.Server == "g.us",
			CreatedAt:   now,
		}

		err = m.messageDB.StoreDirectMessage(storedMsg)
		if err != nil {
			// Log the error but don't fail the send operation
			m.log.Errorf("Failed to store sent message in database: %v", err)
		}

		// Update chat in database
		chatName := jid.User
		if jid.Server == "g.us" {
			if groupInfo, err := m.client.GetGroupInfo(jid); err == nil && groupInfo.Name != "" {
				chatName = groupInfo.Name
			}
		} else {
			// For private chats, try to get contact name
			if contact, err := m.client.Store.Contacts.GetContact(context.Background(), jid); err == nil && contact.Found {
				if contact.PushName != "" {
					chatName = contact.PushName
				} else if contact.BusinessName != "" {
					chatName = contact.BusinessName
				}
			}
		}

		chat := &StoredChat{
			JID:             jid.String(),
			Name:            chatName,
			IsGroup:         jid.Server == "g.us",
			LastMessageID:   response.ID,
			LastMessageTime: now.Unix(),
			UpdatedAt:       now,
		}

		if err := m.messageDB.UpsertChat(chat); err != nil {
			m.log.Errorf("Failed to update chat in database: %v", err)
		}
	}

	return nil
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

// TestAIConnection tests the AI connection with current settings
func (m *Manager) TestAIConnection(provider string) (string, error) {
	if m.autoReply == nil {
		return "", fmt.Errorf("auto-reply not initialized")
	}

	testMessage := "Hello, this is a test message."

	// Temporarily override provider for testing
	originalProvider := m.autoReply.config.AIProvider
	m.autoReply.config.AIProvider = provider
	defer func() {
		m.autoReply.config.AIProvider = originalProvider
	}()

	response, err := m.autoReply.generateAIResponse(testMessage)
	if err != nil {
		return "", fmt.Errorf("AI connection test failed: %v", err)
	}

	return response, nil
}

// Contact management methods

// GetContacts retrieves all WhatsApp contacts
func (m *Manager) GetContacts() ([]Contact, error) {
	if m.client == nil || !m.client.IsConnected() {
		return nil, fmt.Errorf("WhatsApp client not connected")
	}

	ctx := context.Background()

	// Get all contacts from WhatsApp store
	contacts, err := m.client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		m.log.Errorf("Failed to get contacts: %v", err)
		return nil, fmt.Errorf("failed to get contacts: %v", err)
	}

	var contactList []Contact

	for jid, contact := range contacts {
		if jid.IsEmpty() || jid.Server == "broadcast" {
			continue
		}

		// Determine contact name
		contactName := jid.User
		if contact.PushName != "" {
			contactName = contact.PushName
		} else if contact.BusinessName != "" {
			contactName = contact.BusinessName
		}

		// Get profile picture URL if available
		profilePicURL := ""
		if profilePic, err := m.client.GetProfilePictureInfo(jid, &whatsmeow.GetProfilePictureParams{}); err == nil && profilePic != nil {
			profilePicURL = profilePic.URL
		}

		// Check if contact is business
		isBusiness := contact.BusinessName != ""

		// Format phone number
		phoneNumber := "+" + jid.User
		if jid.Server == "g.us" {
			phoneNumber = "" // Groups don't have phone numbers
		}

		contactItem := Contact{
			JID:           jid.String(),
			Name:          contactName,
			PhoneNumber:   phoneNumber,
			PushName:      contact.PushName,
			BusinessName:  contact.BusinessName,
			ProfilePicURL: profilePicURL,
			IsGroup:       jid.Server == "g.us",
			IsBusiness:    isBusiness,
			LastSeen:      "", // We'll implement this later if needed
		}

		contactList = append(contactList, contactItem)
	}

	return contactList, nil
}

// GetContactInfo retrieves detailed information for a specific contact
func (m *Manager) GetContactInfo(jidStr string) (*ContactInfo, error) {
	if m.client == nil || !m.client.IsConnected() {
		return nil, fmt.Errorf("WhatsApp client not connected")
	}

	// Parse JID
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JID: %v", err)
	}

	ctx := context.Background()

	// Get contact from store
	contact, err := m.client.Store.Contacts.GetContact(ctx, jid)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %v", err)
	}

	// Get profile picture
	profilePicURL := ""
	if profilePic, err := m.client.GetProfilePictureInfo(jid, &whatsmeow.GetProfilePictureParams{}); err == nil && profilePic != nil {
		profilePicURL = profilePic.URL
	}

	// Get status/about
	status := ""
	// Note: WhatsApp doesn't easily provide status info through whatsmeow

	contactInfo := &ContactInfo{
		JID:           jid.String(),
		Name:          contact.PushName,
		PhoneNumber:   "+" + jid.User,
		PushName:      contact.PushName,
		BusinessName:  contact.BusinessName,
		ProfilePicURL: profilePicURL,
		Status:        status,
		IsGroup:       jid.Server == "g.us",
		IsBusiness:    contact.BusinessName != "",
		IsBlocked:     false, // We'll implement this later if needed
		LastSeen:      "",    // We'll implement this later if needed
	}

	// For groups, get additional info
	if jid.Server == "g.us" {
		groupInfo, err := m.client.GetGroupInfo(jid)
		if err == nil {
			contactInfo.Name = groupInfo.Name
			contactInfo.PhoneNumber = ""

			// Find group owner
			ownerJID := ""
			for _, participant := range groupInfo.Participants {
				if participant.IsAdmin || participant.IsSuperAdmin {
					ownerJID = participant.JID.String()
					break
				}
			}

			contactInfo.GroupInfo = &GroupInfo{
				Name:        groupInfo.Name,
				Description: groupInfo.Topic,
				Owner:       ownerJID,
				CreatedAt:   time.Now(), // GroupInfo doesn't have CreationTime, using current time as placeholder
				MemberCount: len(groupInfo.Participants),
			}
		}
	}

	return contactInfo, nil
}

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

// Scheduler methods

// AddScheduledTask adds a new scheduled task
func (m *Manager) AddScheduledTask(task *ScheduledTask) error {
	if m.scheduler == nil {
		return fmt.Errorf("scheduler not initialized")
	}
	return m.scheduler.AddTask(task)
}

// RemoveScheduledTask removes a scheduled task
func (m *Manager) RemoveScheduledTask(taskID string) error {
	if m.scheduler == nil {
		return fmt.Errorf("scheduler not initialized")
	}
	return m.scheduler.RemoveTask(taskID)
}

// GetScheduledTask retrieves a scheduled task by ID
func (m *Manager) GetScheduledTask(taskID string) (*ScheduledTask, error) {
	if m.scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	return m.scheduler.GetTask(taskID)
}

// GetAllScheduledTasks retrieves all scheduled tasks
func (m *Manager) GetAllScheduledTasks() []*ScheduledTask {
	if m.scheduler == nil {
		return []*ScheduledTask{}
	}
	return m.scheduler.GetAllTasks()
}

// UpdateScheduledTask updates an existing scheduled task
func (m *Manager) UpdateScheduledTask(taskID string, task *ScheduledTask) error {
	if m.scheduler == nil {
		return fmt.Errorf("scheduler not initialized")
	}
	return m.scheduler.UpdateTask(taskID, task)
}

// GetSchedulerStats returns scheduler statistics
func (m *Manager) GetSchedulerStats() map[string]interface{} {
	if m.scheduler == nil {
		return map[string]interface{}{"error": "scheduler not initialized"}
	}
	return m.scheduler.GetTaskStats()
}
