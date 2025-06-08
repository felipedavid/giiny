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

	imvu.SendChatMessage("*imvu:isPureUser")
	imvu.SendChatMessage("*putOnOutfit 70312022 12444122 13831030 16070306 19442649 23974249 55139083 55595518 63520397 63520471 70082645 70082730 55595754 61753525 62845575 59508957 63520653 63520746")
	imvu.SendChatMessage("*msg SeatAssignment 3 373088882 2 0")

	//err = imvu.LeaveRoom("361230062", "140")
	//if err != nil {
	//	log.Fatalf("Failed to leave room: %v", err)
	//}

	for {
		time.Sleep(1 * time.Second)
	}
}
