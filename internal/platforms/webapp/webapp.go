package webapp

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pltanton/lingti-bot/internal/router"
	"github.com/pltanton/lingti-bot/internal/sentryutil"
)

//go:embed static/index.html
var indexHTML []byte

// Config holds webapp configuration.
type Config struct {
	Port  int    // HTTP port, e.g. 8080
	Token string // Optional Bearer token auth (empty = no auth)
}

type inMsg struct {
	Type      string `json:"type"`       // "message" or "clear"
	SessionID string `json:"session_id"` // UUID
	Text      string `json:"text,omitempty"`
}

type outMsg struct {
	Type      string `json:"type"`            // "response", "progress", "error"
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
	Done      bool   `json:"done,omitempty"`
}

type conn struct {
	ws  *websocket.Conn
	mu  sync.Mutex
	id  string // connID
}

func (c *conn) send(msg outMsg) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ws.WriteMessage(websocket.TextMessage, data)
}

// Platform implements router.Platform for the web chat UI.
type Platform struct {
	cfg            Config
	server         *http.Server
	upgrader       websocket.Upgrader
	clients        sync.Map // connID → *conn
	sessions       sync.Map // sessionID → connID
	messageHandler func(router.Message)
	ctx            context.Context
	cancel         context.CancelFunc
}

// New creates a new webapp Platform.
func New(cfg Config) (*Platform, error) {
	if cfg.Port <= 0 {
		return nil, fmt.Errorf("webapp: port must be > 0")
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &Platform{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	return p, nil
}

func (p *Platform) Name() string { return "webapp" }

func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

func (p *Platform) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.serveIndex)
	mux.HandleFunc("/ws", p.serveWS)

	// Try ports starting from cfg.Port, incrementing until one is free.
	port := p.cfg.Port
	const maxTries = 100
	var ln net.Listener
	var err error
	for range maxTries {
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			break
		}
		log.Printf("webapp: port %d in use, trying %d", port, port+1)
		port++
	}
	if err != nil {
		return fmt.Errorf("webapp: could not bind to any port in range %d-%d", p.cfg.Port, port)
	}

	p.cfg.Port = port // update so Stop/logs reflect actual port
	p.server = &http.Server{Handler: mux}

	sentryutil.Go("webapp server", func() {
		if err := p.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("webapp: server error: %v", err)
		}
	})

	log.Printf("webapp: listening on http://localhost:%d", port)
	return nil
}

func (p *Platform) Stop() error {
	p.cancel()
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.server.Shutdown(ctx)
	}
	return nil
}

func (p *Platform) Send(ctx context.Context, sessionID string, resp router.Response) error {
	connIDVal, ok := p.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("webapp: no connection for session %s", sessionID)
	}
	connVal, ok := p.clients.Load(connIDVal.(string))
	if !ok {
		return fmt.Errorf("webapp: connection gone for session %s", sessionID)
	}
	c := connVal.(*conn)

	// Determine message type from metadata
	msgType := "response"
	done := true
	if resp.Metadata != nil {
		if t, ok := resp.Metadata["type"]; ok {
			msgType = t
		}
		if resp.Metadata["done"] == "false" {
			done = false
		}
	}

	return c.send(outMsg{
		Type:      msgType,
		SessionID: sessionID,
		Text:      resp.Text,
		Done:      done,
	})
}

func (p *Platform) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (p *Platform) checkAuth(r *http.Request) bool {
	if p.cfg.Token == "" {
		return true
	}
	// Check query param
	if r.URL.Query().Get("token") == p.cfg.Token {
		return true
	}
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") && strings.TrimPrefix(auth, "Bearer ") == p.cfg.Token {
		return true
	}
	return false
}

func (p *Platform) serveWS(w http.ResponseWriter, r *http.Request) {
	if !p.checkAuth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ws, err := p.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("webapp: upgrade error: %v", err)
		return
	}

	connID := fmt.Sprintf("%d", time.Now().UnixNano())
	c := &conn{ws: ws, id: connID}
	p.clients.Store(connID, c)
	defer func() {
		p.clients.Delete(connID)
		ws.Close()
	}()

	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			break
		}

		var msg inMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		if msg.SessionID == "" {
			continue
		}

		// Map this session to this connection
		p.sessions.Store(msg.SessionID, connID)

		switch msg.Type {
		case "message":
			if p.messageHandler != nil && msg.Text != "" {
				go p.messageHandler(router.Message{
					Platform:  "webapp",
					ChannelID: msg.SessionID,
					UserID:    "user",
					Username:  "User",
					Text:      msg.Text,
				})
			}
		case "clear":
			// Send acknowledgment
			c.send(outMsg{ //nolint
				Type:      "response",
				SessionID: msg.SessionID,
				Text:      "Conversation cleared.",
				Done:      true,
			})
		}
	}
}
