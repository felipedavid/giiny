package main

import (
	"log"
	"os"
	"time"

	"giiny/internal/imvu"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../.env")

	imvu, err := imvu.New()
	if err != nil {
		log.Fatalf("Failed to create IMVU instance: %v", err)
	}

	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	err = imvu.Login(username, password)
	if err != nil {
		log.Fatalf("Failed to login: %v", err)
	}

	err = imvu.JoinRoom("361230062", "140")
	if err != nil {
		log.Fatalf("Failed to join room: %v", err)
	}

	imvu.SendChatMessage("Hi there")

	err = imvu.LeaveRoom("361230062", "140")
	if err != nil {
		log.Fatalf("Failed to leave room: %v", err)
	}

	for {
		time.Sleep(1 * time.Second)
	}
}
