package matrix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/sentryutil"
)

// Platform implements router.Platform for Matrix (Element)
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	httpClient     *http.Client
	syncToken      string
	txnID          int64
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds Matrix configuration
type Config struct {
	HomeserverURL string // Matrix homeserver URL (e.g., https://matrix.org)
	UserID        string // Bot user ID (e.g., @bot:matrix.org)
	AccessToken   string // Access token for the bot account
}

// New creates a new Matrix platform
func New(cfg Config) (*Platform, error) {
	if cfg.HomeserverURL == "" {
		return nil, fmt.Errorf("Matrix homeserver URL is required")
	}
	if cfg.UserID == "" {
		return nil, fmt.Errorf("Matrix user ID is required")
	}
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("Matrix access token is required")
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
	return "matrix"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins the Matrix sync loop
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// Do an initial sync to get the sync token (ignore old messages)
	if err := p.initialSync(ctx); err != nil {
		return fmt.Errorf("initial sync failed: %w", err)
	}

	sentryutil.Go("matrix syncLoop", p.syncLoop)

	log.Printf("[Matrix] Connected as %s to %s", p.config.UserID, p.config.HomeserverURL)
	return nil
}

// Stop shuts down the Matrix connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

// Send sends a message to a Matrix room
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" {
		return nil
	}

	p.txnID++
	txn := strconv.FormatInt(p.txnID, 10)

	url := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
		p.config.HomeserverURL, channelID, txn)

	payload := map[string]string{
		"msgtype": "m.text",
		"body":    resp.Text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.AccessToken)

	httpResp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("Matrix API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// initialSync performs an initial sync to get the since token
func (p *Platform) initialSync(ctx context.Context) error {
	url := fmt.Sprintf("%s/_matrix/client/v3/sync?timeout=0", p.config.HomeserverURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.config.AccessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var syncResp syncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return fmt.Errorf("failed to decode sync response: %w", err)
	}

	p.syncToken = syncResp.NextBatch
	return nil
}

// syncLoop continuously polls for new Matrix events
func (p *Platform) syncLoop() {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		url := fmt.Sprintf("%s/_matrix/client/v3/sync?since=%s&timeout=30000",
			p.config.HomeserverURL, p.syncToken)

		req, err := http.NewRequestWithContext(p.ctx, http.MethodGet, url, nil)
		if err != nil {
			if p.ctx.Err() != nil {
				return
			}
			log.Printf("[Matrix] Error creating sync request: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+p.config.AccessToken)

		resp, err := p.httpClient.Do(req)
		if err != nil {
			if p.ctx.Err() != nil {
				return
			}
			log.Printf("[Matrix] Sync error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var syncResp syncResponse
		if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
			resp.Body.Close()
			log.Printf("[Matrix] Failed to decode sync: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		resp.Body.Close()

		p.syncToken = syncResp.NextBatch

		// Process room events
		for roomID, room := range syncResp.Rooms.Join {
			for _, event := range room.Timeline.Events {
				p.processEvent(roomID, event)
			}
		}
	}
}

// processEvent handles a single Matrix event
func (p *Platform) processEvent(roomID string, event matrixEvent) {
	// Only process text messages from other users
	if event.Type != "m.room.message" {
		return
	}
	if event.Sender == p.config.UserID {
		return
	}

	content, ok := event.Content.(map[string]any)
	if !ok {
		return
	}
	msgtype, _ := content["msgtype"].(string)
	if msgtype != "m.text" {
		return
	}
	body, _ := content["body"].(string)
	if body == "" {
		return
	}

	if p.messageHandler != nil {
		p.messageHandler(router.Message{
			ID:        event.EventID,
			Platform:  "matrix",
			ChannelID: roomID,
			UserID:    event.Sender,
			Username:  event.Sender,
			Text:      body,
			Metadata: map[string]string{
				"room_id": roomID,
			},
		})
	}
}

// Matrix sync response types
type syncResponse struct {
	NextBatch string `json:"next_batch"`
	Rooms     struct {
		Join map[string]struct {
			Timeline struct {
				Events []matrixEvent `json:"events"`
			} `json:"timeline"`
		} `json:"join"`
	} `json:"rooms"`
}

type matrixEvent struct {
	Type    string `json:"type"`
	EventID string `json:"event_id"`
	Sender  string `json:"sender"`
	Content any    `json:"content"`
}
