package bot

import (
	"giiny/internal/gemini"
	"giiny/internal/imvu"
	"log"
	"strings"
)

const senpaiID = ""

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
	go handleIncomingChatMessages(client)

	<-doneCh

	client.LeaveRoom(roomOwner, chatID)
	return nil
}

func handleIncomingChatMessages(client *imvu.IMVU) {
	for {
		message := <-client.ChatMessageChannel

		msg := message.Message
		firstCh := msg[0]
		switch firstCh {
		case '!':
			runCommand(msg[1:])
		case '*':
			// imvu client commands, ignore for now.
		default:
			if message.UserID.String() == client.UserID || message.UserID.String() != senpaiID {
				continue
			}
			log.Printf("Message: %s", message.Message)
			response, err := gemini.Process(message.Message)
			if err != nil {
				log.Printf("Error processing message with Gemini: %v", err)
				continue
			}
			for _, msgs := range response {
				client.SendChatMessage(msgs)
			}
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
