package whatsapp

import (
	"context"
	"fmt"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
)

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
		return nil, fmt.Errorf("message database not initialized")
	}

	// Get stored chats from message database
	storedChats, err := m.messageDB.GetAllChats()
	if err != nil {
		// Fallback to demo data on error
		return nil, err
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

	return chats, nil
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

// SendMessage sends a text message to a specific chat
// SendChatPresence sends chat presence (typing, etc) to a chat
func (m *Manager) SendChatPresence(chatJID string, presence types.ChatPresence) error {
	if m.client == nil || !m.client.IsConnected() {
		return fmt.Errorf("WhatsApp client not connected")
	}

	// Parse JID
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %v", err)
	}

	err = m.client.SendChatPresence(jid, presence, types.ChatPresenceMediaText)
	if err != nil {
		return fmt.Errorf("failed to send presence: %v", err)
	}

	return nil
}

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
