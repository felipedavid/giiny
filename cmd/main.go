package main

import (
	"log"
	"os"
	"strings"

	"giiny/internal/bot"
	"giiny/internal/gemini"
	"giiny/internal/imvu"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../.env")

	gemini.Start()

	client, err := imvu.New()
	if err != nil {
		log.Fatalf("Failed to create IMVU instance: %v", err)
	}

	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")

	roomURL := os.Getenv("ROOM_URL")
	ownerID, chatroomID := getRoomIDsFromURL(roomURL)

	err = bot.Start(username, password, ownerID, chatroomID, client)
	if err != nil {
		log.Fatalf("Something went wrong")
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
