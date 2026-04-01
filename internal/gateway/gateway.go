package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ruilisi/lsbot/internal/logger"
	"github.com/ruilisi/lsbot/internal/sentryutil"
)

// MessageType defines the type of gateway message
type MessageType string

const (
	// Client to server
	MsgTypeChat    MessageType = "chat"
	MsgTypeCommand MessageType = "command"
	MsgTypePing    MessageType = "ping"
	MsgTypeAuth    MessageType = "auth"

	// Server to client
	MsgTypeResponse   MessageType = "response"
	MsgTypeEvent      MessageType = "event"
	MsgTypePong       MessageType = "pong"
	MsgTypeError      MessageType = "error"
	MsgTypeAuthResult MessageType = "auth_result"
)

// Message represents a gateway message
type Message struct {
	ID        string          `json:"id"`
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// ChatPayload represents a chat message payload
type ChatPayload struct {
	Text      string `json:"text"`
	SessionID string `json:"session_id,omitempty"`
}

// ResponsePayload represents a response payload
type ResponsePayload struct {
	Text      string `json:"text"`
	SessionID string `json:"session_id,omitempty"`
	Done      bool   `json:"done"`
}

// EventPayload represents an event payload
type EventPayload struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID         string
	conn       *websocket.Conn
	send       chan []byte
	gateway    *Gateway
	sessionID  string
	authorized bool
	metadata   map[string]string
	mu         sync.RWMutex
}

// MessageHandler handles incoming chat messages
type MessageHandler func(ctx context.Context, clientID string, sessionID string, text string) (<-chan ResponsePayload, error)

// Gateway manages WebSocket connections and message routing
type Gateway struct {
	addr        string
	clients     map[string]*Client
	register    chan *Client
	unregister  chan *Client
	broadcast   chan []byte
	handler     MessageHandler
	authTokens  []string // Optional allowed authentication tokens (any one is accepted)
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	upgrader    websocket.Upgrader
}

// Config holds gateway configuration
type Config struct {
	Addr       string   // Address to listen on, e.g., ":28788"
	AuthToken  string   // Single auth token (backward-compat; merged with AuthTokens)
	AuthTokens []string // Multiple allowed auth tokens; any one grants access
}

// New creates a new Gateway
func New(cfg Config) *Gateway {
	if cfg.Addr == "" {
		cfg.Addr = ":28788"
	}

	// Merge single AuthToken into the list for backward compatibility
	tokens := cfg.AuthTokens
	if cfg.AuthToken != "" {
		tokens = append(tokens, cfg.AuthToken)
	}
	// Deduplicate
	seen := make(map[string]struct{}, len(tokens))
	unique := tokens[:0]
	for _, t := range tokens {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			unique = append(unique, t)
		}
	}

	return &Gateway{
		addr:       cfg.Addr,
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		authTokens: unique,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for local development
			},
		},
	}
}

// SetMessageHandler sets the handler for incoming chat messages
func (g *Gateway) SetMessageHandler(handler MessageHandler) {
	g.handler = handler
}

// Start begins the gateway server
func (g *Gateway) Start(ctx context.Context) error {
	g.ctx, g.cancel = context.WithCancel(ctx)

	// Start the hub
	sentryutil.Go("gateway hub run", g.run)

	// HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", g.handleWebSocket)
	mux.HandleFunc("/health", g.handleHealth)
	mux.HandleFunc("/status", g.handleStatus)

	server := &http.Server{
		Addr:    g.addr,
		Handler: mux,
	}

	sentryutil.Go("gateway shutdown watcher", func() {
		<-g.ctx.Done()
		server.Shutdown(context.Background())
	})

	logger.Info("[Gateway] Starting on %s", g.addr)
	return server.ListenAndServe()
}

// Stop shuts down the gateway
func (g *Gateway) Stop() error {
	if g.cancel != nil {
		g.cancel()
	}
	return nil
}

// run handles client registration and broadcasting
func (g *Gateway) run() {
	for {
		select {
		case <-g.ctx.Done():
			return
		case client := <-g.register:
			g.mu.Lock()
			g.clients[client.ID] = client
			g.mu.Unlock()
			logger.Info("[Gateway] Client connected: %s", client.ID)

		case client := <-g.unregister:
			g.mu.Lock()
			if _, ok := g.clients[client.ID]; ok {
				delete(g.clients, client.ID)
				close(client.send)
			}
			g.mu.Unlock()
			logger.Info("[Gateway] Client disconnected: %s", client.ID)

		case message := <-g.broadcast:
			g.mu.RLock()
			for _, client := range g.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(g.clients, client.ID)
				}
			}
			g.mu.RUnlock()
		}
	}
}

// handleWebSocket upgrades HTTP to WebSocket
func (g *Gateway) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("[Gateway] Upgrade error: %v", err)
		return
	}

	client := &Client{
		ID:       generateID(),
		conn:     conn,
		send:     make(chan []byte, 256),
		gateway:  g,
		metadata: make(map[string]string),
	}

	// If no auth tokens are configured, auto-authorize
	if len(g.authTokens) == 0 {
		client.authorized = true
	}

	g.register <- client

	sentryutil.Go("gateway client writePump", client.writePump)
	sentryutil.Go("gateway client readPump", client.readPump)
}

