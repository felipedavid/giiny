package imvu

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketClient represents a WebSocket client for IMVU
type WebSocketClient struct {
	conn              *websocket.Conn
	url               string
	headers           http.Header
	mu                sync.Mutex
	done              chan struct{}
	messageHandler    func(message []byte)
	pingInterval      time.Duration
	lastPongReceived  time.Time
	reconnectInterval time.Duration
	isConnected       bool
}

// WebSocketMessage represents a message sent or received over WebSocket
type WebSocketMessage struct {
	Record string `json:"record"`
}

// WebSocketMetadata represents metadata in a connect message
type WebSocketMetadata struct {
	Record string `json:"record"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

// WebSocketConnectMessage represents a connect message to be sent over WebSocket
type WebSocketConnectMessage struct {
	Record   string              `json:"record"`
	UserID   string              `json:"user_id"`
	Cookie   string              `json:"cookie"`
	Metadata []WebSocketMetadata `json:"metadata"`
	OpID     int                 `json:"op_id"`
}

// WebSocketSubscription represents a subscription in a subscribe message
type WebSocketSubscription struct {
	Record string `json:"record"`
	Name   string `json:"name"`
	OpID   int    `json:"op_id"`
}

// WebSocketSubscribeMessage represents a subscribe message to be sent over WebSocket
type WebSocketSubscribeMessage struct {
	Record            string                  `json:"record"`
	QueuesWithResults []WebSocketSubscription `json:"queues_with_results"`
}

// WebSocketSendMessageMessage represents a send message message to be sent over WebSocket
type WebSocketSendMessageMessage struct {
	Record  string `json:"record"`
	Queue   string `json:"queue"`
	Mount   string `json:"mount"`
	Message string `json:"message"`
	OpID    int    `json:"op_id"`
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(url string, headers http.Header) *WebSocketClient {
	return &WebSocketClient{
		url:               url,
		headers:           headers,
		done:              make(chan struct{}),
		pingInterval:      16 * time.Second,
		reconnectInterval: 5 * time.Second,
	}
}

// Connect establishes a WebSocket connection
func (wsc *WebSocketClient) Connect() error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if wsc.isConnected {
		return nil
	}

	// Log connection attempt
	log.Printf("Connecting to WebSocket server at %s", wsc.url)

	dialer := websocket.Dialer{
		HandshakeTimeout: 45 * time.Second,
		// Add subprotocols if needed
		// Subprotocols: []string{"chat", "superchat"},
	}

	// Log headers being sent
	log.Printf("WebSocket connection headers: %v", wsc.headers)

	conn, resp, err := dialer.Dial(wsc.url, wsc.headers)
	if err != nil {
		if resp != nil {
			log.Printf("WebSocket handshake failed with status: %d", resp.StatusCode)
			// Log response headers for debugging
			log.Printf("Response headers: %v", resp.Header)
			return fmt.Errorf("websocket connection failed with status %d: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("websocket connection failed: %w", err)
	}

	// Log successful connection
	log.Printf("WebSocket connection established successfully")

	wsc.conn = conn
	wsc.isConnected = true
	wsc.lastPongReceived = time.Now()

	// Set up pong handler
	wsc.conn.SetPongHandler(func(string) error {
		log.Printf("Received pong from server")
		wsc.mu.Lock()
		wsc.lastPongReceived = time.Now()
		wsc.mu.Unlock()
		return nil
	})

	// Set up close handler
	wsc.conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("WebSocket connection closed by server: code=%d, text=%s", code, text)
		wsc.mu.Lock()
		wsc.isConnected = false
		wsc.mu.Unlock()
		return nil
	})

	// Start reader goroutine
	go wsc.readMessages()

	// Start ping goroutine
	go wsc.pingPeriodically()

	return nil
}

// Close closes the WebSocket connection
func (wsc *WebSocketClient) Close() error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if !wsc.isConnected {
		return nil
	}

	// Signal done to stop goroutines
	close(wsc.done)

	// Close the connection
	err := wsc.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return fmt.Errorf("error sending close message: %w", err)
	}

	err = wsc.conn.Close()
	if err != nil {
		return fmt.Errorf("error closing connection: %w", err)
	}

	wsc.isConnected = false
	return nil
}

// SetMessageHandler sets a handler function for incoming messages
func (wsc *WebSocketClient) SetMessageHandler(handler func(message []byte)) {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()
	wsc.messageHandler = handler
}

// SendMessage sends a message over the WebSocket connection
func (wsc *WebSocketClient) SendMessage(message interface{}) error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if !wsc.isConnected {
		return fmt.Errorf("websocket not connected")
	}

	return wsc.conn.WriteJSON(message)
}

// SendConnect sends a connect message to the server
func (wsc *WebSocketClient) SendConnect(userID, cookie string) error {
	connectMessage := WebSocketConnectMessage{
		Record: "msg_c2g_connect",
		UserID: userID,
		Cookie: base64.StdEncoding.EncodeToString([]byte(cookie)),
		Metadata: []WebSocketMetadata{
			{
				Record: "metadata",
				Key:    "app",
				Value:  "aW12dV9uZXh0",
			},
			{
				Record: "metadata",
				Key:    "platform_type",
				Value:  "Ymln",
			},
		},
		OpID: 0,
	}
	return wsc.SendMessage(connectMessage)
}

// SendSubscribe sends a subscribe message to the server
func (wsc *WebSocketClient) SendSubscribe(name string, opID int) error {
	subscribeMessage := WebSocketSubscribeMessage{
		Record: "msg_c2g_subscribe",
		QueuesWithResults: []WebSocketSubscription{
			{
				Record: "subscription",
				Name:   name,
				OpID:   opID,
			},
		},
	}
	return wsc.SendMessage(subscribeMessage)
}

// SendChatMessage sends a chat message to the server
func (wsc *WebSocketClient) SendChatMessage(queue, mount, message string, opID int) error {
	sendMessageMessage := WebSocketSendMessageMessage{
		Record:  "msg_c2g_send_message",
		Queue:   queue,
		Mount:   mount,
		Message: message,
		OpID:    opID,
	}
	return wsc.SendMessage(sendMessageMessage)
}

// SendPing sends a ping message to the server
func (wsc *WebSocketClient) SendPing() error {
	pingMessage := WebSocketMessage{
		Record: "msg_c2g_ping",
	}
	return wsc.SendMessage(pingMessage)
}

// readMessages reads messages from the WebSocket connection
func (wsc *WebSocketClient) readMessages() {
	defer func() {
		wsc.mu.Lock()
		wsc.isConnected = false
		wsc.mu.Unlock()

		// Log when reader exits
		log.Printf("WebSocket reader routine exiting")
	}()

	for {
		select {
		case <-wsc.done:
			log.Printf("WebSocket reader received done signal")
			return
		default:
			messageType, message, err := wsc.conn.ReadMessage()
			if err != nil {
				// Check if it's a normal closure
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Printf("WebSocket closed normally: %v", err)
				} else if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Printf("WebSocket closed unexpectedly: %v", err)
				} else {
					log.Printf("Error reading message: %v", err)
				}

				// Don't attempt to reconnect if we're intentionally closing
				select {
				case <-wsc.done:
					return
				default:
					// Schedule reconnection
					go func() {
						time.Sleep(wsc.reconnectInterval)
						if err := wsc.Connect(); err != nil {
							log.Printf("Failed to reconnect: %v", err)
						} else {
							log.Printf("Successfully reconnected")
						}
					}()
					return
				}
			}

			// Log message type and content
			log.Printf("Received message type: %d, content: %s", messageType, string(message))

			// Handle the message
			wsc.handleMessage(message)
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (wsc *WebSocketClient) handleMessage(message []byte) {
	// Try to parse as a WebSocketMessage
	var wsMessage WebSocketMessage
	if err := json.Unmarshal(message, &wsMessage); err == nil {
		// Check if it's a pong message
		if wsMessage.Record == "msg_g2c_pong" {
			log.Printf("Received pong message: %s", string(message))
			wsc.mu.Lock()
			wsc.lastPongReceived = time.Now()
			wsc.mu.Unlock()
			return
		} else {
			log.Printf("Received message with record type: %s", wsMessage.Record)
		}
	} else {
		log.Printf("Failed to parse message as WebSocketMessage: %v", err)
	}

	// Pass to message handler if set
	wsc.mu.Lock()
	handler := wsc.messageHandler
	wsc.mu.Unlock()

	if handler != nil {
		handler(message)
	} else {
		log.Printf("No message handler set for message: %s", string(message))
	}
}

// pingPeriodically sends ping messages at regular intervals
func (wsc *WebSocketClient) pingPeriodically() {
	ticker := time.NewTicker(wsc.pingInterval)
	defer ticker.Stop()

	log.Printf("Starting ping routine with interval %v", wsc.pingInterval)

	for {
		select {
		case <-wsc.done:
			log.Printf("Ping routine received done signal")
			return
		case <-ticker.C:
			log.Printf("Sending ping message")
			if err := wsc.SendPing(); err != nil {
				log.Printf("Error sending ping: %v", err)
				wsc.reconnect()
				return
			}

			// Check if we've received a pong recently
			wsc.mu.Lock()
			elapsed := time.Since(wsc.lastPongReceived)
			wsc.mu.Unlock()

			log.Printf("Time since last pong: %v", elapsed)
			if elapsed > 2*wsc.pingInterval {
				log.Printf("No pong received in %v, reconnecting", elapsed)
				wsc.reconnect()
				return
			}
		}
	}
}

// reconnect attempts to reconnect to the WebSocket server
func (wsc *WebSocketClient) reconnect() {
	wsc.mu.Lock()
	if !wsc.isConnected {
		wsc.mu.Unlock()
		return
	}

	// Close the current connection
	wsc.conn.Close()
	wsc.isConnected = false
	wsc.mu.Unlock()

	// Create a new done channel
	wsc.mu.Lock()
	wsc.done = make(chan struct{})
	wsc.mu.Unlock()

	// Wait before reconnecting
	time.Sleep(wsc.reconnectInterval)

	// Try to reconnect
	for {
		err := wsc.Connect()
		if err == nil {
			log.Println("Successfully reconnected to WebSocket server")
			return
		}

		log.Printf("Failed to reconnect: %v, retrying in %v", err, wsc.reconnectInterval)
		time.Sleep(wsc.reconnectInterval)
	}
}

// IsConnected returns whether the client is currently connected
func (wsc *WebSocketClient) IsConnected() bool {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()
	return wsc.isConnected
}
