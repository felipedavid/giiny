package imvu

import (
	"fmt"
	"strings"
	"sync"
)

var OpID = OperationID{ID: 57}

type OperationID struct {
	ID int
	sync.Mutex
}

func (o *OperationID) Get() int {
	o.Lock()
	defer o.Unlock()

	// Increment the ID and return it
	result := o.ID
	o.ID++
	return result
}

type Room struct {
	ID        string
	ChatID    string
	ChatQueue string
}

type IMVU struct {
	Authenticated bool
	UserID        string
	User          *User
	sauce         string
	api           *API
	currentRoom   *Room
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

	i.api.client.AddHeader("X-Imvu-Sauce", me.Sauce)

	i.sauce = me.Sauce
	i.Authenticated = true
	i.User = user

	return nil
}

func (i *IMVU) JoinRoom(roomID, roomChatID string) error {
	chatQueue, err := i.api.GetRoomChatQueue(roomID, roomChatID)
	if err != nil {
		return fmt.Errorf("failed to get room chat ID: %w", err)
	}

	err = i.api.JoinRoom(i.sauce, roomID, roomChatID)
	if err != nil {
		return fmt.Errorf("failed to join room: %w", err)
	}

	err = i.api.SendSubscribe(fmt.Sprintf("inv:/scene/scene-%s-%s", roomID, roomChatID), OpID.Get())
	if err != nil {
		return fmt.Errorf("failed to send scene subscribe message: %w", err)
	}

	err = i.api.SendSubscribe(fmt.Sprintf("inv:/room/room-%s-%s", roomID, roomChatID), OpID.Get())
	if err != nil {
		return fmt.Errorf("failed to send scene subscribe message: %w", err)
	}

	err = i.api.SendSubscribe(chatQueue, OpID.Get())
	if err != nil {
		return fmt.Errorf("failed to send chat subscribe message: %w", err)
	}

	i.currentRoom = &Room{
		ID:        roomID,
		ChatID:    roomChatID,
		ChatQueue: chatQueue,
	}

	return nil
}

func (i *IMVU) LeaveRoom(roomID, chatID string) error {
	err := i.api.LeaveRoom(roomID, chatID, i.UserID)
	if err != nil {
		return fmt.Errorf("failed to leave room: %w", err)
	}

	return nil
}

func (i *IMVU) SendChatMessage(message string) error {
	if i.currentRoom == nil {
		return fmt.Errorf("not in a room, cannot send message")
	}

	room := i.currentRoom

	payload := ChatMessagePayload{
		ChatID:  room.ChatID,
		Message: message,
		To:      0,
		UserID:  i.UserID,
	}

	err := i.api.SendChatMessage(
		room.ChatQueue,
		"messages",
		payload,
	)
	return err
}
