package whatsapp

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
)

type Contact struct {
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
