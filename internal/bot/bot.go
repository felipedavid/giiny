package bot

import (
	"giiny/internal/imvu"
	"log"
	"strings"
)

var doneCh chan bool

func Start(username, password, roomOwner, chatID string, client *imvu.IMVU) error {
	doneCh = make(chan bool)

	log.Printf("Trying to login as %s", username)
	err := client.Login(username, password)
	if err != nil {
		return err
	}

	log.Printf("Login successful!")
	log.Printf("Trying to join a room.")

	err = client.JoinRoom(roomOwner, chatID)
	if err != nil {
		return err
	}

	log.Printf("Joined successfully, starting to consume messages")
	go handleIncomingChatMessages(client.ChatMessageChannel)

	<-doneCh

	client.LeaveRoom(roomOwner, chatID)
	return nil
}

func handleIncomingChatMessages(ch chan imvu.ChatMessagePayload) {
	for {
		message := <-ch

		msg := message.Message
		firstCh := msg[0]
		switch firstCh {
		case '!':
			runCommand(msg[1:])
		case '*':
			// imvu client commands, ignore for now.
		default:
			log.Printf("Message: %s", message.Message)
		}
	}
}

func runCommand(cmd string) {
	cmd = strings.ToLower(cmd)

	log.Printf("Trying to run command: %s", cmd)

	switch cmd {
	case "quit":
		doneCh <- true
	}
}
