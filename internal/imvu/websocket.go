package imvu

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// State represents the state of the WebSocket connection
type State int

const (
	StateClosed State = iota
	StateConnecting
	StateAuthenticating
	StateAuthenticated
	StateWaiting
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateConnecting:
		return "CONNECTING"
	case StateAuthenticating:
		return "AUTHENTICATING"
	case StateAuthenticated:
		return "AUTHENTICATED"
	case StateWaiting:
		return "WAITING"
	default:
		return "UNKNOWN"
	}
}

// Config holds the configuration for the WebSocketClient
type Config struct {
	URL                   string
	Headers               http.Header
	UserID                string
	SessionID             string
	Metadata              map[string]string
	OpID                  *OperationID
	PingInterval          time.Duration
	ServerTimeoutInterval time.Duration
	ReconnectIntervals    []time.Duration
	OnStateChange         func(state State, nextConnectTime *time.Time)
	OnMessage             func(message map[string]any)
	OnPreReconnect        func(callback func(err error, newConfig *Config))
}

// WebSocketClient represents a WebSocket client for IMVU
type WebSocketClient struct {
	config                    Config
	conn                      *websocket.Conn
	mu                        sync.Mutex
	state                     State
	done                      chan struct{}
	connectRetryTimer         *time.Timer
	pingTimer                 *time.Timer
	serverTimeoutTimer        *time.Timer
	lastMessageTime           time.Time
	connectRetryIntervalIndex int
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(config Config) *WebSocketClient {
	// Set default values from the JS code
	if config.PingInterval == 0 {
		config.PingInterval = 15 * time.Second
	}
	if config.ServerTimeoutInterval == 0 {
		config.ServerTimeoutInterval = 60 * time.Second
	}
	if len(config.ReconnectIntervals) == 0 {
		config.ReconnectIntervals = []time.Duration{
			5 * time.Second,
			15 * time.Second,
			45 * time.Second,
			90 * time.Second,
			180 * time.Second,
		}
	}
	if config.OnPreReconnect == nil {
		config.OnPreReconnect = func(callback func(err error, newConfig *Config)) {
			callback(nil, nil)
		}
	}

	client := &WebSocketClient{
		config: config,
	}
	client.setState(StateClosed, nil)
	return client
}

// Connect starts the connection process.
func (c *WebSocketClient) Connect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StateWaiting || c.state == StateClosed {
		c.clearConnectRetryTimer()
		go c.run()
	}
}

// Close disconnects the client.
func (c *WebSocketClient) Close() {
	log.Println("Disconnecting from IMQ")
	c.mu.Lock()
	c.reset()
	c.disconnect()
	c.setState(StateClosed, nil)
	c.mu.Unlock()
}

func (c *WebSocketClient) run() {
	c.mu.Lock()
	if c.state != StateWaiting && c.state != StateClosed {
		c.mu.Unlock()
		return
	}

	c.setState(StateConnecting, nil)
	log.Printf("Connecting to IMQ via '%s' as user '%s'", c.config.URL, c.config.UserID)

	dialer := websocket.Dialer{
		HandshakeTimeout: 45 * time.Second,
	}

	conn, _, err := dialer.Dial(c.config.URL, c.config.Headers)
	if err != nil {
		log.Printf("IMQ WebSocket dial error: %v", err)
		c.mu.Unlock()
		c.onDisconnected()
		return
	}

	c.conn = conn
	c.done = make(chan struct{})
	c.lastMessageTime = time.Now()
	c.scheduleServerTimeout()
	c.mu.Unlock()

	c.onOpen()

	// Reader loop
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			// Check if the error is due to a closed connection
			select {
			case <-c.done:
				// We closed the connection intentionally
				log.Println("WebSocket reader stopping.")
			default:
				// Unexpected close
				log.Printf("IMQ WebSocket read error: %v", err)
				c.onDisconnected()
			}
			return
		}
		c.onMessage(message)
	}
}

func (c *WebSocketClient) setState(state State, nextConnectTime *time.Time) {
	if c.state == state {
		return
	}
	c.state = state
	log.Printf("IMQ State changed to: %s", state)
	if c.config.OnStateChange != nil {
		c.config.OnStateChange(state, nextConnectTime)
	}
}

func (c *WebSocketClient) onOpen() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setState(StateAuthenticating, nil)
	metadata := []map[string]string{}
	for k, v := range c.config.Metadata {
		metadata = append(metadata, map[string]string{
			"record": "metadata",
			"key":    k,
			"value":  base64.StdEncoding.EncodeToString([]byte(v)),
		})
	}

	connectMsg := map[string]any{
		"record":   "msg_c2g_connect",
		"user_id":  c.config.UserID,
		"cookie":   base64.StdEncoding.EncodeToString([]byte(c.config.SessionID)),
		"metadata": metadata,
		"op_id":    c.config.OpID.GetNew(),
	}
	c.sendRaw(connectMsg)
}

