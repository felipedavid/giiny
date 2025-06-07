package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"giiny/internal/imvu"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../.env")

	imvuClient, err := imvu.New()
	if err != nil {
		log.Fatalf("Failed to create IMVU client: %v", err)
	}

	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	err = imvuClient.Authenticate(username, password)
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	fmt.Println("Authentication successful!")

	me, err := imvuClient.Me()
	if err != nil {
		log.Fatalf("Failed to get 'me' data: %v", err)
	}

	urlFields := strings.Split(me.User.ID, "/")
	id := urlFields[len(urlFields)-1]

	// Get user information
	user, err := imvuClient.GetUser(id)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	err = imvuClient.JoinRoom(me.Sauce)
	if err != nil {
		log.Fatalf("Failed to enter chat: %v", err)
	}

	// Display user information
	fmt.Printf("User: %s (Display Name: %s)\n", user.Username, user.DisplayName)
	fmt.Printf("Avatar URL: %s\n", user.AvatarImage)
	fmt.Printf("Online: %t\n", user.Online)
	fmt.Printf("Created: %s\n", user.Created)

	// Connect to WebSocket
	fmt.Println("\nConnecting to WebSocket...")
	err = imvuClient.ConnectWebSocket()
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	fmt.Println("WebSocket connected successfully!")

	// Set up a signal handler to gracefully close the connection
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Wait a moment before sending the connect message
	fmt.Println("Waiting 2 seconds before sending connect message...")
	time.Sleep(2 * time.Second)

	// Extract user ID from the 'me' data
	userID := id

	// Get the osCsid cookie from the HTTP client
	cookies, err := imvuClient.GetCookies("https://wss-imq.imvu.com")
	if err != nil {
		log.Fatalf("Failed to get cookies: %v", err)
	}

	// Find the osCsid cookie
	var osCsid string
	for _, cookie := range cookies {
		if cookie.Name == "osCsid" {
			osCsid = cookie.Value
			break
		}
	}

	if osCsid == "" {
		log.Println("Warning: osCsid cookie not found, using empty value")
	}

	// Send a connect message
	fmt.Println("Sending connect message...")
	err = imvuClient.SendConnect(userID, osCsid)
	if err != nil {
		log.Printf("Failed to send connect message: %v", err)
		// Don't exit on connect failure, try to keep the connection open
	} else {
		fmt.Println("Connect message sent successfully")
	}

	// Wait a moment before sending the subscribe messages
	fmt.Println("Waiting 1 second before sending subscribe messages...")
	time.Sleep(1 * time.Second)

	// Send the first subscribe message for the scene
	fmt.Println("Sending scene subscribe message...")
	err = imvuClient.SendSubscribe("inv:/scene/scene-361230062-339", 50)
	if err != nil {
		log.Printf("Failed to send scene subscribe message: %v", err)
	} else {
		fmt.Println("Scene subscribe message sent successfully")
	}

	// Wait a moment before sending the second subscribe message
	time.Sleep(500 * time.Millisecond)

	// Send the second subscribe message for the chat
	fmt.Println("Sending chat subscribe message...")
	err = imvuClient.SendSubscribe("/chat/1286100305", 55)
	if err != nil {
		log.Printf("Failed to send chat subscribe message: %v", err)
	} else {
		fmt.Println("Chat subscribe message sent successfully")
	}

	// Wait a moment before sending the chat message
	time.Sleep(500 * time.Millisecond)

	err = imvuClient.SendChatMessage(
		"/chat/1286100305",
		"messages",
		generateMessage("*imvu:isPureUser", userID),
		56,
	)
	if err != nil {
		log.Printf("Failed to send chat message: %v", err)
	} else {
		fmt.Println("Chat message sent successfully")
	}

	for i := 0; i < 10; i++ {
		err = imvuClient.SendChatMessage(
			"/chat/1286100305",
			"messages",
			generateMessage(fmt.Sprintf("This is test message %d", i), userID),
			56,
		)
		if err != nil {
			log.Printf("Failed to send chat message: %v", err)
		} else {
			fmt.Println("Chat message sent successfully")
		}
		time.Sleep(200)
	}

	// Wait a moment before sending the ping
	fmt.Println("Waiting 1 second before sending ping message...")
	time.Sleep(1 * time.Second)

	// Send a ping message
	fmt.Println("Sending ping message...")
	err = imvuClient.SendPing()
	if err != nil {
		log.Printf("Failed to send ping: %v", err)
		// Don't exit on ping failure, try to keep the connection open
	} else {
		fmt.Println("Ping sent, waiting for pong response...")
	}

	// Keep the connection alive for a while to receive messages
	fmt.Println("WebSocket connection established. Press Ctrl+C to exit.")

	// Set up a ticker to send pings periodically (every 15 seconds instead of 16)
	ticker := time.NewTicker(16 * time.Second)
	defer ticker.Stop()

	// Add a timeout to automatically exit after 2 minutes if needed for testing
	timeout := time.After(2 * time.Minute)

	// Wait for interrupt signal or ticker
	for {
		select {
		case <-ticker.C:
			if imvuClient.IsWebSocketConnected() {
				fmt.Println("Sending periodic ping...")
				err = imvuClient.SendPing()
				if err != nil {
					log.Printf("Failed to send ping: %v", err)
				}
			} else {
				log.Printf("WebSocket disconnected, attempting to reconnect...")
				err = imvuClient.ConnectWebSocket()
				if err != nil {
					log.Printf("Failed to reconnect: %v", err)
				} else {
					log.Printf("Successfully reconnected")
				}
			}
		case <-timeout:
			fmt.Println("\nTimeout reached, closing connection...")
			imvuClient.CloseWebSocket()
			fmt.Println("WebSocket connection closed.")
			return
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal, closing connection...")
			imvuClient.CloseWebSocket()
			fmt.Println("WebSocket connection closed.")
			return
		}
	}
}

func generateMessage(message, userID string) string {
	// Create the chat message payload
	type ChatMessagePayload struct {
		ChatID  string `json:"chatId"`
		Message string `json:"message"`
		To      int    `json:"to"`
		UserID  string `json:"userId"`
	}

	chatPayload := ChatMessagePayload{
		ChatID:  "339",
		Message: message,
		To:      0,
		UserID:  userID,
	}

	// Marshal the payload to JSON
	payloadJSON, err := json.Marshal(chatPayload)
	if err != nil {
		log.Printf("Failed to marshal chat payload: %v", err)
	}

	// Base64 encode the JSON payload
	encodedPayload := base64.StdEncoding.EncodeToString(payloadJSON)

	return encodedPayload
}
