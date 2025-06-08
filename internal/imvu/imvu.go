package imvu

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

type IMVU struct {
	Authenticated bool
	UserID        string
	User          *User
	sauce         string
	api           *API
}

func New() (*IMVU, error) {
	api, err := NewAPI()
	if err != nil {
		return nil, fmt.Errorf("failed to create IMVU API client: %w", err)
	}

	return &IMVU{
		api: api,
	}, nil
}

func (i *IMVU) Login(username, password string) error {
	err := i.api.Authenticate(username, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	me, err := i.api.Me()
	if err != nil {
		return fmt.Errorf("failed to retrieve 'me' data: %w", err)
	}

	urlFields := strings.Split(me.User.ID, "/")
	i.UserID = urlFields[len(urlFields)-1]

	user, err := i.api.GetUser(i.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	err = i.api.ConnectMsgStream(i.UserID)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	i.sauce = me.Sauce
	i.Authenticated = true
	i.User = user

	return nil
}

func (i *IMVU) JoinRoom(roomID, chatID string) error {
	err := i.api.JoinRoom(i.sauce, roomID, chatID)
	if err != nil {
		return fmt.Errorf("failed to join room: %w", err)
	}

	err = i.api.SendSubscribe(fmt.Sprintf("inv:/scene/scene-%s-%s", roomID, chatID), 148)
	if err != nil {
		return fmt.Errorf("failed to send scene subscribe message: %w", err)
	}

	err = i.api.SendSubscribe("/chat/1285983375", 153)
	if err != nil {
		return fmt.Errorf("failed to send chat subscribe message: %w", err)
	}

	return nil
}

func (i *IMVU) SendChatMessage(message string) error {
	err := i.api.SendChatMessage(
		"/chat/1285983375",
		"messages",
		generateMessage(message, i.UserID),
		154,
	)
	return err
}

func generateMessage(message, userID string) string {
	// Create the chat message payload
	type ChatMessagePayload struct {
		ChatID  string `json:"chatId"`
		Message string `json:"message"`
		To      int    `json:"to"`
		UserID  string `json:"userId"`
	}

	chatPayload := ChatMessagePayload{
		ChatID:  "140",
		Message: message,
		To:      0,
		UserID:  userID,
	}

	// Marshal the payload to JSON
	payloadJSON, err := json.Marshal(chatPayload)
	if err != nil {
		log.Printf("Failed to marshal chat payload: %v", err)
	}

	// Base64 encode the JSON payload
	encodedPayload := base64.StdEncoding.EncodeToString(payloadJSON)

	return encodedPayload
}
