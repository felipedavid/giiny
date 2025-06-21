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
			log.Printf("[%s] Incoming IMVU command: %s", msg.UserID, msg.Message[1:])
		default:
			log.Printf("Message: %s", msg.Message)
			response, err := gemini.Process(msg.Message)
			if err != nil {
				log.Printf("Error processing message with Gemini: %v", err)
				continue
			}
			sentences := strings.Split(response, ";")
			for _, sentence := range sentences {
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
	case "dress":
		outfitItemIDS := []string{
			"69320200", "70312022", "12444122", "13831030", "16070306", "19442649", "23974249", "55139083", "55595518", "63520397", "63520471", "70082645", "70082730", "55595754", "61753525", "62845575", "59508957", "63520653", "63520746",
		}

		client.Exec(imvu.CmdPutOnOutfit, outfitItemIDS...)
		client.Exec(imvu.CmdUse, outfitItemIDS...)
	case "lap":
		client.SendChatMessage("Colinhooo!! uwu *tomato*")
		client.Exec(imvu.CmdMsg, "SeatAssignment 2 361230062 101 99982")
	}
}
