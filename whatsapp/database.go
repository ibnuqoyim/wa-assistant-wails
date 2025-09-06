package whatsapp

import (
	"database/sql"
	"fmt"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	_ "github.com/mattn/go-sqlite3"
)

// MessageDB handles message database operations
type MessageDB struct {
	db *sql.DB
}

// NewMessageDB creates a new message database handler
func NewMessageDB(dbPath string) (*MessageDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	msgDB := &MessageDB{db: db}
	if err := msgDB.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return msgDB, nil
}

// createTables creates the necessary database tables
func (m *MessageDB) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			chat_jid TEXT NOT NULL,
			sender_jid TEXT NOT NULL,
			message_type TEXT NOT NULL,
			content TEXT,
			media_path TEXT,
			media_type TEXT,
			caption TEXT,
			timestamp INTEGER NOT NULL,
			is_from_me BOOLEAN NOT NULL,
			is_group BOOLEAN NOT NULL,
			quoted_message_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_chat_jid ON messages(chat_jid)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp)`,
		`CREATE TABLE IF NOT EXISTS chats (
			jid TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			is_group BOOLEAN NOT NULL,
			last_message_id TEXT,
			last_message_time INTEGER,
			unread_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS contacts (
			jid TEXT PRIMARY KEY,
			name TEXT,
			push_name TEXT,
			business_name TEXT,
			profile_pic_url TEXT,
			is_business BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %v", err)
		}
	}

	return nil
}

