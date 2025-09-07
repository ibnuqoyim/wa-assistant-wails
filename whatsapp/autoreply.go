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

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
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

// GetConfig returns the current auto-reply configuration
func (arm *AutoReplyManager) GetConfig() *AutoReplyConfig {
	if arm.config == nil {
		return &AutoReplyConfig{
			Enabled:           false,
			AIProvider:        "openai",
			OpenAIModel:       "gpt-3.5-turbo",
			OllamaURL:         "http://localhost:11434",
			OllamaModel:       "llama2",
			SystemPrompt:      "You are a helpful WhatsApp assistant. Keep responses concise and friendly.",
			ResponseDelay:     2,
			MaxResponseLength: 500,
		}
	}
	return arm.config
}

// GetDefaultAutoReplyConfig returns default auto-reply configuration
func GetDefaultAutoReplyConfig() *AutoReplyConfig {
	return &AutoReplyConfig{
		Enabled:           false,
		AIProvider:        "openai",
		OpenAIModel:       "gpt-3.5-turbo",
		OllamaURL:         "http://localhost:11434",
		OllamaModel:       "llama2",
		SystemPrompt:      "You are a helpful WhatsApp assistant. Keep responses concise and friendly.",
		ResponseDelay:     2,
		MaxResponseLength: 500,
	}
}

// ProcessIncomingMessage processes incoming messages and generates AI responses with retry logic
func (arm *AutoReplyManager) ProcessIncomingMessage(evt *events.Message, manager *Manager) error {
	if !arm.shouldReply(evt) {
		return nil
	}

	// Extract message text
	messageText := arm.extractMessageText(evt)
	if messageText == "" {
		return nil
	}

	// Add delay before responding (run in goroutine to not block)
	go func() {
		if arm.config.ResponseDelay > 0 {
			time.Sleep(time.Duration(arm.config.ResponseDelay) * time.Second)
		}

		// Retry logic for AI response generation
		maxRetries := 3
		var response string
		var err error

		for attempt := 1; attempt <= maxRetries; attempt++ {
			response, err = arm.generateAIResponse(messageText)
			if err == nil {
				break
			}

			// Check if it's a rate limit error and wait before retrying
			if strings.Contains(err.Error(), "rate limit") && attempt < maxRetries {
				waitTime := time.Duration(attempt*30) * time.Second // Exponential backoff
				fmt.Printf("Rate limit hit, waiting %v before retry %d/%d\n", waitTime, attempt+1, maxRetries)
				time.Sleep(waitTime)
				continue
			}

			// For other errors, don't retry immediately
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt*5) * time.Second)
			}
		}

		if err != nil {
			fmt.Printf("Failed to generate AI response after %d attempts: %v\n", maxRetries, err)
			return
		}

		// Send response via WhatsApp
		if err := manager.SendMessage(evt.Info.Chat.String(), response); err != nil {
			fmt.Printf("Failed to send AI response: %v\n", err)
		}
	}()

	return nil
}

// shouldReply determines if we should reply to this message
func (arm *AutoReplyManager) shouldReply(evt *events.Message) bool {
	if arm.config == nil || !arm.config.Enabled {
		return false
	}

	// Don't reply to our own messages
	if evt.Info.IsFromMe {
		return false
	}

	// Don't reply to group messages (for now)
	if evt.Info.IsGroup {
		return false
	}

	// Extract message text
	messageText := arm.extractMessageText(evt)
	if strings.TrimSpace(messageText) == "" {
		return false
	}

	// Check whitelist if configured
	if len(arm.config.WhitelistNumbers) > 0 {
		phoneNumber := evt.Info.Sender.User
		for _, whitelisted := range arm.config.WhitelistNumbers {
			if phoneNumber == whitelisted {
				return true
			}
		}
		return false
	}

	return true
}

// extractMessageText extracts text from various message types
func (arm *AutoReplyManager) extractMessageText(evt *events.Message) string {
	if evt == nil || evt.Message == nil {
		return ""
	}

	// Handle different message types
	if evt.Message.GetConversation() != "" {
		return evt.Message.GetConversation()
	}

	if extMsg := evt.Message.GetExtendedTextMessage(); extMsg != nil {
		return extMsg.GetText()
	}

	// For other message types, return a placeholder
	if evt.Message.GetImageMessage() != nil {
		return "[Image message]"
	}
	if evt.Message.GetVideoMessage() != nil {
		return "[Video message]"
	}
	if evt.Message.GetAudioMessage() != nil {
		return "[Audio message]"
	}
	if evt.Message.GetDocumentMessage() != nil {
		return "[Document message]"
	}

	return ""
}

