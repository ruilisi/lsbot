package googlechat

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

// Platform implements router.Platform for Google Chat
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	httpClient     *http.Client
	server         *http.Server
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds Google Chat configuration
type Config struct {
	ProjectID       string // Google Cloud project ID
	CredentialsFile string // Path to service account credentials JSON
	WebhookPort     int    // Port for incoming webhooks (default: 8087)
}

// New creates a new Google Chat platform
func New(cfg Config) (*Platform, error) {
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("Google Chat project ID is required")
	}
	if cfg.WebhookPort == 0 {
		cfg.WebhookPort = 8087
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
	return "googlechat"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for Google Chat webhooks
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", p.handleWebhook)

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.config.WebhookPort),
		Handler: mux,
	}

	sentryutil.Go("googlechat webhook server", func() {
		log.Printf("[GoogleChat] Webhook server listening on :%d", p.config.WebhookPort)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[GoogleChat] Server error: %v", err)
		}
	})

	log.Printf("[GoogleChat] Platform started, project: %s", p.config.ProjectID)
	return nil
}

// Stop shuts down the Google Chat connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.server.Shutdown(ctx)
	}
	return nil
}

// Send sends a message via Google Chat API
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" {
		return nil
	}

	// channelID is the space name (spaces/XXXXX)
	url := fmt.Sprintf("https://chat.googleapis.com/v1/%s/messages", channelID)

	payload := map[string]string{
		"text": resp.Text,
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
		return fmt.Errorf("Google Chat API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// handleWebhook processes incoming Google Chat webhook events
func (p *Platform) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var event chatEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// For synchronous responses, we could write a reply here
	w.WriteHeader(http.StatusOK)

	// Only process MESSAGE events
	if event.Type != "MESSAGE" {
		return
	}

	if p.messageHandler != nil {
		p.messageHandler(router.Message{
			ID:        event.Message.Name,
			Platform:  "googlechat",
			ChannelID: event.Space.Name,
			UserID:    event.User.Name,
			Username:  event.User.DisplayName,
			Text:      event.Message.Text,
			Metadata: map[string]string{
				"space_name":   event.Space.Name,
				"space_type":   event.Space.Type,
				"thread_name":  event.Message.Thread.Name,
			},
		})
	}
}

// Google Chat event types
type chatEvent struct {
	Type    string `json:"type"`
	Message struct {
		Name   string `json:"name"`
		Text   string `json:"text"`
		Thread struct {
			Name string `json:"name"`
		} `json:"thread"`
	} `json:"message"`
	User struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"user"`
	Space struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"space"`
}
