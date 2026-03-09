package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pltanton/lingti-bot/internal/router"
	"github.com/pltanton/lingti-bot/internal/sentryutil"
)

// Platform implements router.Platform for WhatsApp Business Cloud API
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	httpClient     *http.Client
	server         *http.Server
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds WhatsApp Business configuration
type Config struct {
	PhoneNumberID string // WhatsApp Business Phone Number ID
	AccessToken   string // Meta Graph API access token
	VerifyToken   string // Webhook verification token
	WebhookPort   int    // Port for incoming webhooks (default: 8084)
}

// New creates a new WhatsApp platform
func New(cfg Config) (*Platform, error) {
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("WhatsApp access token is required")
	}
	if cfg.PhoneNumberID == "" {
		return nil, fmt.Errorf("WhatsApp phone number ID is required")
	}
	if cfg.VerifyToken == "" {
		cfg.VerifyToken = "lingti-bot-verify"
	}
	if cfg.WebhookPort == 0 {
		cfg.WebhookPort = 8084
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
	return "whatsapp"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for WhatsApp webhooks
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", p.handleWebhook)

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.config.WebhookPort),
		Handler: mux,
	}

	sentryutil.Go("whatsapp webhook server", func() {
		log.Printf("[WhatsApp] Webhook server listening on :%d", p.config.WebhookPort)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[WhatsApp] Server error: %v", err)
		}
	})

	log.Printf("[WhatsApp] Connected, phone_number_id: %s", p.config.PhoneNumberID)
	return nil
}

// Stop shuts down the WhatsApp connection
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

// Send sends a message via WhatsApp Business API
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" {
		return nil
	}

	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                channelID,
		"type":              "text",
		"text":              map[string]string{"body": resp.Text},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages", p.config.PhoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
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
		return fmt.Errorf("WhatsApp API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// handleWebhook processes incoming WhatsApp webhook requests
func (p *Platform) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Webhook verification (GET)
	if r.Method == http.MethodGet {
		mode := r.URL.Query().Get("hub.mode")
		token := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")

		if mode == "subscribe" && token == p.config.VerifyToken {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Message webhook (POST)
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var webhook webhookPayload
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)

	// Process messages
	for _, entry := range webhook.Entry {
		for _, change := range entry.Changes {
			if change.Value.Messages == nil {
				continue
			}
			for _, msg := range change.Value.Messages {
				if msg.Type != "text" {
					continue
				}
				if p.messageHandler != nil {
					// Get contact name
					username := msg.From
					for _, contact := range change.Value.Contacts {
						if contact.WaID == msg.From {
							username = contact.Profile.Name
							break
						}
					}

					p.messageHandler(router.Message{
						ID:        msg.ID,
						Platform:  "whatsapp",
						ChannelID: msg.From,
						UserID:    msg.From,
						Username:  username,
						Text:      msg.Text.Body,
						Metadata: map[string]string{
							"phone_number_id": p.config.PhoneNumberID,
						},
					})
				}
			}
		}
	}
}

// WhatsApp webhook payload types
type webhookPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Messages []struct {
					ID   string `json:"id"`
					From string `json:"from"`
					Type string `json:"type"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
				Contacts []struct {
					WaID    string `json:"wa_id"`
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
				} `json:"contacts"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}