// handleHealth returns health status
func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleStatus returns gateway status
func (g *Gateway) handleStatus(w http.ResponseWriter, r *http.Request) {
	g.mu.RLock()
	clientCount := len(g.clients)
	g.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":       "running",
		"clients":      clientCount,
		"addr":         g.addr,
		"auth_enabled": len(g.authTokens) > 0,
	})
}

// SendToClient sends a message to a specific client
func (g *Gateway) SendToClient(clientID string, msg Message) error {
	g.mu.RLock()
	client, ok := g.clients[clientID]
	g.mu.RUnlock()

	if !ok {
		return fmt.Errorf("client not found: %s", clientID)
	}

	msg.Timestamp = time.Now().UnixMilli()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case client.send <- data:
		return nil
	default:
		return fmt.Errorf("client send buffer full")
	}
}

// Broadcast sends a message to all connected clients
func (g *Gateway) Broadcast(msg Message) {
	msg.Timestamp = time.Now().UnixMilli()
	data, _ := json.Marshal(msg)
	g.broadcast <- data
}

// GetClientCount returns the number of connected clients
func (g *Gateway) GetClientCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.clients)
}

// readPump handles incoming messages from client
func (c *Client) readPump() {
	defer func() {
		c.gateway.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(65536)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		logger.Trace("[Gateway] Received pong from client %s", c.ID)
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Trace("[Gateway] Read error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.sendError("invalid_message", "Failed to parse message")
			continue
		}

		c.handleMessage(msg)
	}
}

// writePump handles outgoing messages to client
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch any queued messages
			n := len(c.send)
			for range n {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			logger.Trace("[Gateway] Sending ping to client %s", c.ID)
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes an incoming message
func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case MsgTypePing:
		c.sendPong()

	case MsgTypeAuth:
		c.handleAuth(msg)

	case MsgTypeChat:
		if !c.authorized && len(c.gateway.authTokens) > 0 {
			c.sendError("unauthorized", "Authentication required")
			return
		}
		c.handleChat(msg)

	case MsgTypeCommand:
		if !c.authorized && len(c.gateway.authTokens) > 0 {
			c.sendError("unauthorized", "Authentication required")
			return
		}
		c.handleCommand(msg)

	default:
		c.sendError("unknown_type", "Unknown message type: "+string(msg.Type))
	}
}

// handleAuth handles authentication
func (c *Client) handleAuth(msg Message) {
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("invalid_payload", "Invalid auth payload")
		return
	}

	c.authorized = slices.Contains(c.gateway.authTokens, payload.Token)
	if c.authorized {
		c.sendAuthResult(true, "")
	} else {
		c.sendAuthResult(false, "Invalid token")
	}
}

// handleChat handles chat messages
func (c *Client) handleChat(msg Message) {
	var payload ChatPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("invalid_payload", "Invalid chat payload")
		return
	}

	if c.gateway.handler == nil {
		c.sendError("no_handler", "No message handler configured")
		return
	}

	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = c.sessionID
	}
	if sessionID == "" {
		sessionID = c.ID
	}
	c.sessionID = sessionID

	// Call the message handler
	respChan, err := c.gateway.handler(c.gateway.ctx, c.ID, sessionID, payload.Text)
	if err != nil {
		c.sendError("handler_error", err.Error())
		return
	}

	// Stream responses
	sentryutil.Go("gateway stream responses", func() {
		for resp := range respChan {
			c.sendResponse(msg.ID, resp)
		}
	})
}

// handleCommand handles command messages
func (c *Client) handleCommand(msg Message) {
	var payload struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("invalid_payload", "Invalid command payload")
		return
	}

	// Handle built-in commands
	switch payload.Command {
	case "status":
		c.sendEvent("status", map[string]any{
			"client_id":  c.ID,
			"session_id": c.sessionID,
			"authorized": c.authorized,
		})
	case "clear":
		c.sessionID = ""
		c.sendEvent("cleared", nil)
	default:
		c.sendError("unknown_command", "Unknown command: "+payload.Command)
	}
}

// Helper methods for sending messages

func (c *Client) sendPong() {
	c.sendMessage(Message{Type: MsgTypePong})
}

func (c *Client) sendError(code, message string) {
	payload, _ := json.Marshal(map[string]string{"code": code, "message": message})
	c.sendMessage(Message{Type: MsgTypeError, Payload: payload})
}

func (c *Client) sendAuthResult(success bool, message string) {
	payload, _ := json.Marshal(map[string]any{"success": success, "message": message})
	c.sendMessage(Message{Type: MsgTypeAuthResult, Payload: payload})
}

func (c *Client) sendResponse(requestID string, resp ResponsePayload) {
	payload, _ := json.Marshal(resp)
	c.sendMessage(Message{ID: requestID, Type: MsgTypeResponse, Payload: payload})
}

func (c *Client) sendEvent(event string, data any) {
	dataJSON, _ := json.Marshal(data)
	payload, _ := json.Marshal(EventPayload{Event: event, Data: dataJSON})
	c.sendMessage(Message{Type: MsgTypeEvent, Payload: payload})
}

func (c *Client) sendMessage(msg Message) {
	msg.Timestamp = time.Now().UnixMilli()
	if msg.ID == "" {
		msg.ID = generateID()
	}
	data, _ := json.Marshal(msg)
	defer func() {
		if r := recover(); r != nil {
			logger.Trace("[Gateway] Send on closed channel for client %s (client disconnected)", c.ID)
		}
	}()
	select {
	case c.send <- data:
	default:
		logger.Trace("[Gateway] Send buffer full for client %s", c.ID)
	}
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
