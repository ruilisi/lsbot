package mattermost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ruilisi/lsbot/internal/router"
)

// Platform implements router.Platform for Mattermost
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	httpClient     *http.Client
	wsConn         *websocket.Conn
	botUserID      string
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds Mattermost configuration
type Config struct {
	ServerURL string // Mattermost server URL (e.g., https://mattermost.example.com)
	Token     string // Personal access token or bot token
	TeamName  string // Team name for context
}

// New creates a new Mattermost platform
func New(cfg Config) (*Platform, error) {
	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("Mattermost server URL is required")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("Mattermost token is required")
	}

	return &Platform{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "mattermost"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins the Mattermost WebSocket connection
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// Get bot user info
	if err := p.getBotUser(); err != nil {
		return fmt.Errorf("failed to get bot user: %w", err)
	}

	// Connect WebSocket
	if err := p.connectWebSocket(); err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	go p.listenWebSocket()

	log.Printf("[Mattermost] Connected to %s as user %s", p.config.ServerURL, p.botUserID)
	return nil
}

// Stop shuts down the Mattermost connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.wsConn != nil {
		return p.wsConn.Close()
	}
	return nil
}

// Send sends a message via Mattermost REST API
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" {
		return nil
	}

	apiURL := fmt.Sprintf("%s/api/v4/posts", strings.TrimRight(p.config.ServerURL, "/"))

	payload := map[string]string{
		"channel_id": channelID,
		"message":    resp.Text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.Token)

	httpResp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("Mattermost API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// getBotUser retrieves the bot's user ID
func (p *Platform) getBotUser() error {
	apiURL := fmt.Sprintf("%s/api/v4/users/me", strings.TrimRight(p.config.ServerURL, "/"))

	req, err := http.NewRequestWithContext(p.ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.config.Token)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var user struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return err
	}

	p.botUserID = user.ID
	return nil
}

// connectWebSocket establishes the WebSocket connection
func (p *Platform) connectWebSocket() error {
	serverURL := strings.TrimRight(p.config.ServerURL, "/")
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return err
	}

	scheme := "wss"
	if parsed.Scheme == "http" {
		scheme = "ws"
	}
	wsURL := fmt.Sprintf("%s://%s/api/v4/websocket", scheme, parsed.Host)

	header := http.Header{}
	header.Set("Authorization", "Bearer "+p.config.Token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return fmt.Errorf("WebSocket dial failed: %w", err)
	}

	p.wsConn = conn
	return nil
}

// listenWebSocket processes incoming WebSocket events
func (p *Platform) listenWebSocket() {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		_, msg, err := p.wsConn.ReadMessage()
		if err != nil {
			if p.ctx.Err() != nil {
				return
			}
			log.Printf("[Mattermost] WebSocket error: %v, reconnecting...", err)
			time.Sleep(5 * time.Second)
			if err := p.connectWebSocket(); err != nil {
				log.Printf("[Mattermost] Reconnect failed: %v", err)
			}
			continue
		}

		var event wsEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			continue
		}

		if event.Event == "posted" {
			p.handlePosted(event)
		}
	}
}

// handlePosted processes a new post event
func (p *Platform) handlePosted(event wsEvent) {
	postJSON, ok := event.Data["post"].(string)
	if !ok {
		return
	}

	var post struct {
		ID        string `json:"id"`
		ChannelID string `json:"channel_id"`
		UserID    string `json:"user_id"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal([]byte(postJSON), &post); err != nil {
		return
	}

	// Ignore own messages
	if post.UserID == p.botUserID {
		return
	}

	if p.messageHandler != nil {
		username, _ := event.Data["sender_name"].(string)
		if username == "" {
			username = post.UserID
		}

		p.messageHandler(router.Message{
			ID:        post.ID,
			Platform:  "mattermost",
			ChannelID: post.ChannelID,
			UserID:    post.UserID,
			Username:  username,
			Text:      post.Message,
			Metadata: map[string]string{
				"team_name": p.config.TeamName,
			},
		})
	}
}

// WebSocket event type
type wsEvent struct {
	Event string         `json:"event"`
	Data  map[string]any `json:"data"`
}
