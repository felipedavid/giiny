package bot

import (
	"bufio"
	"fmt"
	"giiny/internal/gemini"
	"giiny/internal/imvu"
	"log"
	"os"
	"strings"
	"time"
)

var startTime time.Time
var pause bool = false

const senpaiID = "361230062"
const giinyID = "373088882"

var doneCh chan bool

var client *imvu.IMVU
var room *imvu.IMVURoom

func Start(username, password, roomOwner, chatID string, iclient *imvu.IMVU) error {
	client = iclient
	doneCh = make(chan bool)

	log.Printf("Trying to login as %s", username)
	err := iclient.Login(username, password)
	if err != nil {
		return err
	}

	startTime = time.Now()

	log.Printf("Login successful!")
	log.Printf("Trying to join a room.")

	room, err = iclient.JoinRoom(roomOwner, chatID)
	if err != nil {
		return err
	}

	log.Printf("Joined successfully, starting to consume messages")
	go handleIncomingChatMessages(iclient)

	<-doneCh

	iclient.LeaveRoom(room)
	return nil
}

func handleIncomingChatMessages(client *imvu.IMVU) {
	for {
		msg := <-client.ChatMessageChannel

		if len(msg.Message) == 0 || msg.UserID.String() == client.UserID || msg.UserID.String() != senpaiID {
			continue
		}

		if msg.UserID.String() == giinyID {
			continue
		}

		firstCh := msg.Message[0]
		switch firstCh {
		case '!':
			runCommand(msg.Message[1:])
		default:
			log.Printf("User: %s, Message: %s", msg.UserID, msg.Message)

			if pause {
				fmt.Println("Bot is paused, ignoring message.")
				continue
			}

			if !strings.Contains(msg.Message, "@giiny") {
				continue
			}

			client.ExecMsg(room, imvu.MsgCmdBeginText, "2", giinyID, "0", "0")
			time.Sleep(2 * time.Second)

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
					client.SendChatMessage(room, sentence)
					time.Sleep(1 * time.Second)
				}
			}

			client.ExecMsg(room, imvu.MsgCmdEraseText, "2", giinyID, "0", "0")
		}
	}
}

func runCommand(cmd string) error {
	cmds := strings.Split(cmd, " ")
	cmd = cmds[0]

	log.Printf("Trying to run command: %s", cmd)

	switch cmd {
	case CmdQuit:
		doneCh <- true
	case CmdUptime:
		msg := fmt.Sprintf("Uptime: %s", time.Since(startTime))
		client.SendChatMessage(room, msg)
	case CmdDress:
		client.WearOutfit(room)
	case CmdLap:
		client.SendChatMessage(room, "Colinhooo!! uwu *tomato*")
		client.Exec(room, imvu.CmdMsg, "SeatAssignment 2 361230062 101 99982")
	case CmdPause:
		pause = !pause
	}

	return nil
}

func prompt() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("> ")
		line, _ := reader.ReadString('\n')
		err := runCommand(line)
		if err != nil {
			fmt.Printf("Error trying to run command: %s", err)
		}

		if line == "quit" {
			break
		}
	}
}
