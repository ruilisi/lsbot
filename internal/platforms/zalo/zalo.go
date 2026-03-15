package zalo

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/sentryutil"
)

// Platform implements router.Platform for Zalo Official Account API
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	httpClient     *http.Client
	server         *http.Server
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds Zalo configuration
type Config struct {
	AppID       string // Zalo App ID
	SecretKey   string // Zalo App Secret Key
	AccessToken string // Zalo OA Access Token
	WebhookPort int    // Port for incoming webhooks (default: 8088)
}

// New creates a new Zalo platform
func New(cfg Config) (*Platform, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("Zalo App ID is required")
	}
	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("Zalo Secret Key is required")
	}
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("Zalo Access Token is required")
	}
	if cfg.WebhookPort == 0 {
		cfg.WebhookPort = 8088
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
	return "zalo"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for Zalo webhooks
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", p.handleWebhook)

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.config.WebhookPort),
		Handler: mux,
	}

	sentryutil.Go("zalo webhook server", func() {
		log.Printf("[Zalo] Webhook server listening on :%d", p.config.WebhookPort)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Zalo] Server error: %v", err)
		}
	})

	log.Printf("[Zalo] Platform started, app_id: %s", p.config.AppID)
	return nil
}

// Stop shuts down the Zalo connection
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

// Send sends a message via Zalo OA API
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" {
		return nil
	}

	payload := map[string]any{
		"recipient": map[string]string{
			"user_id": channelID,
		},
		"message": map[string]string{
			"text": resp.Text,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://openapi.zalo.me/v3.0/oa/message/cs", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token", p.config.AccessToken)

	httpResp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("Zalo API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// verifySignature verifies the Zalo webhook signature
func (p *Platform) verifySignature(body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(p.config.SecretKey))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// handleWebhook processes incoming Zalo webhook events
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

	// Verify signature if present
	signature := r.Header.Get("X-ZEvent-Signature")
	if signature != "" && !p.verifySignature(body, signature) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var event zaloEvent
	if err := json.Unmarshal(body, &event); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)

	// Only process user_send_text events
	if event.EventName != "user_send_text" {
		return
	}

	if p.messageHandler != nil {
		p.messageHandler(router.Message{
			ID:        event.MsgID,
			Platform:  "zalo",
			ChannelID: event.Sender.ID,
			UserID:    event.Sender.ID,
			Username:  event.Sender.ID,
			Text:      event.Message.Text,
			Metadata: map[string]string{
				"app_id": event.AppID,
			},
		})
	}
}

// Zalo event types
type zaloEvent struct {
	EventName string `json:"event_name"`
	AppID     string `json:"app_id"`
	MsgID     string `json:"msg_id"`
	Sender    struct {
		ID string `json:"id"`
	} `json:"sender"`
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
}
