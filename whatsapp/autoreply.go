package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
)

// AutoReplyConfig holds configuration for auto-reply feature
type AutoReplyConfig struct {
	Enabled           bool     `json:"enabled"`
	AIProvider        string   `json:"ai_provider"` // "openai" or "ollama"
	OpenAIAPIKey      string   `json:"openai_api_key"`
	OpenAIModel       string   `json:"openai_model"`
	OllamaURL         string   `json:"ollama_url"`
	OllamaModel       string   `json:"ollama_model"`
	WhitelistNumbers  []string `json:"whitelist_numbers"`
	SystemPrompt      string   `json:"system_prompt"`
	ResponseDelay     int      `json:"response_delay"` // seconds
	MaxResponseLength int      `json:"max_response_length"`
}

// AutoReplyManager handles automatic replies using AI
type AutoReplyManager struct {
	config *AutoReplyConfig
	client *http.Client
}

// NewAutoReplyManager creates a new auto-reply manager
func NewAutoReplyManager(config *AutoReplyConfig) *AutoReplyManager {
	return &AutoReplyManager{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UpdateConfig updates the auto-reply configuration
func (arm *AutoReplyManager) UpdateConfig(config *AutoReplyConfig) {
	arm.config = config
}

// IsWhitelisted checks if a phone number is in the whitelist
func (arm *AutoReplyManager) IsWhitelisted(jid types.JID) bool {
	if !arm.config.Enabled {
		return false
	}

	phoneNumber := jid.User
	for _, whitelistedNumber := range arm.config.WhitelistNumbers {
		if phoneNumber == whitelistedNumber {
			return true
		}
	}
	return false
}

// GenerateAIResponse generates an AI response based on the incoming message
func (arm *AutoReplyManager) GenerateAIResponse(ctx context.Context, message string) (string, error) {
	if !arm.config.Enabled {
		return "", fmt.Errorf("auto-reply is disabled")
	}

	switch arm.config.AIProvider {
	case "openai":
		return arm.generateOpenAIResponse(ctx, message)
	case "ollama":
		return arm.generateOllamaResponse(ctx, message)
	default:
		return "", fmt.Errorf("unsupported AI provider: %s", arm.config.AIProvider)
	}
}

// OpenAI API structures
type OpenAIRequest struct {
	Model     string      `json:"model"`
	Messages  []AIMessage `json:"messages"`
	MaxTokens int         `json:"max_tokens"`
}

type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// generateOpenAIResponse generates response using OpenAI API
func (arm *AutoReplyManager) generateOpenAIResponse(ctx context.Context, message string) (string, error) {
	if arm.config.OpenAIAPIKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	messages := []AIMessage{
		{
			Role:    "system",
			Content: arm.config.SystemPrompt,
		},
		{
			Role:    "user",
			Content: message,
		},
	}

	requestBody := OpenAIRequest{
		Model:     arm.config.OpenAIModel,
		Messages:  messages,
		MaxTokens: arm.config.MaxResponseLength,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+arm.config.OpenAIAPIKey)

	resp, err := arm.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return strings.TrimSpace(openAIResp.Choices[0].Message.Content), nil
}

// Ollama API structures
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// generateOllamaResponse generates response using Ollama local AI
func (arm *AutoReplyManager) generateOllamaResponse(ctx context.Context, message string) (string, error) {
	if arm.config.OllamaURL == "" {
		return "", fmt.Errorf("ollama URL not configured")
	}

	prompt := fmt.Sprintf("%s\n\nUser: %s\nAssistant:", arm.config.SystemPrompt, message)

	requestBody := OllamaRequest{
		Model:  arm.config.OllamaModel,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	url := strings.TrimSuffix(arm.config.OllamaURL, "/") + "/api/generate"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := arm.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama API error: %s", string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	response := strings.TrimSpace(ollamaResp.Response)

	// Limit response length
	if len(response) > arm.config.MaxResponseLength {
		response = response[:arm.config.MaxResponseLength-3] + "..."
	}

	return response, nil
}

// ProcessIncomingMessage processes incoming messages for auto-reply
func (m *Manager) ProcessIncomingMessage(jid types.JID, message *waProto.Message) {
	if m.autoReply == nil {
		return
	}

	// Check if sender is whitelisted
	if !m.autoReply.IsWhitelisted(jid) {
		return
	}

	// Extract text from message
	var messageText string
	if message.GetConversation() != "" {
		messageText = message.GetConversation()
	} else if extText := message.GetExtendedTextMessage(); extText != nil {
		messageText = extText.GetText()
	} else {
		// Skip non-text messages
		return
	}

	// Skip empty messages
	if strings.TrimSpace(messageText) == "" {
		return
	}

	// Generate AI response in a goroutine to avoid blocking
	go func() {
		// Add delay before responding
		if m.autoReply.config.ResponseDelay > 0 {
			time.Sleep(time.Duration(m.autoReply.config.ResponseDelay) * time.Second)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		aiResponse, err := m.autoReply.GenerateAIResponse(ctx, messageText)
		if err != nil {
			m.log.Errorf("Failed to generate AI response: %v", err)
			return
		}

		// Send the AI response
		err = m.SendMessage(jid.String(), aiResponse)
		if err != nil {
			m.log.Errorf("Failed to send auto-reply: %v", err)
		} else {
			m.log.Infof("Sent auto-reply to %s: %s", jid.String(), aiResponse)
		}
	}()
}

// GetDefaultAutoReplyConfig returns default configuration for auto-reply
func GetDefaultAutoReplyConfig() *AutoReplyConfig {
	return &AutoReplyConfig{
		Enabled:           false,
		AIProvider:        "openai",
		OpenAIAPIKey:      "",
		OpenAIModel:       "gpt-3.5-turbo",
		OllamaURL:         "http://localhost:11434",
		OllamaModel:       "llama2",
		WhitelistNumbers:  []string{},
		SystemPrompt:      "You are a helpful WhatsApp assistant. Respond briefly and helpfully to messages.",
		ResponseDelay:     2,
		MaxResponseLength: 500,
	}
}
