package signal

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

// Platform implements router.Platform for Signal via signal-cli REST API
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	httpClient     *http.Client
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds Signal configuration
type Config struct {
	APIURL       string // signal-cli REST API URL
	PhoneNumber  string // Registered phone number
	PollInterval int    // Polling interval in seconds (default: 3)
}

// New creates a new Signal platform
func New(cfg Config) (*Platform, error) {
	if cfg.APIURL == "" {
		return nil, fmt.Errorf("Signal API URL is required")
	}
	if cfg.PhoneNumber == "" {
		return nil, fmt.Errorf("Signal phone number is required")
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 3
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
	return "signal"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins polling for Signal messages
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	go p.pollLoop()

	log.Printf("[Signal] Connected to %s, phone: %s", p.config.APIURL, p.config.PhoneNumber)
	return nil
}

// Stop shuts down the Signal connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

// Send sends a message via signal-cli REST API
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" {
		return nil
	}

	url := fmt.Sprintf("%s/v2/send", p.config.APIURL)

	payload := map[string]any{
		"message":    resp.Text,
		"number":     p.config.PhoneNumber,
		"recipients": []string{channelID},
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
		return fmt.Errorf("Signal API error %d: %s", httpResp.StatusCode, string(respBody))
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

// fetchMessages retrieves new messages from signal-cli
func (p *Platform) fetchMessages() {
	url := fmt.Sprintf("%s/v1/receive/%s", p.config.APIURL, p.config.PhoneNumber)

	req, err := http.NewRequestWithContext(p.ctx, http.MethodGet, url, nil)
	if err != nil {
		if p.ctx.Err() != nil {
			return
		}
		log.Printf("[Signal] Error creating request: %v", err)
		return
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		if p.ctx.Err() != nil {
			return
		}
		log.Printf("[Signal] Poll error: %v", err)
		return
	}
	defer resp.Body.Close()

	var messages []struct {
		Envelope struct {
			Source        string `json:"source"`
			SourceName   string `json:"sourceName"`
			Timestamp    int64  `json:"timestamp"`
			DataMessage  *struct {
				Message   string `json:"message"`
				Timestamp int64  `json:"timestamp"`
				GroupInfo *struct {
					GroupID string `json:"groupId"`
				} `json:"groupInfo"`
			} `json:"dataMessage"`
		} `json:"envelope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		log.Printf("[Signal] Failed to decode response: %v", err)
		return
	}

	for _, msg := range messages {
		if msg.Envelope.DataMessage == nil || msg.Envelope.DataMessage.Message == "" {
			continue
		}

		channelID := msg.Envelope.Source
		if msg.Envelope.DataMessage.GroupInfo != nil {
			channelID = msg.Envelope.DataMessage.GroupInfo.GroupID
		}

		username := msg.Envelope.SourceName
		if username == "" {
			username = msg.Envelope.Source
		}

		if p.messageHandler != nil {
			p.messageHandler(router.Message{
				ID:        fmt.Sprintf("%d", msg.Envelope.Timestamp),
				Platform:  "signal",
				ChannelID: channelID,
				UserID:    msg.Envelope.Source,
				Username:  username,
				Text:      msg.Envelope.DataMessage.Message,
				Metadata: map[string]string{
					"source": msg.Envelope.Source,
				},
			})
		}
	}
}