func (c *WebSocketClient) onMessage(data []byte) {
	c.mu.Lock()
	c.scheduleServerTimeout()
	c.lastMessageTime = time.Now()
	c.mu.Unlock()

	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to decode IMQ message: %v", err)
		return
	}

	msgType, ok := msg["record"].(string)
	if !ok {
		log.Println("IMQ message has no 'record' field")
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StateAuthenticating {
		if msgType == "msg_g2c_result" {
			if status, ok := msg["status"].(float64); ok && status == 0 {
				log.Println("IMQ authenticated")
				c.onAuthenticated()
				c.sendOpenFloodgates()
			} else {
				errorMsg, _ := msg["error_message"].(string)
				log.Printf("Failed to authenticate with IMQ: %s", errorMsg)
				c.disconnect()
				go c.onDisconnected()
			}
		} else {
			log.Printf("Unexpected message type during IMQ authentication: %s", msgType)
		}
	} else if msgType != "msg_g2c_pong" {
		if c.config.OnMessage != nil {
			// To avoid race conditions, we pass the message to the handler in a new goroutine.
			go c.config.OnMessage(msg)
		}
	}
}

func (c *WebSocketClient) onDisconnected() {
	c.mu.Lock()
	c.disconnect()
	log.Println("Connection to IMQ closed")
	c.reconnect()
	c.mu.Unlock()
}

func (c *WebSocketClient) onAuthenticated() {
	c.setState(StateAuthenticated, nil)
	c.reset()
}

func (c *WebSocketClient) reset() {
	c.connectRetryIntervalIndex = 0
}

func (c *WebSocketClient) disconnect() {
	c.clearConnectRetryTimer()
	c.clearPingTimer()
	c.clearServerTimer()
	if c.conn != nil {
		if c.done != nil {
			close(c.done)
			c.done = nil
		}
		c.conn.Close()
		c.conn = nil
	}
}

func (c *WebSocketClient) clearConnectRetryTimer() {
	if c.connectRetryTimer != nil {
		c.connectRetryTimer.Stop()
		c.connectRetryTimer = nil
	}
}

func (c *WebSocketClient) reconnect() {
	interval := c.config.ReconnectIntervals[c.connectRetryIntervalIndex]
	log.Printf("Reconnecting to IMQ in %v", interval)

	nextConnectTime := time.Now().Add(interval)
	c.setState(StateWaiting, &nextConnectTime)

	c.connectRetryTimer = time.AfterFunc(interval, func() {
		c.config.OnPreReconnect(func(err error, newConfig *Config) {
			if err != nil {
				log.Printf("Error in IMQ pre-reconnect callback: %v", err)
				c.mu.Lock()
				c.reconnect() // Try again
				c.mu.Unlock()
				return
			}
			c.mu.Lock()
			if newConfig != nil {
				c.config.SessionID = newConfig.SessionID
				c.config.UserID = newConfig.UserID
			}
			c.mu.Unlock()
			c.Connect()
		})
	})

	c.connectRetryIntervalIndex++
	if c.connectRetryIntervalIndex >= len(c.config.ReconnectIntervals) {
		c.connectRetryIntervalIndex = 0 // Reset to the beginning
	}
}

func (c *WebSocketClient) sendOpenFloodgates() {
	c.send("msg_c2g_open_floodgates", map[string]any{})
}

// Send allows sending a message with a specific record type and payload.
func (c *WebSocketClient) Send(record string, payload map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.send(record, payload)
}

// Internal send function, assumes lock is held.
func (c *WebSocketClient) send(record string, payload map[string]any) {
	if c.state != StateAuthenticated {
		log.Printf("Cannot send message '%s', not authenticated. State: %s", record, c.state)
		return
	}
	c.schedulePing()
	payload["record"] = record
	c.sendRaw(payload)
}

// sendRaw sends a raw message without adding the record or checking state.
func (c *WebSocketClient) sendRaw(message any) {
	if c.conn == nil {
		log.Println("Cannot send raw message, connection is nil.")
		return
	}
	if err := c.conn.WriteJSON(message); err != nil {
		log.Printf("Error sending IMQ message: %v", err)
	}
}

func (c *WebSocketClient) scheduleServerTimeout() {
	c.clearServerTimer()
	c.serverTimeoutTimer = time.AfterFunc(c.config.ServerTimeoutInterval, c.onServerTimeout)
}

func (c *WebSocketClient) clearServerTimer() {
	if c.serverTimeoutTimer != nil {
		c.serverTimeoutTimer.Stop()
		c.serverTimeoutTimer = nil
	}
}

func (c *WebSocketClient) onServerTimeout() {
	log.Printf("No message from IMQ server for %v, disconnecting", c.config.ServerTimeoutInterval)
	c.onDisconnected()
}

func (c *WebSocketClient) schedulePing() {
	c.clearPingTimer()
	c.pingTimer = time.AfterFunc(c.config.PingInterval, c.sendPing)
}

func (c *WebSocketClient) clearPingTimer() {
	if c.pingTimer != nil {
		c.pingTimer.Stop()
		c.pingTimer = nil
	}
}

func (c *WebSocketClient) sendPing() {
	c.mu.Lock()
	defer c.mu.Unlock()
	// The JS version sends a ping via `_send`, which schedules the *next* ping.
	// We will do the same.
	c.send("msg_c2g_ping", map[string]any{})
}

// GetState returns the current state of the client.
func (c *WebSocketClient) GetState() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// The following message structs are kept for reference and potential use in typed message handlers,
// but the core logic now uses map[string]any for flexibility, matching the JS implementation.

// WebSocketMessage represents a message sent or received over WebSocket
type WebSocketMessage struct {
	Record string `json:"record"`
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
	Message any    `json:"message"` // Can be a string or a more complex object
	OpID    int    `json:"op_id"`
}
