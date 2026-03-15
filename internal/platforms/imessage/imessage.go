package imessage

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
)

// Platform implements router.Platform for iMessage via BlueBubbles
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	httpClient     *http.Client
	lastTimestamp   int64
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds iMessage/BlueBubbles configuration
type Config struct {
	BlueBubblesURL      string // BlueBubbles server URL
	BlueBubblesPassword string // BlueBubbles server password
	PollInterval        int    // Polling interval in seconds (default: 5)
}

// New creates a new iMessage platform
func New(cfg Config) (*Platform, error) {
	if cfg.BlueBubblesURL == "" {
		return nil, fmt.Errorf("BlueBubbles URL is required")
	}
	if cfg.BlueBubblesPassword == "" {
		return nil, fmt.Errorf("BlueBubbles password is required")
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 5
	}

	return &Platform{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		lastTimestamp: time.Now().UnixMilli(),
	}, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "imessage"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins polling for new iMessages
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	go p.pollLoop()

	log.Printf("[iMessage] Connected to BlueBubbles at %s", p.config.BlueBubblesURL)
	return nil
}

// Stop shuts down the iMessage connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

// Send sends a message via BlueBubbles API
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/message/text?password=%s",
		p.config.BlueBubblesURL, p.config.BlueBubblesPassword)

	payload := map[string]any{
		"chatGuid": channelID,
		"message":  resp.Text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("BlueBubbles API error %d: %s", httpResp.StatusCode, string(respBody))
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

// fetchMessages retrieves new messages from BlueBubbles
func (p *Platform) fetchMessages() {
	url := fmt.Sprintf("%s/api/v1/message?password=%s&after=%d&sort=ASC&limit=50",
		p.config.BlueBubblesURL, p.config.BlueBubblesPassword, p.lastTimestamp)

	req, err := http.NewRequestWithContext(p.ctx, http.MethodGet, url, nil)
	if err != nil {
		if p.ctx.Err() != nil {
			return
		}
		log.Printf("[iMessage] Error creating request: %v", err)
		return
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		if p.ctx.Err() != nil {
			return
		}
		log.Printf("[iMessage] Poll error: %v", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			GUID        string `json:"guid"`
			Text        string `json:"text"`
			IsFromMe    bool   `json:"isFromMe"`
			DateCreated int64  `json:"dateCreated"`
			Handle      struct {
				Address string `json:"address"`
			} `json:"handle"`
			Chats []struct {
				GUID string `json:"guid"`
			} `json:"chats"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[iMessage] Failed to decode response: %v", err)
		return
	}

	for _, msg := range result.Data {
		// Skip messages from self
		if msg.IsFromMe {
			continue
		}
		if msg.Text == "" {
			continue
		}

		// Update timestamp
		if msg.DateCreated > p.lastTimestamp {
			p.lastTimestamp = msg.DateCreated
		}

		chatGUID := ""
		if len(msg.Chats) > 0 {
			chatGUID = msg.Chats[0].GUID
		}

		if p.messageHandler != nil {
			p.messageHandler(router.Message{
				ID:        msg.GUID,
				Platform:  "imessage",
				ChannelID: chatGUID,
				UserID:    msg.Handle.Address,
				Username:  msg.Handle.Address,
				Text:      msg.Text,
				Metadata: map[string]string{
					"address": msg.Handle.Address,
				},
			})
		}
	}
}