// generateAIResponse generates a response using the configured AI provider
func (arm *AutoReplyManager) generateAIResponse(messageText string) (string, error) {
	switch arm.config.AIProvider {
	case "openai":
		return arm.generateOpenAIResponse(messageText)
	case "ollama":
		return arm.generateOllamaResponse(messageText)
	default:
		return "", fmt.Errorf("unsupported AI provider: %s", arm.config.AIProvider)
	}
}

// OpenAI API structures
type OpenAIRequest struct {
	Model     string          `json:"model"`
	Messages  []OpenAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
}

type OpenAIChoice struct {
	Message OpenAIMessage `json:"message"`
}

// generateOpenAIResponse generates response using OpenAI API with enhanced error handling
func (arm *AutoReplyManager) generateOpenAIResponse(messageText string) (string, error) {
	if arm.config.OpenAIAPIKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	messages := []OpenAIMessage{
		{Role: "system", Content: arm.config.SystemPrompt},
		{Role: "user", Content: messageText},
	}

	request := OpenAIRequest{
		Model:     arm.config.OpenAIModel,
		Messages:  messages,
		MaxTokens: arm.config.MaxResponseLength / 4, // Rough token estimation
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create request with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+arm.config.OpenAIAPIKey)

	resp, err := arm.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	// Enhanced error handling for different HTTP status codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Success, continue processing
	case http.StatusUnauthorized:
		return "", fmt.Errorf("invalid API key")
	case http.StatusTooManyRequests:
		return "", fmt.Errorf("rate limit exceeded, please try again later")
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return "", fmt.Errorf("OpenAI service temporarily unavailable")
	default:
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from OpenAI")
	}

	response := openAIResp.Choices[0].Message.Content
	return arm.truncateResponse(response), nil
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

// generateOllamaResponse generates response using Ollama API with enhanced error handling
func (arm *AutoReplyManager) generateOllamaResponse(messageText string) (string, error) {
	if arm.config.OllamaURL == "" {
		return "", fmt.Errorf("ollama URL not configured")
	}

	prompt := fmt.Sprintf("%s\n\nUser: %s\nAssistant:", arm.config.SystemPrompt, messageText)

	request := OllamaRequest{
		Model:  arm.config.OllamaModel,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create request with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Longer timeout for local models
	defer cancel()

	url := strings.TrimSuffix(arm.config.OllamaURL, "/") + "/api/generate"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := arm.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error (check if Ollama is running): %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	// Enhanced error handling for different HTTP status codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Success, continue processing
	case http.StatusNotFound:
		return "", fmt.Errorf("model '%s' not found in Ollama", arm.config.OllamaModel)
	case http.StatusInternalServerError:
		return "", fmt.Errorf("ollama internal error: %s", string(body))
	case http.StatusServiceUnavailable:
		return "", fmt.Errorf("ollama service unavailable")
	default:
		return "", fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if ollamaResp.Response == "" {
		return "", fmt.Errorf("empty response from Ollama")
	}

	return arm.truncateResponse(ollamaResp.Response), nil
}

// truncateResponse truncates response to max length
func (arm *AutoReplyManager) truncateResponse(response string) string {
	response = strings.TrimSpace(response)
	if len(response) > arm.config.MaxResponseLength {
		return response[:arm.config.MaxResponseLength] + "..."
	}
	return response
}

// TestAIConnection tests the AI service connection
func (arm *AutoReplyManager) TestAIConnection() error {
	if arm.config == nil {
		return fmt.Errorf("auto-reply not configured")
	}

	testMessage := "Hello, this is a test message."
	_, err := arm.generateAIResponse(testMessage)
	return err
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

// SaveConfig saves the auto-reply configuration to the database
func (arm *AutoReplyManager) SaveConfig(db *MessageDB) error {
	if arm.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	return db.SaveConfig(arm.config)
}

// LoadConfig loads the auto-reply configuration from the database
func (arm *AutoReplyManager) LoadConfig(db *MessageDB) error {
	config, err := db.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config from database: %v", err)
	}

	arm.config = config
	return nil
}
