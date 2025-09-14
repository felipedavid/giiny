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
	opID   *OperationID
}

// New creates a new IMVU API client
func NewAPI(opID *OperationID) (*API, error) {
	client, err := NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &API{
		client: client,
		opID:   opID,
	}, nil
}

func (i *API) Authenticate(username, password string) error {
	loginPayload := map[string]any{
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

	var loginResponse map[string]any
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

func (i *API) JoinChatRoom(ownerID, chatroomID string) error {
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

func (i *API) LeaveChatRoom(ownerID, chatID, userID string) error {
	resp, err := i.client.Delete(fmt.Sprintf("/chat/chat-%s-%s/participants/user-%s", ownerID, chatID, userID), nil)
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

func (i *API) ChangeAvalability(userID string) error {
	resp, err := i.client.Post(fmt.Sprintf("/user/user-%s", userID), map[string]any{
		"availability": "Available",
		"online":       true,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to change availability: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to change availability with status %d: %s", resp.StatusCode, string(bodyBytes))
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

func (i *API) ConnectMsgStream(userID string, ch chan ChatMessagePayload) error {
	headers := http.Header{}
	headers.Set("User-Agent", i.client.userAgent)
	headers.Set("Origin", "https://www.imvu.com")

	cookies, err := i.client.GetCookies("https://wss-imq.imvu.com")
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	var cookieStrings []string
	var osCsid string
	for _, cookie := range cookies {
		cookieStrings = append(cookieStrings, cookie.String())
		if cookie.Name == "osCsid" {
			osCsid = cookie.Value
		}
	}
	if len(cookieStrings) > 0 {
		headers.Set("Cookie", strings.Join(cookieStrings, "; "))
	}

	if osCsid == "" {
		log.Println("Warning: osCsid cookie not found, using empty value")
	}

	wsURL := "wss://wss-imq.imvu.com/streaming/imvu_pre"
	config := Config{
		URL:       wsURL,
		Headers:   headers,
		UserID:    userID,
		SessionID: osCsid,
		OpID:      i.opID,
		Metadata: map[string]string{
			"app":           "imvu_next",
			"platform_type": "big",
		},
		OnMessage: func(message *Message) {
			switch message.Record {
			case RecordSendMessage:
				// Re-marshal the message to get it into a byte slice
				payloadBytes, err := json.Marshal(message)
				if err != nil {
					log.Printf("Failed to re-marshal send message payload: %v", err)
					return
				}

				var payload WebSocketSendMessageMessage
				if err := json.Unmarshal(payloadBytes, &payload); err != nil {
					log.Printf("Failed to parse send message payload: %v", err)
					return
				}

				// Now we need to convert payload.Message to ChatMessagePayload
				chatMessageBytes, err := json.Marshal(payload.Message)
				if err != nil {
					log.Printf("Failed to marshal inner chat message: %v", err)
					return
				}

				var chatMessage ChatMessagePayload
				if err := json.Unmarshal(chatMessageBytes, &chatMessage); err != nil {
					log.Printf("Failed to unmarshal inner chat message: %v", err)
					return
				}

				ch <- chatMessage
			case RecordJoinedQueue:
				if strings.Contains(*message.Queue, "/chat") {
					log.Printf("User %s joined the chat!!! (queue: %s)", *message.UserID, *message.Queue)
				}
			case RecordLeftQueue:
				if strings.Contains(*message.Queue, "/chat") {
					log.Printf("User %s left the chat!!! (queue: %s)", *message.UserID, *message.Queue)
				}
			}
		},
	}

	i.ws = NewWebSocketClient(config)
	i.ws.Connect()

	return nil
}

type RecordKind string

const (
	RecordPing        RecordKind = "msg_c2g_ping"
	RecordPong        RecordKind = "msg_g2c_pong"
	RecordSendMessage RecordKind = "msg_g2c_send_message"
	RecordJoinedQueue RecordKind = "msg_g2c_joined_queue"
	RecordLeftQueue   RecordKind = "msg_g2c_left_queue"
)

type Message struct {
	Record       RecordKind `json:"record"`
	UserID       *string    `json:"user_id"`
	Status       *float64   `json:"status"`
	ErrorMessage *string    `json:"error_message"`
	Queue        *string    `json:"queue"`
	Mount        *string    `json:"mount"`
	Message      *string    `json:"message"`
	Sequence     *int       `json:"sequence"`
}

func (i *API) CloseWebSocket() {
	if i.ws != nil {
		i.ws.Close()
	}
}

func (i *API) SendWebSocketMessage(record string, payload map[string]any) {
	if i.ws != nil {
		i.ws.Send(record, payload)
	}
}

func (i *API) SubscribeToQueue(queue string, opID int) {
	if i.ws == nil {
		log.Println("WebSocket not connected")
		return
	}
	subscription := map[string]any{
		"record": "subscription",
		"name":   queue,
		"op_id":  opID,
	}
	payload := map[string]any{
		"queues_with_results": []any{subscription},
	}
	i.SendWebSocketMessage("msg_c2g_subscribe", payload)
}

func (i *API) SendChatMessage(queue, mount string, payload ChatMessagePayload) {
	if i.ws == nil {
		log.Println("WebSocket not connected")
		return
	}

	message := map[string]any{
		"queue":   queue,
		"mount":   mount,
		"message": payload,
		"op_id":   i.opID.GetNew(),
	}

	i.SendWebSocketMessage("msg_c2g_send_message", message)
}

func (i *API) IsWebSocketConnected() bool {
	if i.ws == nil {
		return false
	}
	return i.ws.GetState() == StateAuthenticated
}

func (i *API) GetCookies(urlStr string) ([]*http.Cookie, error) {
	return i.client.GetCookies(urlStr)
}

func (i *API) SearchRooms(userID string, params map[string]string) (*RoomSearchResponse, error) {
	url := fmt.Sprintf("/room_list/room_list-%s-explore/rooms", userID)
	resp, err := i.client.Get(url, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search for rooms: %w", err)
	}

	var res RoomSearchResponse
	if err := ParseResponse(resp, &res); err != nil {
		return nil, fmt.Errorf("failed to parse room search response: %w", err)
	}

	if err := res.ParseRooms(); err != nil {
		return nil, fmt.Errorf("failed to parse rooms data: %w", err)
	}

	return &res, nil
}
