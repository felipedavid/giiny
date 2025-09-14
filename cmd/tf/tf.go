package main

import (
	"log"
	"os"
	"strings"

	"giiny/internal/imvu"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	api, err := imvu.NewAPI(&imvu.OperationID{})
	if err != nil {
		log.Fatalf("Failed to create new api: %s", err.Error())
	}

	err = api.Authenticate(os.Getenv("USERNAME"), os.Getenv("PASSWORD"))
	if err != nil {
		log.Fatalf("Failed to authenticate: %s", err.Error())
	}

	me, err := api.Me()
	if err != nil {
		log.Fatal(err)
	}

	uIDSplit := strings.Split(me.User.ID, "/")
	userID := uIDSplit[len(uIDSplit)-1]

	log.Println("Successfully authenticated")

	var allRooms []imvu.Room
	var next string

	for {
		params := map[string]string{
			"limit":               "350",
			"rating":              "ga",
			"room_type":           "all",
			"scene_occupancy_max": "2",
			"scene_occupancy_min": "1",
			"language":            "pt",
		}

		if next != "" {
			params["next"] = next
		}

		resp, err := api.SearchRooms(userID, params)
		if err != nil {
			log.Fatalf("Failed to search rooms: %s", err.Error())
		}

		allRooms = append(allRooms, resp.Rooms...)

		roomListData, err := imvu.ExtractEntity[imvu.RoomListData](&resp.BaseResponse, resp.ID)
		if err != nil {
			log.Fatalf("Failed to extract room list data: %s", err.Error())
		}

		if roomListData.Next == "" {
			break
		}
		next = roomListData.Next
	}

	log.Printf("Total rooms fetched: %d", len(allRooms))
}
