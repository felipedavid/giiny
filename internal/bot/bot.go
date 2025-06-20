package bot

import (
	"fmt"
	"giiny/internal/gemini"
	"giiny/internal/imvu"
	"log"
	"strings"
	"time"
)

var startTime time.Time

const senpaiID = "361230062"

var doneCh chan bool

func Start(username, password, roomOwner, chatID string, client *imvu.IMVU) error {
	doneCh = make(chan bool)

	log.Printf("Trying to login as %s", username)
	err := client.Login(username, password)
	if err != nil {
		return err
	}

	startTime = time.Now()

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
		msg := <-client.ChatMessageChannel

		if len(msg.Message) == 0 || msg.UserID.String() == client.UserID || msg.UserID.String() != senpaiID {
			continue
		}

		firstCh := msg.Message[0]
		switch firstCh {
		case '!':
			runCommand(client, msg.Message[1:])
		case '*':
			// imvu client commands, ignore for now.
		default:
			log.Printf("Message: %s", msg.Message)
			response, err := gemini.Process(msg.Message)
			if err != nil {
				log.Printf("Error processing message with Gemini: %v", err)
				continue
			}
			sentences := strings.Split(response, ";")
			for _, sentence := range sentences {
				time.Sleep(1 * time.Second)
				sentence = strings.TrimSpace(sentence)
				if len(sentence) > 0 {
					log.Printf("Sending response: %s", sentence)
					client.SendChatMessage(sentence)
				}
			}
		}
	}
}

func runCommand(client *imvu.IMVU, cmd string) {
	cmd = strings.ToLower(cmd)

	log.Printf("Trying to run command: %s", cmd)

	switch cmd {
	case "quit":
		doneCh <- true
	case "uptime":
		msg := fmt.Sprintf("Uptime: %s", time.Since(startTime))
		client.SendChatMessage(msg)
	}
}
