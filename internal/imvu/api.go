package imvu

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// API represents the API API client
type API struct {
	client *HTTPClient
	ws     *WebSocketClient
}

// New creates a new IMVU API client
func NewAPI() (*API, error) {
	client, err := NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &API{
		client: client,
	}, nil
}

func (i *API) Authenticate(username, password string) error {
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

func (i *API) Me() (*MeData, error) {
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

func (i *API) GetUser(userID string) (*User, error) {
	resp, err := i.client.Get(fmt.Sprintf("/user/user-%s", userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var res UserResponse
	if err := ParseResponse(resp, &res); err != nil {
		return nil, err
	}

	if err := res.ParseUser(); err != nil {
		return nil, fmt.Errorf("failed to parse user data: %w", err)
	}

	return res.User, nil
}

func (i *API) JoinRoom(ownerID, chatroomID string) error {
	resp, err := i.client.Post(fmt.Sprintf("/chat/chat-%s-%s/participants", ownerID, chatroomID), map[string]string{}, nil)
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

func (i *API) GetChat(roomID, chatID string) (*BaseResponse, error) {
	resp, err := i.client.Get(fmt.Sprintf("/chat/chat-%s-%s", roomID, chatID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	var chatResp BaseResponse
	if err := ParseResponse(resp, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse chat response: %w", err)
	}

	return &chatResp, nil
}

func (i *API) GetRoomChatQueue(roomID, roomChatID string) (string, error) {
	chat, err := i.GetChat(roomID, roomChatID)
	if err != nil {
		return "", fmt.Errorf("failed to get chat: %w", err)
	}

	entityID := fmt.Sprintf("https://api.imvu.com/chat/chat-%s-%s", roomID, roomChatID)

	type ChatData struct {
		ImqQueue string `json:"imq_queue"`
	}

	chatData, err := ExtractEntity[ChatData](chat, entityID)
	if err != nil {
		return "", fmt.Errorf("failed to extract chat data: %w", err)
	}

	return chatData.ImqQueue, nil
}

func (i *API) LeaveRoom(roomID, chatID, userID string) error {
	resp, err := i.client.Delete(fmt.Sprintf("/chat/chat-%s-%s/participants/user-%s", roomID, chatID, userID), nil)
	if err != nil {
		return fmt.Errorf("failed to leave chat: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to leave chat with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (i *API) ConnectMsgStream(userID string, ch chan ChatMessagePayload) error {
	headers := http.Header{}

	headers.Set("User-Agent", i.client.userAgent)
	headers.Set("Origin", "https://www.imvu.com")
	headers.Set("Host", "wss-imq.imvu.com")
	headers.Set("Server", "Cowboy")

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

	wsURL := "wss://wss-imq.imvu.com/streaming/imvu_pre"
	i.ws = NewWebSocketClient(wsURL, headers)

	i.ws.SetMessageHandler(func(message []byte) {
		//log.Printf("Received WebSocket message: %s", string(message))

		//var msgData map[string]interface{}
		//if err := json.Unmarshal(message, &msgData); err == nil {
		//	log.Printf("Parsed message data: %v", msgData)
		//}
	})

	if err := i.ws.Connect(ch); err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	// Get the osCsid cookie from the HTTP client
	cookies, err = i.GetCookies("https://wss-imq.imvu.com")
	if err != nil {
		log.Fatalf("Failed to get cookies: %v", err)
	}

	// Find the osCsid cookie
	var osCsid string
	for _, cookie := range cookies {
		if cookie.Name == "osCsid" {
			osCsid = cookie.Value
			break
		}
	}

	if osCsid == "" {
		log.Println("Warning: osCsid cookie not found, using empty value")
	}

	err = i.SendConnect(userID, osCsid)
	if err != nil {
		log.Printf("Failed to send connect message: %v", err)
	} else {
		fmt.Println("Connect message sent successfully")
	}

	return nil
}

func (i *API) CloseWebSocket() error {
	if i.ws == nil {
		return nil
	}
	return i.ws.Close()
}

func (i *API) SendWebSocketMessage(message interface{}) error {
	if i.ws == nil {
		return fmt.Errorf("WebSocket not connected")
	}
	return i.ws.SendMessage(message)
}

func (i *API) SendConnect(userID, cookie string) error {
	if i.ws == nil {
		return fmt.Errorf("WebSocket not connected")
	}
	return i.ws.SendConnect(userID, cookie)
}

func (i *API) SubscribeToQueue(queue string, opID int) error {
	if i.ws == nil {
		return fmt.Errorf("WebSocket not connected")
	}
	return i.ws.SendSubscribeToQueue(queue, opID)
}

func (i *API) SendChatMessage(queue, mount string, payload ChatMessagePayload) error {
	if i.ws == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	return i.ws.SendChatMessage(queue, mount, payload)
}

func (i *API) SendPing() error {
	if i.ws == nil {
		return fmt.Errorf("WebSocket not connected")
	}
	return i.ws.SendPing()
}

func (i *API) IsWebSocketConnected() bool {
	if i.ws == nil {
		return false
	}
	return i.ws.IsConnected()
}

func (i *API) GetCookies(urlStr string) ([]*http.Cookie, error) {
	return i.client.GetCookies(urlStr)
}
