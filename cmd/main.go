package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	ownerID, chatroomID := getRoomIDsFromURL(roomURL)

	err = client.JoinRoom(ownerID, chatroomID)
	if err != nil {
		log.Fatalf("Failed to join room: %v", err)
	}

	err = client.SendChatMessage("Hii gomp senpai :3")
	if err != nil {
		log.Printf("Failed to send chat message: %v", err)
	}

	stop := make(chan os.Signal, 1)
	// Listen for termination signals including those sent by debuggers
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		start := time.Now()
		for {
			time.Sleep(90 * time.Second)
			log.Printf("Current time: %s, Uptime: %s\n", time.Now().Format(time.RFC3339), time.Since(start))
			err = client.JoinRoom(ownerID, chatroomID)
			if err != nil {
				log.Fatalf("Failed to join room: %v", err)
			}
		}
	}()

	<-stop

	log.Printf("Shuting down...\n")

	if err := client.LeaveRoom(ownerID, chatroomID); err != nil {
		log.Printf("Error leaving room: %v", err)
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
