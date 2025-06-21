package imvu

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type OperationID struct {
	ID int
	sync.Mutex
}

func (o *OperationID) GetNew() int {
	o.Lock()
	defer o.Unlock()

	// Increment the ID and return it
	result := o.ID
	o.ID++
	return result
}

type Room struct {
	OwnerID    string
	ChatroomID string
	ChatQueue  string
}

type IMVU struct {
	Authenticated      bool
	UserID             string
	User               *User
	sauce              string
	api                *API
	opID               *OperationID
	currentRoom        *Room
	roomCancelFunc     context.CancelFunc
	ChatMessageChannel chan ChatMessagePayload
}

func New() (*IMVU, error) {
	imvu := &IMVU{
		opID: &OperationID{},
	}

	api, err := NewAPI(imvu.opID)
	if err != nil {
		return nil, fmt.Errorf("failed to create IMVU API client: %w", err)
	}

	imvu.api = api
	return imvu, nil
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

	i.ChatMessageChannel = make(chan ChatMessagePayload)

	err = i.api.ConnectMsgStream(i.UserID, i.ChatMessageChannel)
	if err != nil {
		return fmt.Errorf("failed to connect to messages stream: %w", err)
	}

	queues := []string{
		"inv:/user/user-%s",
		"private:/user/user-%s",
		"/user/%s",
		"inv:/wallet/wallet-%s",
		"inv:/roulette/roulette-%s",
		"inv:/store_catalog/store_catalog-next",
		//"inv:/user/user-362179840",
		//"inv:/eligible_quest_event/eligible_quest_event-%s-309",
		//"inv:/eligible_quest_event/eligible_quest_event-%s-300",
		"inv:/profile/%s",
		"inv:/profile/user-%s",
		"inv:/cart/cart-%s",
		//"inv:/user/user-379408304",
		//"inv:/user/user-379942485",
		//"inv:/user/user-375462516",
		//"inv:/user/user-371103562",
		//"inv:/user/user-361230062",
		//"inv:/user/user-375176415",
		//"inv:/user/user-380315149",
		//"inv:/user/user-237374487",
		//"inv:/user/user-379440992",
		//"inv:/account_order/account_order-co67370135",
		//"inv:/account_order/account_order-co67369562",
		//"inv:/account_order/account_order-co67369497",
		//"inv:/account_order/account_order-1694849152",
		//"inv:/account_order/account_order-1694848877",
		//"inv:/account_order/account_order-1694848293",
		"inv:/avatar/avatar-%s",
	}

	time.Sleep(time.Second * 1)
	for _, qName := range queues {
		if strings.Contains(qName, "%s") {
			qName = fmt.Sprintf(qName, i.UserID)
		}
		i.api.SubscribeToQueue(qName, i.opID.GetNew())
		time.Sleep(time.Millisecond * 200)
	}

	i.api.client.AddHeader("X-Imvu-Application", "next_desktop/1")
	i.api.client.AddHeader("X-Imvu-Sauce", me.Sauce)
	i.sauce = me.Sauce
	i.Authenticated = true
	i.User = user

	return nil
}

func (i *IMVU) JoinRoom(roomID, roomChatID string) error {
	if i.roomCancelFunc != nil {
		i.roomCancelFunc()
	}

	err := i.api.JoinRoom(roomID, roomChatID)
	if err != nil {
		return fmt.Errorf("failed to join room: %w", err)
	}

	var ctx context.Context
	ctx, i.roomCancelFunc = context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Printf("Rejoining room %s-%s after 1 minute", roomID, roomChatID)
				err := i.api.JoinRoom(roomID, roomChatID)
				if err != nil {
					log.Printf("Failed to rejoin room %s-%s: %v", roomID, roomChatID, err)
				}
			case <-ctx.Done():
				log.Printf("Stopping rejoining room %s-%s", roomID, roomChatID)
				return
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Printf("Changing availability for user %s", i.UserID)
				err := i.api.ChangeAvalability(i.UserID)
				if err != nil {
					log.Printf("Failed to change availability for user %s: %v", i.UserID, err)
				}
			case <-ctx.Done():
				log.Printf("Stopping availability changes for user %s", i.UserID)
				return
			}
		}
	}()

	sceneQueue := fmt.Sprintf("inv:/scene/scene-%s-%s", roomID, roomChatID)
	i.api.SubscribeToQueue(sceneQueue, i.opID.GetNew())

	roomQueue := fmt.Sprintf("inv:/room/room-%s-%s", roomID, roomChatID)
	i.api.SubscribeToQueue(roomQueue, i.opID.GetNew())

	chatQueue, err := i.api.GetRoomChatQueue(roomID, roomChatID)
	if err != nil {
		return fmt.Errorf("failed to get room chat ID: %w", err)
	}
	i.api.SubscribeToQueue(chatQueue, i.opID.GetNew())

	i.currentRoom = &Room{
		OwnerID:    roomID,
		ChatroomID: roomChatID,
		ChatQueue:  chatQueue,
	}

	time.Sleep(1 * time.Second)

	// TODO: Test how CmdPutOnOutfit and CmdUse work. Maybe create a function to handle the player outfits?
	outfitItemIDS := []string{
		"69320200", "70312022", "12444122", "13831030", "16070306", "19442649", "23974249", "55139083", "55595518", "63520397", "63520471", "70082645", "70082730", "55595754", "61753525", "62845575", "59508957", "63520653", "63520746",
	}

	i.Exec(CmdImvuIsPureUser)
	i.Exec(CmdPutOnOutfit, outfitItemIDS...)
	i.Exec(CmdUse, outfitItemIDS...)

	return nil
}

func (i *IMVU) LeaveRoom(roomID, chatID string) error {
	if i.roomCancelFunc != nil {
		i.roomCancelFunc()
		i.roomCancelFunc = nil
	}

	err := i.api.LeaveRoom(roomID, chatID, i.UserID)
	if err != nil {
		return fmt.Errorf("failed to leave room: %w", err)
	}

	i.currentRoom = nil
	return nil
}

func (i *IMVU) SendChatMessage(message string) error {
	if i.currentRoom == nil {
		return fmt.Errorf("not in a room, cannot send message")
	}

	room := i.currentRoom

	payload := ChatMessagePayload{
		ChatID:  StringOrInt(room.ChatroomID),
		Message: message,
		To:      StringOrInt("0"),
		UserID:  StringOrInt(i.UserID),
	}

	i.api.SendChatMessage(
		room.ChatQueue,
		"messages",
		payload,
	)
	return nil
}