// StoredMessage represents a message stored in database
type StoredMessage struct {
	ID              string    `json:"id"`
	ChatJID         string    `json:"chatJid"`
	SenderJID       string    `json:"senderJid"`
	MessageType     string    `json:"messageType"`
	Content         string    `json:"content"`
	MediaPath       string    `json:"mediaPath,omitempty"`
	MediaType       string    `json:"mediaType,omitempty"`
	Caption         string    `json:"caption,omitempty"`
	Timestamp       int64     `json:"timestamp"`
	IsFromMe        bool      `json:"isFromMe"`
	IsGroup         bool      `json:"isGroup"`
	QuotedMessageID string    `json:"quotedMessageId,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

// StoredChat represents a chat stored in database
type StoredChat struct {
	JID             string    `json:"jid"`
	Name            string    `json:"name"`
	IsGroup         bool      `json:"isGroup"`
	LastMessageID   string    `json:"lastMessageId,omitempty"`
	LastMessageTime int64     `json:"lastMessageTime"`
	UnreadCount     int       `json:"unreadCount"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// StoreMessage stores a message in the database from events.Message
func (m *MessageDB) StoreMessageFromEvent(evt *events.Message) error {
	if evt == nil {
		return fmt.Errorf("invalid message event")
	}

	// Extract message details from event
	messageID := evt.Info.ID
	chatJID := evt.Info.Chat.String()
	timestamp := int64(evt.Info.Timestamp.Unix())
	isFromMe := evt.Info.IsFromMe
	isGroup := evt.Info.IsGroup

	// Determine sender JID
	var actualSenderJID string
	if isFromMe {
		actualSenderJID = "me"
	} else if isGroup && !evt.Info.Sender.IsEmpty() {
		actualSenderJID = evt.Info.Sender.String()
	} else {
		actualSenderJID = chatJID
	}

	// Extract message content
	content, messageType, mediaPath, mediaType, caption := m.extractMessageContent(evt.Message)

	// Store message
	query := `INSERT OR REPLACE INTO messages 
		(id, chat_jid, sender_jid, message_type, content, media_path, media_type, caption, 
		 timestamp, is_from_me, is_group, quoted_message_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	quotedID := ""
	// Extract quoted message ID if available
	if evt.Message.GetExtendedTextMessage() != nil && 
		evt.Message.GetExtendedTextMessage().GetContextInfo() != nil &&
		evt.Message.GetExtendedTextMessage().GetContextInfo().GetStanzaID() != "" {
		quotedID = evt.Message.GetExtendedTextMessage().GetContextInfo().GetStanzaID()
	}

	_, err := m.db.Exec(query, messageID, chatJID, actualSenderJID, messageType, content,
		mediaPath, mediaType, caption, timestamp, isFromMe, isGroup, quotedID)
	if err != nil {
		return fmt.Errorf("failed to store message: %v", err)
	}

	// Update chat last message
	if err := m.updateChatLastMessage(chatJID, messageID, timestamp, !isFromMe); err != nil {
		return fmt.Errorf("failed to update chat: %v", err)
	}

	return nil
}

// StoreMessage stores a message in the database (legacy method)
func (m *MessageDB) StoreMessage(msg *waProto.WebMessageInfo) error {
	if msg == nil || msg.Key == nil {
		return fmt.Errorf("invalid message")
	}

	// Extract message details
	messageID := msg.Key.GetId()
	chatJID := msg.Key.GetRemoteJid()
	timestamp := int64(msg.GetMessageTimestamp())
	isFromMe := msg.Key.GetFromMe()
	isGroup := msg.Key.GetRemoteJid() != "" && msg.Key.GetRemoteJid()[len(msg.Key.GetRemoteJid())-5:] == "@g.us"

	// Determine sender JID
	var actualSenderJID string
	if isFromMe {
		actualSenderJID = "me"
	} else if isGroup && msg.Key.GetParticipant() != "" {
		actualSenderJID = msg.Key.GetParticipant()
	} else {
		actualSenderJID = chatJID
	}

	// Extract message content
	content, messageType, mediaPath, mediaType, caption := m.extractMessageContent(msg.Message)

	// Store message
	query := `INSERT OR REPLACE INTO messages 
		(id, chat_jid, sender_jid, message_type, content, media_path, media_type, caption, 
		 timestamp, is_from_me, is_group, quoted_message_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	quotedID := ""
	if msg.Message != nil && msg.Message.ExtendedTextMessage != nil && 
		msg.Message.ExtendedTextMessage.ContextInfo != nil &&
		msg.Message.ExtendedTextMessage.ContextInfo.StanzaID != nil {
		quotedID = *msg.Message.ExtendedTextMessage.ContextInfo.StanzaID
	}

	_, err := m.db.Exec(query, messageID, chatJID, actualSenderJID, messageType, content,
		mediaPath, mediaType, caption, timestamp, isFromMe, isGroup, quotedID)
	if err != nil {
		return fmt.Errorf("failed to store message: %v", err)
	}

	// Update chat last message
	if err := m.updateChatLastMessage(chatJID, messageID, timestamp, !isFromMe); err != nil {
		return fmt.Errorf("failed to update chat: %v", err)
	}

	return nil
}

// extractMessageContent extracts content from WhatsApp message
func (m *MessageDB) extractMessageContent(msg *waProto.Message) (content, messageType, mediaPath, mediaType, caption string) {
	if msg == nil {
		return "", "unknown", "", "", ""
	}

	switch {
	case msg.Conversation != nil:
		return *msg.Conversation, "text", "", "", ""
	case msg.ExtendedTextMessage != nil:
		content = ""
		if msg.ExtendedTextMessage.Text != nil {
			content = *msg.ExtendedTextMessage.Text
		}
		return content, "text", "", "", ""
	case msg.ImageMessage != nil:
		caption = ""
		if msg.ImageMessage.Caption != nil {
			caption = *msg.ImageMessage.Caption
		}
		return "[Image]", "image", "", "image", caption
	case msg.VideoMessage != nil:
		caption = ""
		if msg.VideoMessage.Caption != nil {
			caption = *msg.VideoMessage.Caption
		}
		return "[Video]", "video", "", "video", caption
	case msg.AudioMessage != nil:
		return "[Audio]", "audio", "", "audio", ""
	case msg.DocumentMessage != nil:
		filename := "Document"
		if msg.DocumentMessage.FileName != nil {
			filename = *msg.DocumentMessage.FileName
		}
		return fmt.Sprintf("[Document: %s]", filename), "document", "", "document", ""
	case msg.StickerMessage != nil:
		return "[Sticker]", "sticker", "", "sticker", ""
	case msg.LocationMessage != nil:
		return "[Location]", "location", "", "", ""
	case msg.ContactMessage != nil:
		return "[Contact]", "contact", "", "", ""
	default:
		return "[Unknown Message]", "unknown", "", "", ""
	}
}

// updateChatLastMessage updates the last message info for a chat
func (m *MessageDB) updateChatLastMessage(chatJID, messageID string, timestamp int64, incrementUnread bool) error {
	// First, ensure chat exists
	_, err := m.db.Exec(`INSERT OR IGNORE INTO chats (jid, name, is_group, last_message_time, unread_count) 
		VALUES (?, ?, ?, 0, 0)`, chatJID, chatJID, chatJID[len(chatJID)-5:] == "@g.us")
	if err != nil {
		return err
	}

	// Update last message and optionally increment unread count
	if incrementUnread {
		_, err = m.db.Exec(`UPDATE chats 
			SET last_message_id = ?, last_message_time = ?, unread_count = unread_count + 1, updated_at = CURRENT_TIMESTAMP 
			WHERE jid = ?`, messageID, timestamp, chatJID)
	} else {
		_, err = m.db.Exec(`UPDATE chats 
			SET last_message_id = ?, last_message_time = ?, updated_at = CURRENT_TIMESTAMP 
			WHERE jid = ?`, messageID, timestamp, chatJID)
	}

	return err
}

// GetChatMessages retrieves messages for a specific chat
func (m *MessageDB) GetChatMessages(chatJID string, limit int, offset int) ([]StoredMessage, error) {
	query := `SELECT id, chat_jid, sender_jid, message_type, content, media_path, media_type, 
		caption, timestamp, is_from_me, is_group, quoted_message_id, created_at 
		FROM messages WHERE chat_jid = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?`

	rows, err := m.db.Query(query, chatJID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []StoredMessage
	for rows.Next() {
		var msg StoredMessage
		var mediaPath, mediaType, caption, quotedID sql.NullString

		err := rows.Scan(&msg.ID, &msg.ChatJID, &msg.SenderJID, &msg.MessageType, &msg.Content,
			&mediaPath, &mediaType, &caption, &msg.Timestamp, &msg.IsFromMe, &msg.IsGroup,
			&quotedID, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}

		msg.MediaPath = mediaPath.String
		msg.MediaType = mediaType.String
		msg.Caption = caption.String
		msg.QuotedMessageID = quotedID.String

		messages = append(messages, msg)
	}

	return messages, nil
}

// GetAllChats retrieves all chats from database
func (m *MessageDB) GetAllChats() ([]StoredChat, error) {
	query := `SELECT jid, name, is_group, last_message_id, last_message_time, unread_count, created_at, updated_at 
		FROM chats ORDER BY last_message_time DESC`

	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []StoredChat
	for rows.Next() {
		var chat StoredChat
		var lastMessageID sql.NullString

		err := rows.Scan(&chat.JID, &chat.Name, &chat.IsGroup, &lastMessageID,
			&chat.LastMessageTime, &chat.UnreadCount, &chat.CreatedAt, &chat.UpdatedAt)
		if err != nil {
			return nil, err
		}

		chat.LastMessageID = lastMessageID.String
		chats = append(chats, chat)
	}

	return chats, nil
}

// GetLastMessage retrieves the last message for a chat
func (m *MessageDB) GetLastMessage(chatJID string) (*StoredMessage, error) {
	query := `SELECT id, chat_jid, sender_jid, message_type, content, media_path, media_type, 
		caption, timestamp, is_from_me, is_group, quoted_message_id, created_at 
		FROM messages WHERE chat_jid = ? ORDER BY timestamp DESC LIMIT 1`

	row := m.db.QueryRow(query, chatJID)

	var msg StoredMessage
	var mediaPath, mediaType, caption, quotedID sql.NullString

	err := row.Scan(&msg.ID, &msg.ChatJID, &msg.SenderJID, &msg.MessageType, &msg.Content,
		&mediaPath, &mediaType, &caption, &msg.Timestamp, &msg.IsFromMe, &msg.IsGroup,
		&quotedID, &msg.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	msg.MediaPath = mediaPath.String
	msg.MediaType = mediaType.String
	msg.Caption = caption.String
	msg.QuotedMessageID = quotedID.String

	return &msg, nil
}

// MarkChatAsRead marks all messages in a chat as read
func (m *MessageDB) MarkChatAsRead(chatJID string) error {
	_, err := m.db.Exec(`UPDATE chats SET unread_count = 0, updated_at = CURRENT_TIMESTAMP WHERE jid = ?`, chatJID)
	return err
}

// Close closes the database connection
func (m *MessageDB) Close() error {
	return m.db.Close()
}
