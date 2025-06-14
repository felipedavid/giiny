package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"giiny/internal/imvu"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../.env")

	client, err := imvu.New()
	if err != nil {
		log.Fatalf("Failed to create IMVU instance: %v", err)
	}

	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	err = client.Login(username, password)
	if err != nil {
		log.Fatalf("Failed to login: %v", err)
	}

	roomURL := os.Getenv("ROOM_URL")
	roomID, roomChatID := getRoomIDsFromURL(roomURL)

	err = client.JoinRoom(roomID, roomChatID)
	if err != nil {
		log.Fatalf("Failed to join room: %v", err)
	}

	time.Sleep(5 * time.Second)
	client.SendChatMessage("Hii gomp senpai :3")
	time.Sleep(5 * time.Second)

	err = client.LeaveRoom(roomID, roomChatID)
	if err != nil {
		log.Fatalf("Failed to leave room: %v", err)
	}

	start := time.Now()

	go func() {
		for {
			time.Sleep(1 * time.Minute)
			client.SendChatMessage(fmt.Sprintf("Current time: %s, Uptime: %s", time.Now().Format(time.RFC3339), time.Since(start)))
			err = client.JoinRoom("361230062", "140")
			if err != nil {
				log.Fatalf("Failed to join room: %v", err)
			}
		}
	}()

	for {
		time.Sleep(10 * time.Second)
	}
}

func getRoomIDsFromURL(roomURL string) (string, string) {
	roomURLSplit := strings.Split(roomURL, "/")
	roomURLSplit = strings.Split(roomURLSplit[len(roomURLSplit)-1], "-")
	if len(roomURLSplit) < 3 {
		return "", ""
	}

	return roomURLSplit[1], roomURLSplit[2]
}
