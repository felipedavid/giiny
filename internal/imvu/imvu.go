package imvu

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// IMVU represents the IMVU API client
type IMVU struct {
	client *Client
	ws     *WebSocketClient
}

// New creates a new IMVU API client
func New() (*IMVU, error) {
	client, err := NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &IMVU{
		client: client,
	}, nil
}

// WithClient sets a custom client for the IMVU API
func WithClient(client *Client) func(*IMVU) {
	return func(i *IMVU) {
		i.client = client
	}
}

func NewWithOptions(options ...func(*IMVU)) (*IMVU, error) {
	client, err := NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	imvu := &IMVU{
		client: client,
	}

	for _, option := range options {
		option(imvu)
	}

	return imvu, nil
}

func (i *IMVU) Authenticate(username, password string) error {
	loginPayload := map[string]interface{}{
		"username":               username,
		"password":               password,
		"gdpr_cookie_acceptance": false,
	}

	headers := map[string]string{
		"Origin": "https://pt.secure.imvu.com",
	}

	resp, err := i.client.Post("/login", loginPayload, headers)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var loginResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	return nil
}

func (i *IMVU) Me() (*MeData, error) {
	resp, err := i.client.Get("/login/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	defer resp.Body.Close()

	var res MeResponse
	if err := ParseResponse(resp, &res); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w", err)
	}

	if err := res.ParseMe(); err != nil {
		return nil, fmt.Errorf("failed to parse user data: %w", err)
	}

	return res.Me, nil
}

// GetUser retrieves a user by username
func (i *IMVU) GetUser(username string) (*User, error) {
	resp, err := i.client.Get(fmt.Sprintf("/user/user-%s", username), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var userResp UserResponse
	if err := ParseResponse(resp, &userResp); err != nil {
		return nil, err
	}

	if err := userResp.ParseUser(); err != nil {
		return nil, fmt.Errorf("failed to parse user data: %w", err)
	}

	return userResp.User, nil
}

func (i *IMVU) JoinRoom(sauce string) error {
	headers := map[string]string{
		"X-Imvu-Sauce": sauce,
	}

	resp, err := i.client.Post("/chat/chat-361230062-339/participants", map[string]string{}, headers)
	if err != nil {
		return fmt.Errorf("failed to enter chat: %w", err)
	}

	defer resp.Body.Close()
	var chatResp EnterChatResponse
	if err := ParseResponse(resp, &chatResp); err != nil {
		return fmt.Errorf("failed to parse chat response: %w", err)
	}
	if err := chatResp.ParseEnterChatResponse(); err != nil {
		return fmt.Errorf("failed to parse chat data: %w", err)
	}

	return nil
}

// ConnectWebSocket establishes a WebSocket connection to the IMVU streaming API
func (i *IMVU) ConnectWebSocket() error {
	// Create headers for the WebSocket connection
	headers := http.Header{}

	// Set User-Agent header
	headers.Set("User-Agent", i.client.userAgent)

	// Set Origin header as per example
	headers.Set("Origin", "https://www.imvu.com")

	// Set Host header as per example
	headers.Set("Host", "wss-imq.imvu.com")

	// Get cookies and concatenate into a single Cookie header string
	cookies, err := i.client.GetCookies("https://wss-imq.imvu.com")
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	var cookieStrings []string
	for _, cookie := range cookies {
		cookieStrings = append(cookieStrings, cookie.String())
	}
	if len(cookieStrings) > 0 {
		headers.Set("Cookie", strings.Join(cookieStrings, "; "))
	}

	// Log the connection attempt
	log.Printf("Attempting to connect to WebSocket with headers: %v", headers)

	// Create the WebSocket client with the correct URL
	wsURL := "wss://wss-imq.imvu.com/streaming/imvu_pre"
	i.ws = NewWebSocketClient(wsURL, headers)

	// Set a message handler with detailed logging
	i.ws.SetMessageHandler(func(message []byte) {
		// Log and handle incoming messages
		log.Printf("Received WebSocket message: %s", string(message))

		// Try to parse the message
		var msgData map[string]interface{}
		if err := json.Unmarshal(message, &msgData); err == nil {
			log.Printf("Parsed message data: %v", msgData)
		}
	})

	// Connect to the WebSocket server
	if err := i.ws.Connect(); err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	log.Printf("WebSocket connection established successfully")
	return nil
}

// CloseWebSocket closes the WebSocket connection
func (i *IMVU) CloseWebSocket() error {
	if i.ws == nil {
		return nil
	}
	return i.ws.Close()
}

// SendWebSocketMessage sends a message over the WebSocket connection
func (i *IMVU) SendWebSocketMessage(message interface{}) error {
	if i.ws == nil {
		return fmt.Errorf("WebSocket not connected")
	}
	return i.ws.SendMessage(message)
}

// SendConnect sends a connect message over the WebSocket connection
func (i *IMVU) SendConnect(userID, cookie string) error {
	if i.ws == nil {
		return fmt.Errorf("WebSocket not connected")
	}
	return i.ws.SendConnect(userID, cookie)
}

// SendSubscribe sends a subscribe message over the WebSocket connection
func (i *IMVU) SendSubscribe(name string, opID int) error {
	if i.ws == nil {
		return fmt.Errorf("WebSocket not connected")
	}
	return i.ws.SendSubscribe(name, opID)
}

// SendChatMessage sends a chat message over the WebSocket connection
func (i *IMVU) SendChatMessage(queue, mount, message string, opID int) error {
	if i.ws == nil {
		return fmt.Errorf("WebSocket not connected")
	}
	return i.ws.SendChatMessage(queue, mount, message, opID)
}

// SendPing sends a ping message over the WebSocket connection
func (i *IMVU) SendPing() error {
	if i.ws == nil {
		return fmt.Errorf("WebSocket not connected")
	}
	return i.ws.SendPing()
}

// IsWebSocketConnected returns whether the WebSocket is currently connected
func (i *IMVU) IsWebSocketConnected() bool {
	if i.ws == nil {
		return false
	}
	return i.ws.IsConnected()
}

// GetCookies returns all cookies for a given URL
func (i *IMVU) GetCookies(urlStr string) ([]*http.Cookie, error) {
	return i.client.GetCookies(urlStr)
}
