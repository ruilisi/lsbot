package nextcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/sentryutil"
)

// Platform implements router.Platform for Nextcloud Talk
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	httpClient     *http.Client
	lastKnownID    int
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds Nextcloud Talk configuration
type Config struct {
	ServerURL    string // Nextcloud server URL
	Username     string // Bot username
	Password     string // Bot password or app password
	RoomToken    string // Talk room token
	PollInterval int    // Polling interval in seconds (default: 3)
}

// New creates a new Nextcloud Talk platform
func New(cfg Config) (*Platform, error) {
	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("Nextcloud server URL is required")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("Nextcloud username is required")
	}
	if cfg.Password == "" {
		return nil, fmt.Errorf("Nextcloud password is required")
	}
	if cfg.RoomToken == "" {
		return nil, fmt.Errorf("Nextcloud room token is required")
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 3
	}

	return &Platform{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "nextcloud"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins polling for Nextcloud Talk messages
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// Get initial last known message ID
	if err := p.getLastMessageID(); err != nil {
		log.Printf("[Nextcloud] Warning: failed to get last message ID: %v", err)
	}

	sentryutil.Go("nextcloud pollLoop", p.pollLoop)

	log.Printf("[Nextcloud] Connected to %s, room: %s", p.config.ServerURL, p.config.RoomToken)
	return nil
}

// Stop shuts down the Nextcloud Talk connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

// Send sends a message to a Nextcloud Talk room
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" {
		return nil
	}

	url := fmt.Sprintf("%s/ocs/v2.php/apps/spreed/api/v1/chat/%s",
		p.config.ServerURL, channelID)

	payload := map[string]string{
		"message": resp.Text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	p.setHeaders(req)

	httpResp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("Nextcloud API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// setHeaders sets common headers for Nextcloud API requests
func (p *Platform) setHeaders(req *http.Request) {
	req.SetBasicAuth(p.config.Username, p.config.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OCS-APIRequest", "true")
	req.Header.Set("Accept", "application/json")
}

// getLastMessageID fetches the most recent message to set baseline
func (p *Platform) getLastMessageID() error {
	url := fmt.Sprintf("%s/ocs/v2.php/apps/spreed/api/v1/chat/%s?lookIntoFuture=0&limit=1",
		p.config.ServerURL, p.config.RoomToken)

	req, err := http.NewRequestWithContext(p.ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	p.setHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result ocsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if len(result.OCS.Data) > 0 {
		p.lastKnownID = result.OCS.Data[0].ID
	}

	return nil
}

// pollLoop continuously polls for new messages
func (p *Platform) pollLoop() {
	ticker := time.NewTicker(time.Duration(p.config.PollInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.fetchMessages()
		}
	}
}

// fetchMessages retrieves new messages from Nextcloud Talk
func (p *Platform) fetchMessages() {
	url := fmt.Sprintf("%s/ocs/v2.php/apps/spreed/api/v1/chat/%s?lookIntoFuture=1&limit=50&lastKnownMessageId=%d",
		p.config.ServerURL, p.config.RoomToken, p.lastKnownID)

	req, err := http.NewRequestWithContext(p.ctx, http.MethodGet, url, nil)
	if err != nil {
		if p.ctx.Err() != nil {
			return
		}
		log.Printf("[Nextcloud] Error creating request: %v", err)
		return
	}
	p.setHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		if p.ctx.Err() != nil {
			return
		}
		log.Printf("[Nextcloud] Poll error: %v", err)
		return
	}
	defer resp.Body.Close()

	// 304 means no new messages
	if resp.StatusCode == http.StatusNotModified {
		return
	}

	var result ocsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[Nextcloud] Failed to decode response: %v", err)
		return
	}

	for _, msg := range result.OCS.Data {
		// Update last known ID
		if msg.ID > p.lastKnownID {
			p.lastKnownID = msg.ID
		}

		// Skip system messages and bot's own messages
		if msg.SystemMessage != "" {
			continue
		}
		if msg.ActorID == p.config.Username {
			continue
		}
		if msg.Message == "" {
			continue
		}

		if p.messageHandler != nil {
			p.messageHandler(router.Message{
				ID:        fmt.Sprintf("%d", msg.ID),
				Platform:  "nextcloud",
				ChannelID: msg.Token,
				UserID:    msg.ActorID,
				Username:  msg.ActorDisplayName,
				Text:      msg.Message,
				Metadata: map[string]string{
					"room_token": msg.Token,
					"actor_type": msg.ActorType,
				},
			})
		}
	}
}

// Nextcloud OCS response types
type ocsResponse struct {
	OCS struct {
		Data []chatMessage `json:"data"`
	} `json:"ocs"`
}

type chatMessage struct {
	ID               int    `json:"id"`
	Token            string `json:"token"`
	ActorType        string `json:"actorType"`
	ActorID          string `json:"actorId"`
	ActorDisplayName string `json:"actorDisplayName"`
	Message          string `json:"message"`
	SystemMessage    string `json:"systemMessage"`
}
