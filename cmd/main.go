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

	roomID := os.Getenv("ROOM_ID")
	ownerID, chatroomID := decomposeRoomID(roomID)

	err = bot.Start(username, password, ownerID, chatroomID, client)
	if err != nil {
		log.Fatalf("Something went wrong")
	}
}

func decomposeRoomID(roomURL string) (string, string) {
	roomIDSplit := strings.Split(roomURL, "-")
	if len(roomIDSplit) < 3 {
		return "", ""
	}

	return roomIDSplit[1], roomIDSplit[2]
}
