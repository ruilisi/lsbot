package line

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/sentryutil"
)

// Platform implements router.Platform for LINE Messaging API
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	httpClient     *http.Client
	server         *http.Server
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds LINE configuration
type Config struct {
	ChannelSecret string // LINE Channel Secret for signature verification
	ChannelToken  string // LINE Channel Access Token
	WebhookPort   int    // Port for incoming webhooks (default: 8085)
}

// New creates a new LINE platform
func New(cfg Config) (*Platform, error) {
	if cfg.ChannelSecret == "" {
		return nil, fmt.Errorf("LINE channel secret is required")
	}
	if cfg.ChannelToken == "" {
		return nil, fmt.Errorf("LINE channel token is required")
	}
	if cfg.WebhookPort == 0 {
		cfg.WebhookPort = 8085
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
	return "line"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for LINE webhooks
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", p.handleWebhook)

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.config.WebhookPort),
		Handler: mux,
	}

	sentryutil.Go("line webhook server", func() {
		log.Printf("[LINE] Webhook server listening on :%d", p.config.WebhookPort)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[LINE] Server error: %v", err)
		}
	})

	log.Printf("[LINE] Platform started")
	return nil
}

// Stop shuts down the LINE connection
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

// Send sends a message via LINE Push API
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" {
		return nil
	}

	payload := map[string]any{
		"to": channelID,
		"messages": []map[string]string{
			{
				"type": "text",
				"text": resp.Text,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.line.me/v2/bot/message/push", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.ChannelToken)

	httpResp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("LINE API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// sendReply sends a reply message using a reply token
func (p *Platform) sendReply(ctx context.Context, replyToken string, text string) error {
	payload := map[string]any{
		"replyToken": replyToken,
		"messages": []map[string]string{
			{
				"type": "text",
				"text": text,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal reply: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.line.me/v2/bot/message/reply", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.ChannelToken)

	httpResp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send reply: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("LINE reply API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// verifySignature verifies the LINE webhook signature
func (p *Platform) verifySignature(body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(p.config.ChannelSecret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// handleWebhook processes incoming LINE webhook requests
func (p *Platform) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify signature
	signature := r.Header.Get("X-Line-Signature")
	if !p.verifySignature(body, signature) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var webhook webhookPayload
	if err := json.Unmarshal(body, &webhook); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)

	// Process events
	for _, event := range webhook.Events {
		if event.Type != "message" || event.Message.Type != "text" {
			continue
		}
		if p.messageHandler != nil {
			p.messageHandler(router.Message{
				ID:        event.Message.ID,
				Platform:  "line",
				ChannelID: event.Source.UserID,
				UserID:    event.Source.UserID,
				Username:  event.Source.UserID,
				Text:      event.Message.Text,
				Metadata: map[string]string{
					"reply_token": event.ReplyToken,
					"source_type": event.Source.Type,
					"group_id":    event.Source.GroupID,
				},
			})
		}
	}
}

// LINE webhook payload types
type webhookPayload struct {
	Events []struct {
		Type       string `json:"type"`
		ReplyToken string `json:"replyToken"`
		Source     struct {
			Type    string `json:"type"`
			UserID  string `json:"userId"`
			GroupID string `json:"groupId"`
		} `json:"source"`
		Message struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"message"`
	} `json:"events"`
}
